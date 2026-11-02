package aliyun

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
	"github.com/realzhangm/xaux/pkg/resample"
	"github.com/realzhangm/xaux/pkg/x"
)

var _ x.ISession = (*Session)(nil)
var _ x.ISessionMaker = (*SessionMaker)(nil)

var (
	AKID   = os.Getenv("ALI_AKID")
	AKKEY  = os.Getenv("ALI_AKKEY")
	APPKEY = os.Getenv("ALI_APPKEY")

	WsUrlBeijing = "wss://nls-gateway-cn-beijing.aliyuncs.com/ws/v1"
)

var (
	ErrAliAsrSession   = errors.New("asr session")
	ErrAlreadyStarted  = fmt.Errorf("%w, %s", ErrAliAsrSession, "StatusAlreadyStarted")
	ErrAlreadyFinished = fmt.Errorf("%w, %s", ErrAliAsrSession, "ErrAlreadyFinished")
)

const (
	NLSStatusNone            = 0
	NLSStatusOpened          = 1
	NLSStatusClosed          = 2
	NLSStatusSessionStarted  = 4 | NLSStatusOpened
	NLSStatusSessionFinished = 8 | NLSStatusOpened
)

const (
	StatusNone = iota
	StatusSessionStarted
	StatusSessionFinished
)

type NLSStatus struct {
	sync.Mutex
	status uint32
}

func (n *NLSStatus) reset() {
	n.Lock()
	defer n.Unlock()
	n.status = NLSStatusNone
}

func (n *NLSStatus) setOpened() bool {
	n.Lock()
	defer n.Unlock()
	if n.status&NLSStatusOpened != 0 {
		return false
	}
	n.status = NLSStatusOpened
	return true
}

func (n *NLSStatus) setClosed() bool {
	n.Lock()
	defer n.Unlock()
	if n.status&NLSStatusClosed != 0 {
		return false
	}
	n.status = NLSStatusClosed
	return true
}

type SessionMaker struct {
	cnt uint32
}

func SetSecureKey(akID, akKey, appKey string) {
	AKID = akID
	AKKEY = akKey
	APPKEY = appKey
}

func NewSessionMaker() *SessionMaker {
	return &SessionMaker{}
}

func (s *SessionMaker) MakeSession(response x.IResponse) (x.ISession, error) {
	if response == nil {
		return nil, errors.New("aliyun ASR response writer is nil")
	}
	sess := &Session{
		id:     atomic.AddUint32(&s.cnt, 1) - 1,
		netRsp: response,
	}
	if err := sess.newNLS(); err != nil {
		return nil, err
	}
	return sess, nil
}

type Session struct {
	id                  uint32
	netRsp              x.IResponse
	speechTranscription *nls.SpeechTranscription
	startConfig         x.StartConfig
	nlsStatus           NLSStatus
	status              int32
	recorder            *x.AudioRecorder
	mu                  sync.Mutex
}

func (s *Session) ID() uint32 {
	return s.id
}

func (s *Session) SetAudioRecorder(recorder *x.AudioRecorder) {
	s.mu.Lock()
	s.recorder = recorder
	s.mu.Unlock()
}

func (s *Session) audioRecorder() *x.AudioRecorder {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.recorder
}

func (s *Session) setStatus(newStatus int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch newStatus {
	case StatusSessionStarted:
		if s.status == StatusSessionStarted {
			return ErrAlreadyStarted
		}
	case StatusSessionFinished:
		if s.status != StatusSessionStarted {
			return ErrAlreadyFinished
		}
	}
	s.status = newStatus
	return nil
}

func (s *Session) getStatus() int32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Session) newNLS() error {
	if !s.nlsStatus.setOpened() {
		return nil
	}
	config, err := nls.NewConnectionConfigWithAKInfoDefault(WsUrlBeijing, APPKEY, AKID, AKKEY)
	if err != nil {
		s.nlsStatus.reset()
		return err
	}

	s.speechTranscription, err = nls.NewSpeechTranscription(config, nil,
		onTaskFailed, onStarted,
		onSentenceBegin, onSentenceEnd, onResultChanged,
		onCompleted, onClose, s)
	if err != nil {
		s.nlsStatus.reset()
		return err
	}
	return nil
}

