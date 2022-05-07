package aliyun

import (
	"encoding/json"
	"errors"
	"fmt"
	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
	"os"
	"sync"
	"time"
)

import (
	"github.com/realzhangm/xaux/pkg/resample"
	"github.com/realzhangm/xaux/pkg/x"
)

var _ x.ISession = (*Session)(nil)
var _ x.ISessionMaker = (*SessionMaker)(nil)

var (
	AKID   = "xxxx"
	AKKEY  = "xxxx"
	APPKEY = "xxxx"

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

func (n *NLSStatus) setOpened() bool {
	ok := false
	n.Lock()
	if n.status&NLSStatusOpened == 0 {
		ok = true
		n.status = NLSStatusOpened
	}
	n.Unlock()
	return ok
}

func (n *NLSStatus) setClosed() bool {
	ok := false
	n.Lock()
	if n.status&NLSStatusClosed == 0 {
		ok = true
		n.status = NLSStatusClosed
	}
	n.Unlock()
	return ok
}

type SessionMaker struct {
	cnt uint32
}

func init() {
	AKID = os.Getenv("ALI_AKID")
	AKKEY = os.Getenv("ALI_AKKEY")
	APPKEY = os.Getenv("ALI_APPKEY")
}

func SetSecureKey(akID, akKey, appKey string) {
	AKID = akID
	AKKEY = akKey
	APPKEY = appKey
}

func NewSessionMaker() *SessionMaker {
	return &SessionMaker{}
}

func (s *SessionMaker) MakeSession(r x.IResponse) (x.ISession, error) {
	sess := Session{id: s.cnt, netRsp: r.(*x.TCPResponse)}
	err := sess.newNLS()
	if err != nil {
		return nil, err
	}

	s.cnt++
	return &sess, nil
}

// Session TODO,
// 两个状态，一个是 nls 的状态，一个是本身的会话状态
type Session struct {
	id          uint32
	netRsp      *x.TCPResponse
	_st         *nls.SpeechTranscription
	startConfig x.StartConfig
	nlsStatus   NLSStatus
	status      int32
	mu          sync.Mutex
}

func (s *Session) ID() uint32 {
	return s.id
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
	status := s.status
	return status
}

func (s *Session) getSpeechTrans() error {
	return s.newNLS()
}

func (s *Session) newNLS() error {
	if ok := s.nlsStatus.setOpened(); !ok {
		return nil
	}
	config, err := nls.NewConnectionConfigWithAKInfoDefault(WsUrlBeijing, APPKEY, AKID, AKKEY)
	if err != nil {
		return err
	}

	s._st, err = nls.NewSpeechTranscription(config, nil,
		onTaskFailed, onStarted,
		onSentenceBegin, onSentenceEnd, onResultChanged,
		onCompleted, onClose, s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) closeNLS() {
	ok := s.nlsStatus.setClosed()
	if ok {
		s._st.Shutdown()
	}
}

func waitReady(ch chan bool) error {
	select {
	case done := <-ch:
		{
			if !done {
				fmt.Println("Wait failed")
				return errors.New("wait failed")
			}
			fmt.Println("Wait done")
		}
	case <-time.After(20 * time.Second):
		{
			fmt.Println("Wait timeout")
			return errors.New("wait timeout")
		}
	}
	return nil
}

// 打开识别会话
func (s *Session) startNLSSpeechTrans() error {
	// new NLS
	if err := s.getSpeechTrans(); err != nil {
		return err
	}
	fmt.Println("startNLSSpeechTrans")
	param := nls.DefaultSpeechTranscriptionParam()
	param.Format = "wav"
	exMap := make(map[string]interface{})
	exMap["disfluency"] = true
	exMap["enable_words"] = false
	exMap["enable_semantic_sentence_detection"] = false
	ready, err := s._st.Start(param, exMap)
	if err != nil {
		s.closeNLS()
		return err
	}
	return waitReady(ready)
}

// 结束识别会话
func (s *Session) stopNLSSpeechTrans() error {
	ready, err := s._st.Stop()
	if err != nil {
		s.closeNLS()
		fmt.Println("xx error :", err.Error())
		return err
	}

	err = waitReady(ready)
	if err != nil {
		s.closeNLS()
		fmt.Println("xxxx error :", err.Error())
		return err
	}
	return nil
}

// 客户端发送 Start
func (s *Session) onCmdStart() error {
	err := s.setStatus(StatusSessionStarted)
	if err != nil {
		return err
	}

	err = s.startNLSSpeechTrans()
	if err != nil {
		return err
	}
	return nil
}

// 客户端发送 End
func (s *Session) onCmdEnd() error {
	err := s.setStatus(StatusSessionFinished)
	if err != nil {
		return err
	}
	return s.stopNLSSpeechTrans()
}

// CloseAll 暂时是先关掉NLS连接
func (s *Session) CloseAll() {
	err := s.setStatus(StatusSessionFinished)
	if err == nil {
		s.stopNLSSpeechTrans()
	}
	s.setStatus(StatusNone)
	s.closeNLS()
	return
}

func (s *Session) CommandCb(allRequest *x.AllRequest) error {
	//fmt.Println("client addr =", conn.RemoteAddr().String(), ":")
	//buf, _ := json.MarshalIndent(allRequest, "", " ")
	//fmt.Println(string(buf))
	var rspBuf []byte
	var err error = nil

	switch allRequest.Cmd {
	case x.CmdStart:
		var startRsp x.StartResponse
		err = s.onCmdStart()
		if err != nil {
			startRsp = x.StartResponse{
				Type:  x.TypeRspStart,
				Error: x.Error{Msg: err.Error()},
			}
		} else {
			startRsp = x.StartResponse{
				Type:      x.TypeRspStart,
				SessionID: s.id,
				UDPPort:   x.UDPPort,
			}
		}
		s.startConfig = allRequest.Config
		rspBuf, _ = json.Marshal(&startRsp)
	case x.CmdEnd:
		endRsp := x.EndResponse{}
		err = s.onCmdEnd()
		if err != nil {
			endRsp.Error = x.Error{Msg: err.Error()}
		}
		endRsp.Type = x.TypeRspEnd
		rspBuf, _ = json.Marshal(&endRsp)
	}

	_, err = s.netRsp.Write(rspBuf)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) DataCb(data []byte, seq uint32) {
	if s.getStatus() != StatusSessionStarted {
		fmt.Println("s.status = ", s.getStatus())
		return
	}

	var buf16k []byte

	// 这里是 48K
	if s.startConfig.SampleRate == 48000 {
		if resample.R48kTO16k != nil {
			buf16k, _ = resample.R48kTO16k(data)
		} else {
			panic("resample not support")
		}
	} else if s.startConfig.SampleRate == 16000 {
		buf16k = data[0:]
	} else {
		panic(fmt.Sprintf("not support SampleRate, %d", s.startConfig.SampleRate))
	}

	s._st.SendAudioData(buf16k)
}
