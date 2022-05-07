package x

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/realzhangm/xaux/pkg/common/doa"
	"net"
	"sync"
	"sync/atomic"
)

import (
	"golang.org/x/sync/errgroup"
)

type ISessionMaker interface {
	MakeSession(IResponse) (ISession, error)
}

type ISession interface {
	ID() uint32
	CommandCb(allResponse *AllRequest) error
	DataCb(data []byte, seq uint32)
	CloseAll()
}

type IResponse interface {
	Write([]byte) (int, error)
}

type TCPResponse struct {
	Conn net.Conn
}

func (t *TCPResponse) Write(data []byte) (int, error) {
	return t.Conn.Write(data)
}

func (t *TCPResponse) Close() {
	t.Conn.Close()
}

type Conf struct {
	TcpPort int
	UdpPort int
}

type Server struct {
	Conf
	sessionCnt   int64
	sessionMap   map[uint32]ISession
	tcpListener  net.Listener
	udpConn      *net.UDPConn
	sessionMaker ISessionMaker
	sessionMu    sync.Mutex
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
		server.TcpPort = TCPPort
	}

	if server.UdpPort == 0 {
		server.UdpPort = UDPPort
	}

	if server.sessionMaker == nil {
		server.sessionMaker = NewFakeSessionMaker()
	}

	doa.MustTrue(server.sessionMaker != nil, "sessionMake is nil")
	return server
}

func (s *Server) getNewSession(conn net.Conn) (ISession, error) {
	sess, err := s.sessionMaker.MakeSession(&TCPResponse{Conn: conn})
	if err != nil {
		fmt.Println("MakeSession err :", err)
		return nil, err
	}
	atomic.AddInt64(&s.sessionCnt, 1)

	s.sessionMu.Lock()
	s.sessionMap[sess.ID()] = sess
	s.sessionMu.Unlock()
	return sess, nil
}

func (s *Server) releaseSession(sess ISession) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	delete(s.sessionMap, sess.ID())
}

func (s *Server) processTcp(conn net.Conn) {
	defer conn.Close()

	sess, err := s.getNewSession(conn)
	if err != nil {
		return
	}
	defer s.releaseSession(sess)

	reader := json.NewDecoder(conn)
	for {
		allReq := AllRequest{}
		err = reader.Decode(&allReq)
		if err != nil {
			break
		}
		err = sess.CommandCb(&allReq)
		if err != nil {
			break
		}
	}
	sess.CloseAll()
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
		dataLen, _, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}
		//fmt.Println("cAddr :=", cAddr.String())

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
