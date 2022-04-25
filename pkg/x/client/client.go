package client

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"
	"xaux/pkg/x"
)

var (
	ErrClient   = errors.New("x client error")
	ErrNotStart = fmt.Errorf("%w, %s", ErrClient, "status not start")
	ErrNotEnd   = fmt.Errorf("%w, %s", ErrClient, "status not end")
)

type Client struct {
	status    int32
	tcpConn   net.Conn
	udpConn   *net.UDPConn
	sessionID uint32
	udpPort   int32
	seq       uint32
	agentIP   string
	endMsg    string
}

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
		status:  x.StatusInit,
		tcpConn: conn,
		agentIP: addr.IP.String(),
	}
	return client, nil
}

func (c *Client) Close() {
	if c.tcpConn != nil {
		c.tcpConn.Close()
	}
	if c.udpConn != nil {
		c.udpConn.Close()
	}
}

func (c *Client) Start(conf x.StartConfig) error {
	start := x.Start{
		Cmd:    x.CmdStart,
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

	startRsp := x.StartResponse{}
	c.tcpConn.SetReadDeadline(time.Now().Add(time.Second * 10))
	jDecoder := json.NewDecoder(c.tcpConn)
	err = jDecoder.Decode(&startRsp)
	if err != nil {
		return err
	}
	if startRsp.Cmd != x.CmdStart {
		return ErrNotStart
	}

	c.sessionID = startRsp.SessionID
	c.udpPort = startRsp.UDPPort
	c.status = x.StatusStart
	c.tcpConn.SetReadDeadline(time.Time{})
	return nil
}

func (c *Client) End() error {
	end := x.End{
		Cmd: x.CmdEnd,
	}
	buf, err := json.Marshal(&end)
	if err != nil {
		return err
	}
	_, err = c.tcpConn.Write(buf)
	if err != nil {
		return err
	}

	endRsp := x.EndResponse{}
	c.tcpConn.SetReadDeadline(time.Now().Add(time.Second * 10))
	jDecoder := json.NewDecoder(c.tcpConn)
	err = jDecoder.Decode(&endRsp)
	if err != nil {
		return err
	}
	if endRsp.Cmd != x.CmdEnd {
		return ErrNotEnd
	}

	c.status = x.StatusEnd
	c.endMsg = endRsp.Msg
	c.tcpConn.SetReadDeadline(time.Time{})
	return nil
}

func (c *Client) Send(data []byte) (err error) {
	if c.status != x.StatusStart {
		return ErrNotStart
	}

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
	// 这里处理 IP 分片啊
	dataLen := len(data)
	bufAll := make([]byte, dataLen+16)
	for {
		sentLen := dataLen
		if sentLen == 0 {
			break
		} else if sentLen > 1200 {
			sentLen = 1200
		}
		dataLen -= sentLen

		buf := bufAll[0 : sentLen+16]
		c.seq++
		binary.BigEndian.PutUint32(buf[4:8], c.seq)
		binary.BigEndian.PutUint32(buf[0:4], c.sessionID)
		copy(buf[16:], data)
		_, err = c.udpConn.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}
