package x

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestFakeSessionResponses(t *testing.T) {
	maker := NewFakeSessionMaker()
	var output bytes.Buffer
	session, err := maker.MakeSession(&output)
	if err != nil {
		t.Fatal(err)
	}

	if err := session.CommandCb(&AllRequest{Cmd: CmdStart}); err != nil {
		t.Fatal(err)
	}
	if err := session.CommandCb(&AllRequest{Cmd: CmdEnd}); err != nil {
		t.Fatal(err)
	}

	decoder := json.NewDecoder(&output)
	var start StartResponse
	if err := decoder.Decode(&start); err != nil {
		t.Fatal(err)
	}
	var end EndResponse
	if err := decoder.Decode(&end); err != nil {
		t.Fatal(err)
	}
	if start.Type != TypeRspStart || start.SessionID != session.ID() {
		t.Fatalf("start response = %+v", start)
	}
	if end.Type != TypeRspEnd {
		t.Fatalf("end response = %+v", end)
	}
}

func TestServerSendsConnectingStatusBeforeStartResponse(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	server := NewServer(Conf{}, WithOptionSessionMaker(NewFakeSessionMaker()))
	serverDone := make(chan struct{})
	go func() {
		server.processTcp(serverConn)
		close(serverDone)
	}()

	if err := json.NewEncoder(clientConn).Encode(Start{Cmd: CmdStart, StatusUpdates: true}); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(clientConn)

	var status StatusResponse
	if err := decoder.Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Type != TypeStatus || status.Status != StartStatusConnecting {
		t.Fatalf("status response = %+v", status)
	}

	var start StartResponse
	if err := decoder.Decode(&start); err != nil {
		t.Fatal(err)
	}
	if start.Type != TypeRspStart {
		t.Fatalf("start response = %+v", start)
	}

	_ = clientConn.Close()
	select {
	case <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after client disconnect")
	}
}
