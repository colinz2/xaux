package x

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type AsrResultCallBack func(rsp *AllResponse) error

const (
	UDPSendLen      = 1460
	udpHeaderLen    = 16
	responseTimeout = 5 * time.Second
)

var (
	ErrClient          = errors.New("x client error")
	ErrNoLooping       = fmt.Errorf("%w, %s", ErrClient, "no looping read")
	ErrNotStart        = fmt.Errorf("%w, %s", ErrClient, "status not start")
	ErrStartRspTimeOut = fmt.Errorf("%w, %s", ErrClient, "start response timeout")
	ErrEndRspTimeOut   = fmt.Errorf("%w, %s", ErrClient, "end response timeout")
	ErrNotEnd          = fmt.Errorf("%w, %s", ErrClient, "status not end")
)

type Client struct {
	status    int32
	running   int32
	tcpConn   net.Conn
	udpConn   *net.UDPConn
	sessionID uint32
	udpPort   int32
	seq       uint32
	agentIP   string
	buffer    bytes.Buffer

	startRspChan chan *AllResponse
	endRspChan   chan *AllResponse
	done         chan struct{}
	closeOnce    sync.Once
	commandMu    sync.Mutex
	audioMu      sync.Mutex
	cb           AsrResultCallBack
}

func NewClient(agentAddr string, cb AsrResultCallBack) (*Client, error) {
	if cb == nil {
		return nil, errors.New("x client callback is nil")
	}
	addr, err := net.ResolveTCPAddr("tcp", agentAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		status:       StatusInit,
		running:      1,
		tcpConn:      conn,
		agentIP:      addr.IP.String(),
		seq:          1,
		startRspChan: make(chan *AllResponse, 1),
		endRspChan:   make(chan *AllResponse, 1),
		done:         make(chan struct{}),
		cb:           cb,
	}
	go client.loopResponse()
	return client, nil
}

func (c *Client) Start(config StartConfig) error {
	if !c.isRunning() {
		return ErrNoLooping
	}
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	if err := c.start(config); err != nil {
		return err
	}
	atomic.StoreInt32(&c.status, StatusStart)
	return nil
}

func (c *Client) End() error {
	if !c.isRunning() {
		return ErrNoLooping
	}
	if atomic.LoadInt32(&c.status) != StatusStart {
		return nil
	}

	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	c.audioMu.Lock()
	err := c.sendBuffered(true)
	c.audioMu.Unlock()
	if err != nil {
		return err
	}
	if err := c.end(); err != nil {
		return err
	}
	atomic.StoreInt32(&c.status, StatusEnd)
	return nil
}

func (c *Client) Send(data []byte) error {
	if !c.isRunning() {
		return ErrNoLooping
	}
	if atomic.LoadInt32(&c.status) != StatusStart {
		return ErrNotStart
	}

	c.audioMu.Lock()
	defer c.audioMu.Unlock()
	return c.send(data)
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		atomic.StoreInt32(&c.running, 0)
		atomic.StoreInt32(&c.status, StatusInit)
		_ = c.tcpConn.Close()

		c.audioMu.Lock()
		if c.udpConn != nil {
			_ = c.udpConn.Close()
			c.udpConn = nil
		}
		c.audioMu.Unlock()
	})
}

func (c *Client) isRunning() bool {
	return atomic.LoadInt32(&c.running) == 1
}

func (c *Client) loopResponse() {
	defer func() {
		atomic.StoreInt32(&c.status, StatusInit)
		atomic.StoreInt32(&c.running, 0)
		close(c.done)
	}()

	reader := json.NewDecoder(c.tcpConn)
	for {
		response := AllResponse{}
		if err := reader.Decode(&response); err != nil {
			if c.isRunning() && !errors.Is(err, io.EOF) {
				log.Printf("x client response loop: %v", err)
			}
			return
		}

		switch response.Type {
		case TypeRspStart:
			queueResponse(c.startRspChan, &response)
		case TypeRspEnd:
			queueResponse(c.endRspChan, &response)
		case TypeStop:
			if !response.ConnectionClose {
				atomic.StoreInt32(&c.status, StatusEnd)
			}
			if err := c.cb(&response); err != nil {
				log.Printf("x client stop callback: %v", err)
			}
			if response.ConnectionClose {
				return
			}
		default:
			if err := c.cb(&response); err != nil {
				log.Printf("x client result callback: %v", err)
			}
		}
	}
}

func queueResponse(queue chan *AllResponse, response *AllResponse) {
	select {
	case queue <- response:
		return
	default:
	}
	select {
	case <-queue:
	default:
	}
	select {
	case queue <- response:
	default:
	}
}

func (c *Client) getStartRsp() (*StartResponse, error) {
	timer := time.NewTimer(responseTimeout)
	defer timer.Stop()
	select {
	case response := <-c.startRspChan:
		return &StartResponse{
			Type:      response.Type,
			SessionID: response.SessionID,
			TaskID:    response.TaskID,
			UDPPort:   response.UDPPort,
			Error:     response.Error,
		}, nil
	case <-c.done:
		return nil, ErrNoLooping
	case <-timer.C:
		return nil, ErrStartRspTimeOut
	}
}

func (c *Client) start(config StartConfig) error {
	request := Start{Cmd: CmdStart, Config: config}
	if err := json.NewEncoder(c.tcpConn).Encode(request); err != nil {
		return err
	}
	response, err := c.getStartRsp()
	if err != nil {
		return err
	}
	if response.Type != TypeRspStart {
		return ErrNotStart
	}
	if response.Error.Msg != "" {
		return errors.New(response.Error.Msg)
	}

	c.sessionID = response.SessionID
	c.udpPort = response.UDPPort
	return nil
}

func (c *Client) end() error {
	if err := json.NewEncoder(c.tcpConn).Encode(End{Cmd: CmdEnd}); err != nil {
		return err
	}

	timer := time.NewTimer(responseTimeout)
	defer timer.Stop()
	select {
	case response := <-c.endRspChan:
		if response.Type != TypeRspEnd {
			return ErrNotEnd
		}
		if response.Error.Msg != "" {
			return errors.New(response.Error.Msg)
		}
		return nil
	case <-c.done:
		return ErrNoLooping
	case <-timer.C:
		return ErrEndRspTimeOut
	}
}

func (c *Client) send(data []byte) error {
	c.buffer.Write(data)
	if c.udpConn == nil {
		addr := net.UDPAddr{IP: net.ParseIP(c.agentIP), Port: int(c.udpPort)}
		conn, err := net.DialUDP("udp", nil, &addr)
		if err != nil {
			return err
		}
		c.udpConn = conn
	}
	if c.buffer.Len() < UDPSendLen {
		return nil
	}
	return c.sendBuffered(false)
}

func (c *Client) sendBuffered(sendAll bool) error {
	for c.buffer.Len() >= UDPSendLen || (sendAll && c.buffer.Len() > 0) {
		if c.udpConn == nil {
			return errors.New("x client UDP connection is nil")
		}
		payloadLen := UDPSendLen
		if c.buffer.Len() < payloadLen {
			payloadLen = c.buffer.Len()
		}
		packet := make([]byte, udpHeaderLen+payloadLen)
		binary.BigEndian.PutUint32(packet[0:4], c.sessionID)
		binary.BigEndian.PutUint32(packet[4:8], c.seq)
		if _, err := io.ReadFull(&c.buffer, packet[udpHeaderLen:]); err != nil {
			return err
		}
		if _, err := c.udpConn.Write(packet); err != nil {
			return err
		}
		c.seq++
	}
	return nil
}
