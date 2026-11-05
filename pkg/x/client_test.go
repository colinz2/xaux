package x

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestClientSendsRemainingAudioBeforeEnd(t *testing.T) {
	client, udp := newTestClient(t, "", "")
	payload := []byte{1, 2, 3, 4}
	if err := client.Start(StartConfig{SampleRate: 16000, BitsPerSample: 16}); err != nil {
		t.Fatal(err)
	}
	if err := client.Send(payload); err != nil {
		t.Fatal(err)
	}
	if err := client.End(); err != nil {
		t.Fatal(err)
	}

	if err := udp.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 2048)
	n, _, err := udp.ReadFromUDP(packet)
	if err != nil {
		t.Fatal(err)
	}
	if n != udpHeaderLen+len(payload) {
		t.Fatalf("packet length = %d, want %d", n, udpHeaderLen+len(payload))
	}
	if got := binary.BigEndian.Uint32(packet[:4]); got != 42 {
		t.Fatalf("session ID = %d, want 42", got)
	}
	if got := binary.BigEndian.Uint32(packet[4:8]); got != 1 {
		t.Fatalf("sequence = %d, want 1", got)
	}
	if !bytes.Equal(packet[udpHeaderLen:n], payload) {
		t.Fatalf("payload = %v, want %v", packet[udpHeaderLen:n], payload)
	}
}

func TestClientFlushesSmallAudioWithoutEnd(t *testing.T) {
	client, udp := newTestClient(t, "", "")
	payload := []byte{5, 6, 7, 8}
	if err := client.Start(StartConfig{SampleRate: 16000, BitsPerSample: 16}); err != nil {
		t.Fatal(err)
	}
	if err := client.Send(payload); err != nil {
		t.Fatal(err)
	}

	// The payload is smaller than UDPSendLen, so only the timed flush can
	// deliver it before End is called.
	if err := udp.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 2048)
	n, _, err := udp.ReadFromUDP(packet)
	if err != nil {
		t.Fatal(err)
	}
	if n != udpHeaderLen+len(payload) {
		t.Fatalf("packet length = %d, want %d", n, udpHeaderLen+len(payload))
	}
	if !bytes.Equal(packet[udpHeaderLen:n], payload) {
		t.Fatalf("payload = %v, want %v", packet[udpHeaderLen:n], payload)
	}
	if err := client.End(); err != nil {
		t.Fatal(err)
	}
}

func TestClientPropagatesStartError(t *testing.T) {
	client, _ := newTestClient(t, "start failed", "")
	err := client.Start(StartConfig{SampleRate: 16000})
	if err == nil || err.Error() != "start failed" {
		t.Fatalf("Start error = %v", err)
	}
	if err := client.Send([]byte{1}); !errors.Is(err, ErrNotStart) {
		t.Fatalf("Send error = %v, want ErrNotStart", err)
	}
}

func TestClientPropagatesEndError(t *testing.T) {
	client, _ := newTestClient(t, "", "end failed")
	if err := client.Start(StartConfig{SampleRate: 16000}); err != nil {
		t.Fatal(err)
	}
	if err := client.End(); err == nil || err.Error() != "end failed" {
		t.Fatalf("End error = %v", err)
	}
}

func TestClientReceivesConnectingStatusWhileStartWaits(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	allowStartResponse := make(chan struct{})
	defer func() {
		select {
		case <-allowStartResponse:
		default:
			close(allowStartResponse)
		}
	}()
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		var request Start
		if err := json.NewDecoder(conn).Decode(&request); err != nil {
			serverDone <- err
			return
		}
		if request.Cmd != CmdStart || !request.StatusUpdates {
			serverDone <- fmt.Errorf("start request = %+v, want status updates enabled", request)
			return
		}

		encoder := json.NewEncoder(conn)
		if err := encoder.Encode(&StatusResponse{Type: TypeStatus, Status: StartStatusConnecting}); err != nil {
			serverDone <- err
			return
		}
		<-allowStartResponse
		serverDone <- encoder.Encode(&StartResponse{Type: TypeRspStart, SessionID: 1, UDPPort: UDPPort})
	}()

	statuses := make(chan *AllResponse, 1)
	client, err := NewClient(listener.Addr().String(), func(response *AllResponse) error {
		statuses <- response
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	startDone := make(chan error, 1)
	go func() {
		startDone <- client.Start(StartConfig{SampleRate: 16000})
	}()

	select {
	case status := <-statuses:
		if status.Type != TypeStatus || status.Status != StartStatusConnecting {
			t.Fatalf("status response = %+v", status)
		}
	case err := <-startDone:
		t.Fatalf("Start completed before status was received: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for connecting status")
	}

	close(allowStartResponse)
	select {
	case err := <-startDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Start response")
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("test server: %v", err)
	}
}

func TestNewClientRejectsNilCallback(t *testing.T) {
	if _, err := NewClient("127.0.0.1:1", nil); err == nil {
		t.Fatal("expected nil callback to be rejected")
	}
}

func TestClientReportsASRStop(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var request AllRequest
		if err := json.NewDecoder(conn).Decode(&request); err != nil {
			return
		}
		encoder := json.NewEncoder(conn)
		_ = encoder.Encode(&StartResponse{Type: TypeRspStart, SessionID: 1, UDPPort: UDPPort})
		time.Sleep(20 * time.Millisecond)
		_ = encoder.Encode(&StopResponse{Type: TypeStop, Error: Error{Msg: "ASR timeout"}})
		_, _ = conn.Read(make([]byte, 1))
	}()

	responses := make(chan *AllResponse, 1)
	client, err := NewClient(listener.Addr().String(), func(response *AllResponse) error {
		responses <- response
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.Start(StartConfig{SampleRate: 16000}); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-responses:
		if response.Type != TypeStop || response.Error.Msg != "ASR timeout" {
			t.Fatalf("stop response = %+v", response)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ASR stop")
	}
	if err := client.Send([]byte{1}); !errors.Is(err, ErrNotStart) {
		t.Fatalf("Send after STOP error = %v", err)
	}
	client.Close()
	<-serverDone
}

func newTestClient(t *testing.T, startError, endError string) (*Client, *net.UDPConn) {
	t.Helper()
	udp, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		udp.Close()
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		decoder := json.NewDecoder(conn)
		encoder := json.NewEncoder(conn)
		for {
			var request AllRequest
			if err := decoder.Decode(&request); err != nil {
				if errors.Is(err, io.EOF) {
					serverDone <- nil
				} else {
					serverDone <- err
				}
				return
			}
			var response interface{}
			switch request.Cmd {
			case CmdStart:
				response = &StartResponse{
					Type: TypeRspStart, SessionID: 42,
					UDPPort: int32(udp.LocalAddr().(*net.UDPAddr).Port),
					Error:   Error{Msg: startError},
				}
			case CmdEnd:
				response = &EndResponse{Type: TypeRspEnd, Error: Error{Msg: endError}}
			default:
				serverDone <- fmt.Errorf("unexpected command %q", request.Cmd)
				return
			}
			if err := encoder.Encode(response); err != nil {
				serverDone <- err
				return
			}
		}
	}()

	client, err := NewClient(listener.Addr().String(), func(*AllResponse) error { return nil })
	if err != nil {
		listener.Close()
		udp.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.Close()
		listener.Close()
		udp.Close()
		if err := <-serverDone; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("test server: %v", err)
		}
	})
	return client, udp
}
