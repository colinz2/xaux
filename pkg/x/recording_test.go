package x

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type recordingTestSession struct {
	id       uint32
	recorder *AudioRecorder
}

func (s *recordingTestSession) ID() uint32                        { return s.id }
func (s *recordingTestSession) CommandCb(*AllRequest) error       { return nil }
func (s *recordingTestSession) DataCb([]byte, uint32)             {}
func (s *recordingTestSession) CloseAll()                         {}
func (s *recordingTestSession) SetAudioRecorder(r *AudioRecorder) { s.recorder = r }

type recordingTestMaker struct {
	session *recordingTestSession
}

func (m *recordingTestMaker) MakeSession(IResponse) (ISession, error) {
	return m.session, nil
}

func TestDebugSessionRecording(t *testing.T) {
	inner := &recordingTestSession{id: 7}
	maker := NewDebugSessionMaker(&recordingTestMaker{session: inner}, DebugConfig{
		Provider:  "doubao",
		RecordDir: t.TempDir(),
	})
	var output bytes.Buffer
	session, err := maker.MakeSession(&output)
	if err != nil {
		t.Fatal(err)
	}
	debug := session.(*debugSession)
	config := StartConfig{SampleRate: 16000, BitsPerSample: 16}
	if err := session.CommandCb(&AllRequest{Cmd: CmdStart, Config: config}); err != nil {
		t.Fatal(err)
	}
	if inner.recorder == nil {
		t.Fatal("ASR recorder was not configured")
	}
	recordDir := inner.recorder.dir
	if err := session.CommandCb(&AllRequest{Cmd: CmdStart, Config: config}); err != nil {
		t.Fatal(err)
	}
	if inner.recorder.dir != recordDir {
		t.Fatal("duplicate Start reset the active recording")
	}

	session.DataCb([]byte{1, 2}, 1)
	session.DataCb([]byte{3}, 3)
	inner.recorder.WriteASR([]byte{4, 5})
	if err := session.CommandCb(&AllRequest{Cmd: CmdEnd}); err != nil {
		t.Fatal(err)
	}

	input, err := os.ReadFile(filepath.Join(recordDir, "input.pcm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, []byte{1, 2, 3}) {
		t.Fatalf("input recording = %v", input)
	}
	asr, err := os.ReadFile(filepath.Join(recordDir, "asr_16k.pcm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(asr, []byte{4, 5}) {
		t.Fatalf("ASR recording = %v", asr)
	}

	metadataData, err := os.ReadFile(filepath.Join(recordDir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		InputBytes int64      `json:"inputBytes"`
		ASRBytes   int64      `json:"asrBytes"`
		InputAudio audioLevel `json:"inputAudio"`
		ASRAudio   audioLevel `json:"asrAudio"`
		EndedAt    string     `json:"endedAt"`
	}
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.InputBytes != 3 || metadata.ASRBytes != 2 || metadata.InputAudio.Samples != 1 || metadata.ASRAudio.Samples != 1 || metadata.EndedAt == "" {
		t.Fatalf("metadata = %+v", metadata)
	}
	if debug.lost != 1 || debug.reordered != 0 || debug.packets != 2 {
		t.Fatalf("UDP stats: packets=%d lost=%d reordered=%d", debug.packets, debug.lost, debug.reordered)
	}

	// Close is idempotent and writes after close are ignored.
	session.CloseAll()
	inner.recorder.WriteInput([]byte{9})
}
