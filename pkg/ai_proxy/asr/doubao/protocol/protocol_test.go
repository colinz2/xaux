package protocol

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestRequestPackets(t *testing.T) {
	payload := AsrRequestPayload{
		Audio: AudioMeta{Format: "pcm", Codec: "raw", Rate: 16000, Bits: 16, Channel: 1},
		Request: RequestMeta{
			ModelName: "bigmodel", ShowUtterances: true, EnableNonstream: true,
			ResultType: "full", EndWindowSize: 800,
		},
	}
	full, err := NewFullClientRequest(payload)
	if err != nil {
		t.Fatal(err)
	}
	if got := MessageType(full[1] >> 4); got != CLIENT_FULL_REQUEST {
		t.Fatalf("full request type = %d", got)
	}
	compressed := full[12:]
	decoded, err := GzipDecompress(compressed)
	if err != nil {
		t.Fatal(err)
	}
	var gotPayload AsrRequestPayload
	if err := json.Unmarshal(decoded, &gotPayload); err != nil {
		t.Fatal(err)
	}
	if gotPayload.Audio.Format != "pcm" || gotPayload.Audio.Rate != 16000 || !gotPayload.Request.ShowUtterances || gotPayload.Request.ResultType != "full" || gotPayload.Request.EndWindowSize != 800 {
		t.Fatalf("unexpected full payload: %+v", gotPayload)
	}

	audio, err := NewAudioOnlyRequest(-2, []byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if got := MessageType(audio[1] >> 4); got != CLIENT_AUDIO_ONLY_REQUEST {
		t.Fatalf("audio request type = %d", got)
	}
	if got := MessageTypeSpecificFlags(audio[1] & 0x0f); got != NEG_WITH_SEQUENCE {
		t.Fatalf("audio flags = %d", got)
	}
	if got := int32(binary.BigEndian.Uint32(audio[4:8])); got != -2 {
		t.Fatalf("audio sequence = %d", got)
	}
	decoded, err = GzipDecompress(audio[12:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, []byte{1, 2, 3}) {
		t.Fatalf("audio payload = %v", decoded)
	}
}

func TestParseResponse(t *testing.T) {
	compressed, err := GzipCompress([]byte(`{"result":{"text":"你好","additions":{"log_id":"log-1"},"utterances":[{"definite":true,"text":"你好","start_time":10,"end_time":200}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	message := []byte{0x11, byte(SERVER_FULL_RESPONSE<<4) | byte(NEG_WITH_SEQUENCE), 0x11, 0x00}
	sequence := int32(-7)
	message = binary.BigEndian.AppendUint32(message, uint32(sequence))
	message = binary.BigEndian.AppendUint32(message, uint32(len(compressed)))
	message = append(message, compressed...)

	response, err := ParseResponse(message)
	if err != nil {
		t.Fatal(err)
	}
	if !response.IsLastPackage || response.PayloadSequence != -7 {
		t.Fatalf("unexpected response metadata: %+v", response)
	}
	utterances := response.PayloadMsg.Result.Utterances
	if len(utterances) != 1 || !utterances[0].Definite || utterances[0].Text != "你好" {
		t.Fatalf("unexpected response payload: %+v", response.PayloadMsg)
	}
	if response.PayloadMsg.Result.Additions.LogID != "log-1" {
		t.Fatalf("response log ID = %q", response.PayloadMsg.Result.Additions.LogID)
	}
}

func TestResponseErrorIncludesMessageAndLogID(t *testing.T) {
	response := &AsrResponse{PayloadMsg: &AsrResponsePayload{Message: "invalid request"}}
	response.PayloadMsg.Result.Additions.LogID = "log-error"
	err := responseError(response)
	if err == nil || !strings.Contains(err.Error(), "invalid request") || !strings.Contains(err.Error(), "log-error") {
		t.Fatalf("response error = %v", err)
	}
}

func TestParseResponseRejectsMalformedPackets(t *testing.T) {
	tests := [][]byte{
		nil,
		{0x11, 0x90, 0x11},
		{0x10, 0x90, 0x11, 0x00},
		{0x11, 0x91, 0x11, 0x00},
		{0x11, 0x90, 0x11, 0x00},
		{0x11, 0x90, 0x11, 0x00, 0x00, 0x00, 0x00, 0x01},
		{0x11, 0x90, 0x12, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00},
	}
	for _, message := range tests {
		if _, err := ParseResponse(message); err == nil {
			t.Fatalf("ParseResponse(%v) unexpectedly succeeded", message)
		}
	}
}

func TestAuthHeader(t *testing.T) {
	header := NewAuthHeader(Config{APIKey: "api", ResourceID: "resource"})
	if header.Get("X-Api-Key") != "api" {
		t.Fatalf("missing API key: %v", header)
	}
	if header.Get("X-Api-App-Key") != "" || header.Get("X-Api-Access-Key") != "" {
		t.Fatalf("unexpected legacy auth headers: %v", header)
	}
	if header.Get("X-Api-Resource-Id") != "resource" {
		t.Fatalf("missing resource ID: %v", header)
	}
	if _, err := uuid.Parse(header.Get("X-Api-Request-Id")); err != nil {
		t.Fatalf("request ID is not a UUID: %v", err)
	}
}

func TestAsrWsClientStreaming(t *testing.T) {
	serverErr := make(chan error, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" {
			serverErr <- fmt.Errorf("missing API key")
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			serverErr <- err
			return
		}
		defer conn.Close()

		messageType, _, err := conn.ReadMessage()
		if err != nil {
			serverErr <- err
			return
		}
		if messageType != websocket.BinaryMessage {
			serverErr <- fmt.Errorf("full request websocket type = %d", messageType)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, testServerResponse(NO_SEQUENCE, false, `{}`)); err != nil {
			serverErr <- err
			return
		}

		_, firstAudio, err := conn.ReadMessage()
		if err != nil {
			serverErr <- err
			return
		}
		if got := int32(binary.BigEndian.Uint32(firstAudio[4:8])); got != 2 {
			serverErr <- fmt.Errorf("first audio sequence = %d", got)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, testServerResponse(NO_SEQUENCE, false, `{"result":{"text":"一"}}`)); err != nil {
			serverErr <- err
			return
		}

		_, lastAudio, err := conn.ReadMessage()
		if err != nil {
			serverErr <- err
			return
		}
		if got := int32(binary.BigEndian.Uint32(lastAudio[4:8])); got != -3 {
			serverErr <- fmt.Errorf("last audio sequence = %d", got)
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, testServerResponse(NEG_SEQUENCE, true, `{"result":{"text":"一二"}}`)); err != nil {
			serverErr <- err
			return
		}
		serverErr <- nil
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewAsrWsClient(wsURL)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	config := Config{WSURL: wsURL, APIKey: "test-key", ResourceID: "test-resource"}
	if err := client.Start(ctx, config, AsrRequestPayload{}); err != nil {
		t.Fatal(err)
	}
	if err := client.SendAudio([]byte{1}, false); err != nil {
		t.Fatal(err)
	}

	var responses []*AsrResponse
	receiveDone := make(chan error, 1)
	go func() {
		receiveDone <- client.Receive(func(response *AsrResponse) error {
			responses = append(responses, response)
			return nil
		})
	}()
	if err := client.SendAudio([]byte{2}, true); err != nil {
		t.Fatal(err)
	}

	for name, done := range map[string]chan error{"server": serverErr, "receive": receiveDone} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %s", name)
		}
	}
	if len(responses) != 2 || !responses[1].IsLastPackage || responses[1].PayloadMsg.Result.Text != "一二" {
		t.Fatalf("unexpected responses: %+v", responses)
	}
	if err := client.SendAudio(nil, false); err != ErrAudioFinished {
		t.Fatalf("send after final error = %v", err)
	}
}

func testServerResponse(flags MessageTypeSpecificFlags, last bool, payload string) []byte {
	message := []byte{0x11, byte(SERVER_FULL_RESPONSE<<4) | byte(flags), 0x10, 0x00}
	if flags&0x01 != 0 {
		message = binary.BigEndian.AppendUint32(message, 1)
	}
	if last {
		message[1] |= byte(NEG_SEQUENCE)
	}
	message = binary.BigEndian.AppendUint32(message, uint32(len(payload)))
	return append(message, payload...)
}
