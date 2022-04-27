package x

import (
	"encoding/json"
	"fmt"
	"net"
)

var _ ISessionMaker = (*FakeSessionMaker)(nil)
var _ ISession = (*FakeSession)(nil)

type FakeSessionMaker struct {
	cnt uint32
}

type FakeSession struct {
	id uint32
}

func NewFakeSessionMaker() *FakeSessionMaker {
	return &FakeSessionMaker{}
}

func (f *FakeSessionMaker) MakeSession() (ISession, error) {
	f.cnt++
	return &FakeSession{id: f.cnt}, nil
}

func (f *FakeSession) ID() uint32 {
	return f.id
}

func (f *FakeSession) CommandCb(conn net.Conn, allResponse *AllRequest) error {
	fmt.Println("client addr =", conn.RemoteAddr().String(), ":")
	buf, _ := json.MarshalIndent(allResponse, "", " ")
	fmt.Println(string(buf))

	cmd := allResponse.Cmd
	var rspBuf []byte
	if cmd == CmdStart {
		startRsp := StartResponse{
			Cmd:       cmd,
			SessionID: f.id,
			UDPPort:   UDPPort,
		}
		rspBuf, _ = json.Marshal(&startRsp)
	} else if cmd == CmdEnd {
		endRsp := EndResponse{
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
func (f *FakeSession) DataCb(data []byte, seq uint32) error {
	fmt.Println("get seq=", seq, ", data len=", len(data))
	return nil
}
