package x

import (
	"encoding/json"
	"fmt"
)

var _ ISessionMaker = (*FakeSessionMaker)(nil)
var _ ISession = (*FakeSession)(nil)

type FakeSessionMaker struct {
	cnt uint32
}

type FakeSession struct {
	id          uint32
	netResponse IResponse
}

func NewFakeSessionMaker() *FakeSessionMaker {
	return &FakeSessionMaker{}
}

func (f *FakeSessionMaker) MakeSession(response IResponse) (ISession, error) {
	f.cnt++
	return &FakeSession{id: f.cnt, netResponse: response}, nil
}

func (f *FakeSession) ID() uint32 {
	return f.id
}

func (f *FakeSession) CloseAll() {
	return
}

func (f *FakeSession) CommandCb(allResponse *AllRequest) error {
	buf, _ := json.MarshalIndent(allResponse, "", " ")
	fmt.Println(string(buf))

	cmd := allResponse.Cmd
	var rspBuf []byte
	if cmd == CmdStart {
		startRsp := StartResponse{
			Type:      cmd,
			SessionID: f.id,
			UDPPort:   UDPPort,
		}
		rspBuf, _ = json.Marshal(&startRsp)
	} else if cmd == CmdEnd {
		endRsp := EndResponse{
			Type: cmd,
		}
		rspBuf, _ = json.Marshal(&endRsp)
	}
	_, err := f.netResponse.Write(rspBuf)
	if err != nil {
		panic(err)
	}
	return nil
}
func (f *FakeSession) DataCb(data []byte, seq uint32) {
	return
}
