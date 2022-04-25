package main

import (
	"xaux/pkg/x"
	"xaux/pkg/x/server"
)

func init() {

}

func main() {
	conf := server.Conf{
		TcpPort: x.TCPPort,
		UdpPort: x.UDPPort,
	}
	server := server.NewServer(conf)
	err := server.Start()
	if err != nil {
		panic(err)
	}
}
