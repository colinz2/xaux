package x

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

type AsrResultCallBack func(rsp *AllResponse) error

const (
	UDPSendLen = 1460
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
	status        int32
	tcpConn       net.Conn
	udpConn       *net.UDPConn
	sessionID     uint32
	udpPort       int32
	seq           uint32
	agentIP       string
	buffer        bytes.Buffer
	rspChan       chan *AllResponse
	endRspChan    chan *AllResponse
	startRspChan  chan *AllResponse
	isLoopingRead int32
	cb            AsrResultCallBack
}

// NewClient TODO 处理返回结果
func NewClient(agentAddr string, cb AsrResultCallBack) (*Client, error) {
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
		tcpConn:      conn,
		agentIP:      addr.IP.String(),
		seq:          1,
		rspChan:      make(chan *AllResponse, 1),
		endRspChan:   make(chan *AllResponse, 0),
		startRspChan: make(chan *AllResponse, 0),
		cb:           cb,
	}
	atomic.AddInt32(&client.isLoopingRead, 1)
	go client.loopResponse()
	return client, nil
}

func (c *Client) Start(conf StartConfig) error {
	if atomic.LoadInt32(&c.isLoopingRead) != 1 {
		return ErrNoLooping
	}

	err := c._start(&conf)
	if err != nil {
		return err
	}
	c.status = StatusStart
	return nil
}

func (c *Client) End() error {
	if atomic.LoadInt32(&c.isLoopingRead) != 1 {
		return ErrNoLooping
	}
	if c.status != StatusStart {
		return nil
	}

	err := c._end()
	if err != nil {
		return nil
	}
	c.status = StatusEnd
	return nil
}

func (c *Client) Send(data []byte) (err error) {
	if atomic.LoadInt32(&c.isLoopingRead) != 1 {
		return ErrNoLooping
	}

	if c.status != StatusStart {
		return ErrNotStart
	}
	return c._send(data)
}

func (c *Client) Close() {
	c.isLoopingRead = 0
	c.status = StatusInit

	if c.tcpConn != nil {
		c.tcpConn.Close()
		c.tcpConn = nil
	}
	if c.udpConn != nil {
		c.udpConn.Close()
		c.udpConn = nil
		c.isLoopingRead = 0
	}
}

func (c *Client) loopResponse() {
	if c.cb == nil {
		panic("c.cb is nil")
	}
	reader := json.NewDecoder(c.tcpConn)
	var err error = nil
	for {
		allRsp := AllResponse{}
		err = reader.Decode(&allRsp)
		if err != nil {
			break
		}
		switch allRsp.Type {
		case TypeRspStart:
			c.startRspChan <- &allRsp
		case TypeRspEnd:
			c.endRspChan <- &allRsp
		case TypeStop:
			sr := StopResponse{
				Type:            allRsp.Type,
				Error:           allRsp.Error,
				ConnectionClose: allRsp.ConnectionClose,
			}
			if sr.ConnectionClose {
				break
			} else {
				c.status = StatusEnd
			}
		default:
			if c.cb != nil {
				c.cb(&allRsp)
			}
		}
	}
	fmt.Println("loopResponse err := ", err.Error())
	c.status = StatusInit
	atomic.AddInt32(&c.isLoopingRead, -1)
}

func (c *Client) getStartRsp() (*StartResponse, error) {
	timer := time.NewTimer(time.Second * 5)
	var allRsp *AllResponse
	select {
	case allRsp = <-c.startRspChan:
		timer.Stop()
	case <-timer.C:
		return nil, ErrStartRspTimeOut
	}
	return &StartResponse{
		Type:      allRsp.Type,
		SessionID: allRsp.SessionID,
		TaskID:    allRsp.TaskID,
		UDPPort:   allRsp.UDPPort,
		Error:     allRsp.Error,
	}, nil
}

func (c *Client) _start(conf *StartConfig) error {
	start := Start{
		Cmd:    CmdStart,
		Config: *conf,
	}
	buf, err := json.Marshal(&start)
	if err != nil {
		return err
	}
	_, err = c.tcpConn.Write(buf)
	if err != nil {
		return err
	}

	startRsp, err := c.getStartRsp()
	if err != nil {
		return err
	}
	if startRsp.Type != TypeRspStart {
		return ErrNotStart
	}

	c.sessionID = startRsp.SessionID
	c.udpPort = startRsp.UDPPort
	return nil
}

func (c *Client) _end() error {
	end := End{
		Cmd: CmdEnd,
	}
	buf, err := json.Marshal(&end)
	if err != nil {
		return err
	}
	_, err = c.tcpConn.Write(buf)
	if err != nil {
		return err
	}

	select {
	case allRsp := <-c.endRspChan:
		if allRsp.Type != TypeRspEnd {
			return ErrNotEnd
		}
	case <-time.After(time.Second * 5):
		return ErrEndRspTimeOut
	}
	return nil
}

func (c *Client) _send(data []byte) (err error) {
	c.buffer.Write(data)

	if c.udpConn == nil {
		addr := net.UDPAddr{
			IP:   net.ParseIP(c.agentIP),
			Port: int(c.udpPort),
		}
		c.udpConn, err = net.DialUDP("udp", nil, &addr)
		if err != nil {
			return err
		}
	}
	// 先判断一下长度
	if c.buffer.Len() < UDPSendLen {
		return nil
	}
	// 这里用 false ，因为每次写入buffer的数据是一个采样
	return c.sentBuffer(false)
}

func (c *Client) sentBuffer(sendALL bool) error {
	// 这里处理 IP 分片啊
	var err error = nil
	bufAll := make([]byte, UDPSendLen+16)
	for {
		buf := bufAll[0:]
		if c.buffer.Len() < UDPSendLen {
			if sendALL {
				buf = bufAll[0 : c.buffer.Len()+16]
			} else {
				buf = bufAll[0:16]
			}
		}

		if len(buf) <= 16 {
			break
		}

		binary.BigEndian.PutUint32(buf[0:4], c.sessionID)
		binary.BigEndian.PutUint32(buf[4:8], c.seq)
		_, err = c.buffer.Read(buf[16:])
		if err != nil {
			return err
		}

		_, err = c.udpConn.Write(buf)
		if err != nil {
			return err
		}
		c.seq++
	}
	return nil
}
