package x

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
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
	id := atomic.AddUint32(&f.cnt, 1)
	return &FakeSession{id: id, netResponse: response}, nil
}

func (f *FakeSession) ID() uint32 {
	return f.id
}

func (f *FakeSession) CloseAll() {}

func (f *FakeSession) CommandCb(request *AllRequest) error {
	requestJSON, err := json.MarshalIndent(request, "", " ")
	if err != nil {
		return err
	}
	fmt.Println(string(requestJSON))

	var response interface{}
	switch request.Cmd {
	case CmdStart:
		response = &StartResponse{
			Type:      TypeRspStart,
			SessionID: f.id,
			UDPPort:   UDPPort,
		}
	case CmdEnd:
		response = &EndResponse{Type: TypeRspEnd}
	default:
		return fmt.Errorf("unsupported command %q", request.Cmd)
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = f.netResponse.Write(responseJSON)
	return err
}

func (f *FakeSession) DataCb(data []byte, seq uint32) {}