func (s *Session) closeNLS() {
	if s.nlsStatus.setClosed() && s.speechTranscription != nil {
		s.speechTranscription.Shutdown()
	}
}

func waitReady(ch <-chan bool) error {
	select {
	case done := <-ch:
		if !done {
			return errors.New("wait failed")
		}
	case <-time.After(20 * time.Second):
		return errors.New("wait timeout")
	}
	return nil
}

func (s *Session) startNLSSpeechTrans() error {
	if err := s.newNLS(); err != nil {
		return err
	}
	param := nls.DefaultSpeechTranscriptionParam()
	param.Format = "pcm"
	extra := map[string]interface{}{
		"disfluency":                         true,
		"enable_words":                       false,
		"enable_semantic_sentence_detection": false,
	}
	ready, err := s.speechTranscription.Start(param, extra)
	if err != nil {
		s.closeNLS()
		return err
	}
	if err := waitReady(ready); err != nil {
		s.closeNLS()
		return err
	}
	return nil
}

func (s *Session) stopNLSSpeechTrans() error {
	ready, err := s.speechTranscription.Stop()
	if err != nil {
		s.closeNLS()
		return err
	}
	if err := waitReady(ready); err != nil {
		s.closeNLS()
		return err
	}
	return nil
}

func (s *Session) onCmdStart() error {
	if s.getStatus() == StatusSessionStarted {
		return ErrAlreadyStarted
	}
	if err := s.startNLSSpeechTrans(); err != nil {
		return err
	}
	return s.setStatus(StatusSessionStarted)
}

func (s *Session) onCmdEnd() error {
	if err := s.setStatus(StatusSessionFinished); err != nil {
		return err
	}
	return s.stopNLSSpeechTrans()
}

func (s *Session) CloseAll() {
	if err := s.setStatus(StatusSessionFinished); err == nil {
		_ = s.stopNLSSpeechTrans()
	}
	_ = s.setStatus(StatusNone)
	s.closeNLS()
}

func (s *Session) CommandCb(request *x.AllRequest) error {
	var response interface{}
	switch request.Cmd {
	case x.CmdStart:
		s.startConfig = request.Config
		startResponse := x.StartResponse{Type: x.TypeRspStart}
		if err := s.onCmdStart(); err != nil {
			startResponse.Error = x.Error{Msg: err.Error()}
		} else {
			startResponse.SessionID = s.id
			startResponse.UDPPort = x.UDPPort
		}
		response = &startResponse
	case x.CmdEnd:
		endResponse := x.EndResponse{Type: x.TypeRspEnd}
		if err := s.onCmdEnd(); err != nil {
			endResponse.Error = x.Error{Msg: err.Error()}
		}
		response = &endResponse
	default:
		return fmt.Errorf("unsupported command %q", request.Cmd)
	}
	return s.writeJSON(response)
}

func (s *Session) DataCb(data []byte, _ uint32) {
	if status := s.getStatus(); status != StatusSessionStarted {
		log.Printf("aliyun session %d is not started: status=%d", s.id, status)
		return
	}

	var pcm16k []byte
	switch s.startConfig.SampleRate {
	case 48000:
		if resample.R48kTO16k == nil {
			log.Printf("aliyun session %d: resample plugin is unavailable", s.id)
			return
		}
		var err error
		pcm16k, err = resample.R48kTO16k(data)
		if err != nil {
			log.Printf("aliyun session %d resample error: %v", s.id, err)
			return
		}
	case 16000:
		pcm16k = data
	default:
		log.Printf("aliyun session %d: unsupported sample rate %d", s.id, s.startConfig.SampleRate)
		return
	}

	if s.speechTranscription == nil {
		log.Printf("aliyun session %d: speech transcription is unavailable", s.id)
		return
	}
	if err := s.speechTranscription.SendAudioData(pcm16k); err != nil {
		log.Printf("aliyun session %d send error: %v", s.id, err)
		return
	}
	if recorder := s.audioRecorder(); recorder != nil {
		recorder.WriteASR(pcm16k)
	}
}

func (s *Session) writeJSON(value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.netRsp.Write(data)
	return err
}
