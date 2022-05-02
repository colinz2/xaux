package x

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	UDPSendLen = 1300
)

var (
	ErrClient   = errors.New("x client error")
	ErrNotStart = fmt.Errorf("%w, %s", ErrClient, "status not start")
	ErrNotEnd   = fmt.Errorf("%w, %s", ErrClient, "status not end")
)

type Client struct {
	status     int32
	tcpConn    net.Conn
	udpConn    *net.UDPConn
	sessionID  uint32
	udpPort    int32
	seq        uint32
	agentIP    string
	endMsg     string
	buffer     bytes.Buffer
	rspChan    chan *AllResponse
	endRspChan chan *AllResponse
}

// TODO 处理返回结果

func NewClient(agentAddr string) (*Client, error) {
	addr, err := net.ResolveTCPAddr("tcp", agentAddr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	client := &Client{
		status:     StatusInit,
		tcpConn:    conn,
		agentIP:    addr.IP.String(),
		seq:        1,
		rspChan:    make(chan *AllResponse, 1),
		endRspChan: make(chan *AllResponse, 1),
	}
	return client, nil
}

func (c *Client) Close() {
	if c.tcpConn != nil {
		c.tcpConn.Close()
	}
	if c.udpConn != nil {
		c.sentBuffer(true)
		c.udpConn.Close()
	}
}

func (c *Client) goToLoopResponse(cb func(rsp *AllResponse) error) {
	reader := json.NewDecoder(c.tcpConn)
	for {
		allRsp := AllResponse{}
		err := reader.Decode(&allRsp)
		if err != nil {
			return
		}
		fmt.Println("allRsp : ", allRsp.Cmd)
		switch allRsp.Cmd {
		case CmdStart:
		case CmdEnd:
			c.endRspChan <- &allRsp
		default:
			cb(&allRsp)
		}
	}
	c.status = StatusInit
}

func (c *Client) Start(conf StartConfig, cb func(rsp *AllResponse) error) error {
	start := Start{
		Cmd:    CmdStart,
		Config: conf,
	}
	buf, err := json.Marshal(&start)
	if err != nil {
		return err
	}
	_, err = c.tcpConn.Write(buf)
	if err != nil {
		return err
	}

	startRsp := StartResponse{}
	c.tcpConn.SetReadDeadline(time.Now().Add(time.Second * 10))
	jDecoder := json.NewDecoder(c.tcpConn)
	err = jDecoder.Decode(&startRsp)
	if err != nil {
		return err
	}
	if startRsp.Cmd != CmdStart {
		return ErrNotStart
	}

	c.sessionID = startRsp.SessionID
	c.udpPort = startRsp.UDPPort
	c.status = StatusStart
	c.tcpConn.SetReadDeadline(time.Time{})
	go c.goToLoopResponse(cb)
	return nil
}

func (c *Client) End() error {
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

	// TODO timeout
	var allRsp *AllResponse
	select {
	case allRsp = <-c.endRspChan:
	}

	if allRsp.Cmd != CmdEnd {
		return ErrNotEnd
	}
	c.status = StatusEnd
	c.endMsg = allRsp.Msg
	return nil
}

func (c *Client) Send(data []byte) (err error) {
	if c.status != StatusStart {
		return ErrNotStart
	}

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
