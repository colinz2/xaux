package main

import (
	"github.com/realzhangm/xaux/pkg/x"
)

func init() {

}

func main() {
	conf := x.Conf{
		TcpPort: x.TCPPort,
		UdpPort: x.UDPPort,
	}
	server := x.NewServer(conf)
	err := server.Start()
	if err != nil {
		panic(err)
	}
}
