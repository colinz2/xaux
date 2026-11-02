package doubao

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/realzhangm/xaux/pkg/ai_proxy/asr/doubao/protocol"
	"github.com/realzhangm/xaux/pkg/x"
)

type recordingResponse struct {
	bytes.Buffer
}

func TestHandleResponseMapsAndDeduplicatesUtterances(t *testing.T) {
	writer := &recordingResponse{}
	session := &Session{
		netRsp:          writer,
		seenUtterances:  make(map[int]bool),
		finalUtterances: make(map[int]bool),
	}
	response := &protocol.AsrResponse{PayloadMsg: &protocol.AsrResponsePayload{}}
	response.PayloadMsg.Result.Utterances = []protocol.Utterance{{
		StartTime: 100,
		EndTime:   200,
		Text:      "你",
		Words:     []protocol.Word{{Text: "你", StartTime: 100, EndTime: 200}},
	}}
	if err := session.handleResponse(response); err != nil {
		t.Fatal(err)
	}

	response.PayloadMsg.Result.Utterances[0].Definite = true
	response.PayloadMsg.Result.Utterances[0].Text = "你好"
	response.PayloadMsg.Result.Utterances[0].Words = []protocol.Word{{Text: "你好", StartTime: 100, EndTime: 300}}
	response.PayloadMsg.Result.Utterances[0].EndTime = 300
	if err := session.handleResponse(response); err != nil {
		t.Fatal(err)
	}
	// A definite utterance can be present in subsequent cumulative responses.
	if err := session.handleResponse(response); err != nil {
		t.Fatal(err)
	}

	responses := decodeResponses(t, writer.Bytes())
	if len(responses) != 3 {
		t.Fatalf("response count = %d, want 3: %+v", len(responses), responses)
	}
	if responses[0].Type != x.TypeSentenceStart || responses[0].StartResult.Time != 100 {
		t.Fatalf("sentence start = %+v", responses[0])
	}
	if responses[1].Type != x.TypeRecognizing || !responses[1].Result.Interim || responses[1].Result.Result != "你" {
		t.Fatalf("interim result = %+v", responses[1])
	}
	if responses[2].Type != x.TypeSentenceEnd || responses[2].Result.Interim || responses[2].Result.Result != "你好" {
		t.Fatalf("final result = %+v", responses[2])
	}
	if len(responses[2].Result.Words) != 1 || responses[2].Result.Words[0].EndTime != 300 {
		t.Fatalf("final words = %+v", responses[2].Result.Words)
	}
}

func TestHandleFallbackResultEmitsFinalOnLastPackage(t *testing.T) {
	writer := &recordingResponse{}
	session := &Session{netRsp: writer}
	response := &protocol.AsrResponse{PayloadMsg: &protocol.AsrResponsePayload{}}
	response.PayloadMsg.Result.Text = "中间结果"
	if err := session.handleResponse(response); err != nil {
		t.Fatal(err)
	}

	response.IsLastPackage = true
	response.PayloadMsg.Result.Text = ""
	if err := session.handleResponse(response); err != nil {
		t.Fatal(err)
	}

	responses := decodeResponses(t, writer.Bytes())
	if len(responses) != 3 {
		t.Fatalf("response count = %d, want 3", len(responses))
	}
	if responses[2].Type != x.TypeSentenceEnd || responses[2].Result.Result != "中间结果" || responses[2].Result.Interim {
		t.Fatalf("fallback final result = %+v", responses[2])
	}
}

func TestSessionMakerValidatesAuth(t *testing.T) {
	_, err := NewSessionMakerWithConfig(Config{WSURL: DefaultWSURL, ResourceID: DefaultResourceID})
	if err == nil {
		t.Fatal("expected missing auth to be rejected")
	}
	maker, err := NewSessionMakerWithConfig(Config{
		WSURL: DefaultWSURL, APIKey: "test", ResourceID: DefaultResourceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := maker.MakeSession(&recordingResponse{})
	if err != nil {
		t.Fatal(err)
	}
	second, err := maker.MakeSession(&recordingResponse{})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() != 0 || second.ID() != 1 {
		t.Fatalf("session IDs = %d, %d", first.ID(), second.ID())
	}
}

func decodeResponses(t *testing.T, data []byte) []x.AllResponse {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	var responses []x.AllResponse
	for {
		var response x.AllResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatal(err)
		}
		responses = append(responses, response)
	}
	return responses
}
