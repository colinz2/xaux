package server

import (
	"context"
	"fmt"
	"net"
)

import (
	"golang.org/x/sync/errgroup"
)

type ISession interface {
	MakeSession() (ISession, error)
	GetID() uint32
	CommandCb(con net.TCPConn) error
	DataCb(data []byte, seq uint32) error
}

type Server struct {
	tcpPort     int
	udpPort     int
	sessionCnt  uint32
	sessionMap  map[uint32]ISession
	tcpListener net.Listener
	udpConn     *net.UDPConn
}

func NewServer() *Server {
	return &Server{
		tcpPort:    0,
		udpPort:    0,
		sessionCnt: 0,
		sessionMap: nil,
	}
}

func (s *Server) processTcp(conn net.Conn) {

}

func (s *Server) tcpServerStart() (err error) {
	s.tcpListener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.tcpPort))
	if err != nil {
		return
	}

	for {
		conn, err := s.tcpListener.Accept()
		if err != nil {
			return err
		}
		go s.processTcp(conn)
	}
	return nil
}

func (s *Server) udpServerStart() (err error) {
	s.udpConn, err = net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: s.udpPort,
	})
	if err != nil {
		return err
	}
	buf := make([]byte, 2048)
	for {
		dataLen, cAddr, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		fmt.Println("cAddr :=", cAddr.String())

		if dataLen > 16 {
			//sID := binary.BigEndian.Uint32(buf[0:4])
			//seq := binary.BigEndian.Uint32(buf[4:8])

		}
	}
	return nil
}

func (s *Server) Start() error {
	g, _ := errgroup.WithContext(context.TODO())
	g.Go(func() error {
		return s.udpServerStart()
	})

	g.Go(func() error {
		return s.tcpServerStart()
	})

	err := g.Wait()
	return err
}

func (s *Server) Close() {
	s.tcpListener.Close()
}
