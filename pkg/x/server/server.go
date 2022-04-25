package server

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"xaux/pkg/doa"
	"xaux/pkg/x"
)

import (
	"golang.org/x/sync/errgroup"
)

type ISessionMaker interface {
	MakeSession() (ISession, error)
}

type ISession interface {
	ID() uint32
	CommandCb(con net.Conn, allResponse *x.AllRequest) error
	DataCb(data []byte, seq uint32) error
}

type Conf struct {
	TcpPort int
	UdpPort int
}

type Server struct {
	Conf
	sessionCnt   int64
	sessionMap   map[uint32]ISession
	sessionMu    sync.Mutex
	tcpListener  net.Listener
	udpConn      *net.UDPConn
	sessionMaker ISessionMaker
}

type Option func(s *Server)

func WithOptionSessionMaker(sm ISessionMaker) Option {
	return func(s *Server) {
		s.sessionMaker = sm
	}
}

func NewServer(conf Conf, opts ...Option) *Server {
	server := &Server{
		Conf:       conf,
		sessionCnt: 0,
		sessionMap: make(map[uint32]ISession),
	}
	for _, op := range opts {
		op(server)
	}

	if server.TcpPort == 0 {
		server.TcpPort = x.TCPPort
	}

	if server.UdpPort == 0 {
		server.UdpPort = x.UDPPort
	}

	if server.sessionMaker == nil {
		server.sessionMaker = NewFakeSessionMaker()
	}

	doa.MustTrue(server.sessionMaker != nil, "sessionMake is nil")
	return server
}

func (s *Server) processTcp(conn net.Conn) {
	sess, err := s.sessionMaker.MakeSession()
	if err != nil {
		conn.Close()
		return
	}
	atomic.AddInt64(&s.sessionCnt, 1)
	defer conn.Close()

	s.sessionMu.Lock()
	s.sessionMap[sess.ID()] = sess
	s.sessionMu.Unlock()

	reader := json.NewDecoder(conn)
	for {
		allRsp := x.AllRequest{}
		err := reader.Decode(&allRsp)
		if err != nil {
			panic(err)
		}
		err = sess.CommandCb(conn, &allRsp)
		if err != nil {
			return
		}
	}
}

func (s *Server) tcpServerStart() (err error) {
	s.tcpListener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.TcpPort))
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
		Port: s.UdpPort,
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
			sID := binary.BigEndian.Uint32(buf[0:4])
			seq := binary.BigEndian.Uint32(buf[4:8])
			s.sessionMu.Lock()
			sess, exist := s.sessionMap[sID]
			s.sessionMu.Unlock()
			if exist {
				doa.MustTrue(sID == sess.ID(), "sID == sess.ID()")
				sess.DataCb(buf[16:dataLen], seq)
			} else {
				fmt.Println(sID, ",can not find")
			}
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
	if s.udpConn != nil {
		s.udpConn.Close()
	}

	if s.tcpListener != nil {
		s.tcpListener.Close()
	}
}