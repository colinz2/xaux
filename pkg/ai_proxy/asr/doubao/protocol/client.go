package protocol

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var ErrAudioFinished = errors.New("doubao audio stream already finished")

// AsrWsClient implements the bidirectional streaming part of the Doubao
// websocket protocol. Audio data is supplied by the caller instead of a file.
type AsrWsClient struct {
	seq      int32
	url      string
	connect  *websocket.Conn
	writeMu  sync.Mutex
	finished bool
}

func NewAsrWsClient(url string) *AsrWsClient {
	return &AsrWsClient{url: url}
}

func (c *AsrWsClient) Start(ctx context.Context, config Config, payload AsrRequestPayload) error {
	if err := config.Validate(); err != nil {
		return err
	}
	if c.connect != nil {
		return errors.New("doubao websocket already started")
	}

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, c.url, NewAuthHeader(config))
	if err != nil {
		if resp != nil {
			return fmt.Errorf("dial doubao websocket: %s: %w", resp.Status, err)
		}
		return fmt.Errorf("dial doubao websocket: %w", err)
	}
	started := false
	defer func() {
		if !started {
			_ = conn.Close()
		}
	}()
	if resp != nil {
		log.Printf("doubao logid: %s", resp.Header.Get("X-Tt-Logid"))
	}

	// A context deadline also bounds the initial response read.
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetReadDeadline(deadline); err != nil {
			return err
		}
		defer conn.SetReadDeadline(time.Time{})
	}

	fullRequest, err := NewFullClientRequest(payload)
	if err != nil {
		return err
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, fullRequest); err != nil {
		return fmt.Errorf("write doubao full request: %w", err)
	}

	_, message, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read doubao full response: %w", err)
	}
	response, err := ParseResponse(message)
	if err != nil {
		return fmt.Errorf("parse doubao full response: %w", err)
	}
	if err := responseError(response); err != nil {
		return err
	}
	c.seq = 2
	c.connect = conn
	started = true
	return nil
}

// SendAudio sends one binary audio packet. The final packet must have last set
// to true so its sequence number is negative, as required by the protocol.
func (c *AsrWsClient) SendAudio(segment []byte, last bool) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.connect == nil {
		return errors.New("doubao websocket is not connected")
	}
	if c.finished {
		return ErrAudioFinished
	}

	seq := c.seq
	if last {
		seq = -seq
		c.finished = true
	}
	message, err := NewAudioOnlyRequest(seq, segment)
	if err != nil {
		return err
	}
	if err := c.connect.WriteMessage(websocket.BinaryMessage, message); err != nil {
		return fmt.Errorf("write doubao audio packet: %w", err)
	}
	if !last {
		c.seq++
	}
	return nil
}

// Receive reads responses until the server's final package. The handler is
// called synchronously, so all returned recognition results keep their order.
func (c *AsrWsClient) Receive(handler func(*AsrResponse) error) error {
	if c.connect == nil {
		return errors.New("doubao websocket is not connected")
	}
	for {
		_, message, err := c.connect.ReadMessage()
		if err != nil {
			return fmt.Errorf("read doubao response: %w", err)
		}
		response, err := ParseResponse(message)
		if err != nil {
			return fmt.Errorf("parse doubao response: %w", err)
		}
		if err := responseError(response); err != nil {
			return err
		}
		if handler != nil {
			if err := handler(response); err != nil {
				return err
			}
		}
		if response.IsLastPackage {
			return nil
		}
	}
}

func (c *AsrWsClient) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.connect == nil {
		return nil
	}
	return c.connect.Close()
}

func responseError(response *AsrResponse) error {
	if response == nil {
		return errors.New("doubao returned an empty response")
	}

	message := "unknown error"
	logID := ""
	hasMessage := false
	if response.PayloadMsg != nil {
		logID = response.PayloadMsg.Result.Additions.LogID
		switch {
		case response.PayloadMsg.Error != "":
			message = response.PayloadMsg.Error
			hasMessage = true
		case response.PayloadMsg.Message != "":
			message = response.PayloadMsg.Message
			hasMessage = true
		}
	}
	if response.Code == 0 && !hasMessage {
		return nil
	}
	if response.Code != 0 {
		message = fmt.Sprintf("doubao ASR error %d: %s", response.Code, message)
	} else {
		message = "doubao ASR error: " + message
	}
	if logID != "" {
		message += " (log_id=" + logID + ")"
	}
	return errors.New(message)
}
