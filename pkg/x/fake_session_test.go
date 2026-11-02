package x

import (
	"bytes"
	"encoding/json"
	"testing"
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
