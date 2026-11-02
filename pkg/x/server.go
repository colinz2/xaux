package x

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"

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
	sessionMu    sync.RWMutex
	debug        bool
}

type Option func(s *Server)

func WithOptionSessionMaker(sm ISessionMaker) Option {
	return func(s *Server) {
		s.sessionMaker = sm
	}
}

func WithDebug(debug bool) Option {
	return func(s *Server) {
		s.debug = debug
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

	return server
}

func (s *Server) getNewSession(conn net.Conn) (ISession, error) {
	sess, err := s.sessionMaker.MakeSession(&TCPResponse{Conn: conn})
	if err != nil {
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
	if s.debug {
		log.Printf("client connected: %s", conn.RemoteAddr())
		defer log.Printf("client disconnected: %s", conn.RemoteAddr())
	}

	sess, err := s.getNewSession(conn)
	if err != nil {
		if s.debug {
			log.Printf("create ASR session: %v", err)
		}
		return
	}
	defer s.releaseSession(sess)
	defer sess.CloseAll()

	reader := json.NewDecoder(conn)
	for {
		request := AllRequest{}
		if err := reader.Decode(&request); err != nil {
			if s.debug && err != io.EOF {
				log.Printf("session %d: decode client command: %v", sess.ID(), err)
			}
			return
		}
		if s.debug {
			log.Printf("session %d: command=%q config=%+v", sess.ID(), request.Cmd, request.Config)
		}
		if request.Cmd == CmdStart && request.StatusUpdates {
			status := StatusResponse{Type: TypeStatus, Status: StartStatusConnecting}
			if err := json.NewEncoder(conn).Encode(status); err != nil {
				log.Printf("session %d: write start status: %v", sess.ID(), err)
				return
			}
		}
		if err := sess.CommandCb(&request); err != nil {
			if s.debug {
				log.Printf("session %d: execute command %q: %v", sess.ID(), request.Cmd, err)
			}
			return
		}
	}
}

func (s *Server) tcpServerStart() error {
	for {
		conn, err := s.tcpListener.Accept()
		if err != nil {
			return err
		}
		go s.processTcp(conn)
	}
}

func (s *Server) udpServerStart() error {
	buf := make([]byte, 2048)
	for {
		dataLen, _, err := s.udpConn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if dataLen <= 16 {
			continue
		}

		sessionID := binary.BigEndian.Uint32(buf[0:4])
		seq := binary.BigEndian.Uint32(buf[4:8])
		s.sessionMu.RLock()
		sess, exists := s.sessionMap[sessionID]
		if exists && sess.ID() == sessionID {
			sess.DataCb(buf[16:dataLen], seq)
		}
		s.sessionMu.RUnlock()
		if s.debug && !exists {
			log.Printf("UDP packet for unknown session %d: seq=%d bytes=%d", sessionID, seq, dataLen-16)
		}
	}
}

func (s *Server) Start() error {
	if err := s.listen(); err != nil {
		s.Close()
		return err
	}
	if s.debug {
		log.Printf("proxy listening: tcp=%s udp=%s", s.tcpListener.Addr(), s.udpConn.LocalAddr())
	}

	var group errgroup.Group
	group.Go(func() error {
		defer s.Close()
		return s.udpServerStart()
	})
	group.Go(func() error {
		defer s.Close()
		return s.tcpServerStart()
	})
	return group.Wait()
}

func (s *Server) listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.TcpPort))
	if err != nil {
		return err
	}
	s.tcpListener = listener

	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: s.UdpPort,
	})
	if err != nil {
		return err
	}
	s.udpConn = udpConn
	if err := udpConn.SetReadBuffer(4 * 1024 * 1024); err != nil && s.debug {
		log.Printf("set UDP read buffer: %v", err)
	}
	return nil
}

func (s *Server) Close() {
	if s.udpConn != nil {
		s.udpConn.Close()
	}

	if s.tcpListener != nil {
		s.tcpListener.Close()
	}
}
