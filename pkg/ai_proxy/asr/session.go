package asr

import (
	"encoding/json"
	"fmt"
	"github.com/realzhangm/xaux/pkg/x"
	"net"
)

var _ x.ISession = (*Session)(nil)
var _ x.ISessionMaker = (*SessionMaker)(nil)

type SessionMaker struct {
	cnt uint32
}

func NewSessionMaker() *SessionMaker {
	return &SessionMaker{}
}

func (s *SessionMaker) MakeSession() (x.ISession, error) {
	s.cnt++
	return &Session{id: s.cnt}, nil
}

type Session struct {
	id uint32
}

func (f *Session) ID() uint32 {
	return f.id
}

func (f *Session) CommandCb(conn net.Conn, allResponse *x.AllRequest) error {
	fmt.Println("client addr =", conn.RemoteAddr().String(), ":")
	buf, _ := json.MarshalIndent(allResponse, "", " ")
	fmt.Println(string(buf))

	cmd := allResponse.Cmd
	var rspBuf []byte
	if cmd == x.CmdStart {
		startRsp := x.StartResponse{
			Cmd:       cmd,
			SessionID: f.id,
			UDPPort:   x.UDPPort,
		}
		rspBuf, _ = json.Marshal(&startRsp)
	} else if cmd == x.CmdEnd {
		endRsp := x.EndResponse{
			Cmd: cmd,
			Msg: "session end!",
		}
		rspBuf, _ = json.Marshal(&endRsp)
	}
	_, err := conn.Write(rspBuf)
	if err != nil {
		panic(err)
	}
	return nil
}
func (f *Session) DataCb(data []byte, seq uint32) error {
	fmt.Println("get seq=", seq, ", data len=", len(data))
	return nil
}
