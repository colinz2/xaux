package ai_proxy

import (
	"github.com/realzhangm/xaux/pkg/ai_proxy/asr/aliyun"
	"github.com/realzhangm/xaux/pkg/x"
)

type Proxy struct {
	xServer *x.Server
}

func NewProxy() (*Proxy, error) {
	proxy := Proxy{}

	conf := x.Conf{
		TcpPort: x.TCPPort,
		UdpPort: x.UDPPort,
	}
	asrSessionMaker := aliyun.NewSessionMaker()
	proxy.xServer = x.NewServer(conf, x.WithOptionSessionMaker(asrSessionMaker))
	return &proxy, nil
}

func (p *Proxy) Start() error {
	return p.xServer.Start()
}
