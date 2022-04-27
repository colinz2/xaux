package aliyun

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

import (
	"github.com/aliyun/alibabacloud-nls-go-sdk"
	"github.com/realzhangm/xaux/pkg/resample"
	"github.com/realzhangm/xaux/pkg/x"
)

var _ x.ISession = (*Session)(nil)
var _ x.ISessionMaker = (*SessionMaker)(nil)

var (
	AKID   = "xxxx"
	AKKEY  = "xxxx"
	APPKEY = "xxxx"
)

type SessionMaker struct {
	cnt uint32
}

func init() {
	AKID = os.Getenv("AKID")
	AKKEY = os.Getenv("AKKEY")
	APPKEY = os.Getenv("APPKEY")
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
	var err error
	sess := Session{id: s.cnt, netRsp: r.(*x.TCPResponse)}

	config, err := nls.NewConnectionConfigWithAKInfoDefault(nls.DEFAULT_URL, APPKEY, AKID, AKKEY)
	if err != nil {
		return nil, err
	}

	sess.st, err = nls.NewSpeechTranscription(config, nil,
		onTaskFailed, onStarted,
		onSentenceBegin, onSentenceEnd, onResultChanged,
		onCompleted, onClose, &sess)
	if err != nil {
		return nil, err
	}

	s.cnt++
	return &sess, err
}

type Session struct {
	id     uint32
	netRsp *x.TCPResponse
	st     *nls.SpeechTranscription
}

func (s *Session) ID() uint32 {
	return s.id
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

func (s *Session) onStart() error {
	param := nls.DefaultSpeechTranscriptionParam()
	exMap := make(map[string]interface{})
	exMap["disfluency"] = true
	exMap["enable_words"] = true
	ready, err := s.st.Start(param, exMap)
	if err != nil {
		s.st.Shutdown()
		return err
	}
	return waitReady(ready)
}

func (s *Session) onEnd() error {
	ready, err := s.st.Stop()
	if err != nil {
		s.st.Shutdown()
		return err
	}

	err = waitReady(ready)
	if err != nil {
		s.st.Shutdown()
		return err
	}
	s.st.Shutdown()
	return nil
}

func (s *Session) CommandCb(allRequest *x.AllRequest) error {
	//fmt.Println("client addr =", conn.RemoteAddr().String(), ":")
	buf, _ := json.MarshalIndent(allRequest, "", " ")
	fmt.Println(string(buf))
	var rspBuf []byte
	var err error = nil

	switch allRequest.Cmd {
	case x.CmdStart:
		var startRsp x.StartResponse
		err = s.onStart()
		if err != nil {
			startRsp = x.StartResponse{
				Cmd:   allRequest.Cmd,
				Error: x.Error{Msg: err.Error()},
			}
		} else {
			startRsp = x.StartResponse{
				Cmd:       allRequest.Cmd,
				SessionID: s.id,
				UDPPort:   x.UDPPort,
			}
		}
		rspBuf, _ = json.Marshal(&startRsp)
	case x.CmdEnd:
		var endRsp x.EndResponse
		err = s.onEnd()
		if err != nil {
			endRsp = x.EndResponse{
				Cmd:   allRequest.Cmd,
				Error: x.Error{Msg: err.Error()},
			}
		} else {
			endRsp = x.EndResponse{
				Cmd: allRequest.Cmd,
				Msg: "session end!",
			}
		}
		rspBuf, _ = json.Marshal(&endRsp)
	}

	_, err = s.netRsp.Write(rspBuf)
	if err != nil {
		return err
	}
	return nil
}
func (s *Session) DataCb(data []byte, seq uint32) error {
	var buf16k []byte
	var err error
	if resample.R48kTO16k != nil {
		buf16k, err = resample.R48kTO16k(data)
	} else {
		panic("resample not support")
	}

	err = s.st.SendAudioData(buf16k)
	if err != nil {
		return err
	}
	return err
}
