package ai_proxy

import (
	"fmt"
	"strings"

	"github.com/realzhangm/xaux/pkg/ai_proxy/asr/aliyun"
	"github.com/realzhangm/xaux/pkg/ai_proxy/asr/doubao"
	"github.com/realzhangm/xaux/pkg/x"
)

type Proxy struct {
	xServer *x.Server
}

type ProxyOptions struct {
	Debug     bool
	RecordDir string
}

const (
	ASRProviderAliyun = "aliyun"
	ASRProviderDoubao = "doubao"
)

// NewProxy keeps the original Aliyun default.
func NewProxy() (*Proxy, error) {
	return NewProxyWithProvider(ASRProviderAliyun)
}

func NewProxyWithProvider(provider string) (*Proxy, error) {
	return NewProxyWithOptions(provider, ProxyOptions{})
}

func NewProxyWithOptions(provider string, options ProxyOptions) (*Proxy, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = ASRProviderAliyun
	}

	var asrSessionMaker x.ISessionMaker
	var err error
	switch provider {
	case ASRProviderAliyun:
		asrSessionMaker = aliyun.NewSessionMaker()
	case ASRProviderDoubao:
		asrSessionMaker, err = doubao.NewSessionMaker()
	default:
		return nil, fmt.Errorf("unsupported ASR provider %q: expected %q or %q", provider, ASRProviderAliyun, ASRProviderDoubao)
	}
	if err != nil {
		return nil, err
	}

	serverOptions := []x.Option{x.WithOptionSessionMaker(asrSessionMaker)}
	if options.Debug {
		serverOptions = append(serverOptions, x.WithDebug(true))
		serverOptions[0] = x.WithOptionSessionMaker(x.NewDebugSessionMaker(asrSessionMaker, x.DebugConfig{
			Provider:  provider,
			RecordDir: options.RecordDir,
		}))
	}

	conf := x.Conf{TcpPort: x.TCPPort, UdpPort: x.UDPPort}
	return &Proxy{xServer: x.NewServer(conf, serverOptions...)}, nil
}

func (p *Proxy) Start() error {
	return p.xServer.Start()
}
