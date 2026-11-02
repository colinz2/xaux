package x

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultRecordDir = "recordings"

type DebugConfig struct {
	Provider  string
	RecordDir string
}

type AudioRecorder struct {
	mu          sync.Mutex
	dir         string
	provider    string
	sessionID   uint32
	startedAt   time.Time
	startConfig StartConfig
	inputFile   *os.File
	asrFile     *os.File
	inputBytes  int64
	asrBytes    int64
	inputStats  pcmStatistics
	asrStats    pcmStatistics
	closed      bool
}

func newAudioRecorder(config DebugConfig, sessionID uint32, startConfig StartConfig) (*AudioRecorder, error) {
	dir := config.RecordDir
	if dir == "" {
		dir = defaultRecordDir
	}
	name := fmt.Sprintf("%s_session_%d_%s", config.Provider, sessionID, time.Now().UTC().Format("20060102T150405.000000000Z"))
	sessionDir := filepath.Join(dir, name)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, err
	}

	r := &AudioRecorder{
		dir:         sessionDir,
		provider:    config.Provider,
		sessionID:   sessionID,
		startedAt:   time.Now().UTC(),
		startConfig: startConfig,
	}
	var err error
	r.inputFile, err = os.OpenFile(filepath.Join(sessionDir, "input.pcm"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	r.asrFile, err = os.OpenFile(filepath.Join(sessionDir, "asr_16k.pcm"), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		r.inputFile.Close()
		return nil, err
	}
	if err := r.writeMetadata(); err != nil {
		r.closeFiles()
		return nil, err
	}
	return r, nil
}

func (r *AudioRecorder) WriteInput(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || len(data) == 0 || r.inputFile == nil {
		return
	}
	n, err := r.inputFile.Write(data)
	r.inputBytes += int64(n)
	r.inputStats.add(data[:n])
	if err != nil {
		log.Printf("debug session %d: write input recording: %v", r.sessionID, err)
		r.inputFile.Close()
		r.inputFile = nil
	}
}

func (r *AudioRecorder) WriteASR(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || len(data) == 0 || r.asrFile == nil {
		return
	}
	n, err := r.asrFile.Write(data)
	r.asrBytes += int64(n)
	r.asrStats.add(data[:n])
	if err != nil {
		log.Printf("debug session %d: write ASR recording: %v", r.sessionID, err)
		r.asrFile.Close()
		r.asrFile = nil
	}
}

func (r *AudioRecorder) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	r.closeFiles()
	if err := r.writeMetadata(); err != nil {
		log.Printf("debug session %d: write recording metadata: %v", r.sessionID, err)
	}
	inputLevel := r.inputStats.level()
	log.Printf("debug session %d: recording saved to %s (input=%d bytes rms=%.1fdBFS, asr=%d bytes rms=%.1fdBFS)", r.sessionID, r.dir, r.inputBytes, inputLevel.RMSDBFS, r.asrBytes, r.asrStats.level().RMSDBFS)
}

func (r *AudioRecorder) closeFiles() {
	if r.inputFile != nil {
		_ = r.inputFile.Close()
		r.inputFile = nil
	}
	if r.asrFile != nil {
		_ = r.asrFile.Close()
		r.asrFile = nil
	}
}

func (r *AudioRecorder) writeMetadata() error {
	metadata := struct {
		Provider    string      `json:"provider"`
		SessionID   uint32      `json:"sessionID"`
		StartedAt   time.Time   `json:"startedAt"`
		EndedAt     time.Time   `json:"endedAt,omitempty"`
		StartConfig StartConfig `json:"startConfig"`
		InputFile   string      `json:"inputFile"`
		ASRFile     string      `json:"asrFile"`
		InputBytes  int64       `json:"inputBytes"`
		ASRBytes    int64       `json:"asrBytes"`
		InputAudio  audioLevel  `json:"inputAudio"`
		ASRAudio    audioLevel  `json:"asrAudio"`
		PCMFormat   string      `json:"pcmFormat"`
	}{
		Provider:    r.provider,
		SessionID:   r.sessionID,
		StartedAt:   r.startedAt,
		StartConfig: r.startConfig,
		InputFile:   "input.pcm",
		ASRFile:     "asr_16k.pcm",
		InputBytes:  r.inputBytes,
		ASRBytes:    r.asrBytes,
		InputAudio:  r.inputStats.level(),
		ASRAudio:    r.asrStats.level(),
		PCMFormat:   "signed 16-bit little-endian PCM; input uses startConfig.sampleRate and client-defined channels; asr_16k.pcm is mono 16 kHz",
	}
	if r.closed {
		metadata.EndedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.dir, "session.json"), data, 0600)
}

func (r *AudioRecorder) inputLevel() audioLevel {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inputStats.level()
}

type audioRecorderSetter interface {
	SetAudioRecorder(*AudioRecorder)
}

type DebugSessionMaker struct {
	maker  ISessionMaker
	config DebugConfig
}

func NewDebugSessionMaker(maker ISessionMaker, config DebugConfig) *DebugSessionMaker {
	return &DebugSessionMaker{maker: maker, config: config}
}

func (m *DebugSessionMaker) MakeSession(response IResponse) (ISession, error) {
	session, err := m.maker.MakeSession(response)
	if err != nil {
		return nil, err
	}
	return &debugSession{ISession: session, config: m.config}, nil
}

type debugSession struct {
	ISession
	config         DebugConfig
	mu             sync.Mutex
	recorder       *AudioRecorder
	nextSeq        uint32
	hasSeq         bool
	packets        uint64
	receivedBytes  uint64
	lost           uint64
	reordered      uint64
	lowLevelWarned bool
}

func (s *debugSession) CommandCb(request *AllRequest) error {
	if request.Cmd == CmdStart {
		s.mu.Lock()
		needsRecorder := s.recorder == nil
		s.mu.Unlock()
		if needsRecorder {
			if err := s.startRecording(request.Config); err != nil {
				log.Printf("debug session %d: start recording: %v", s.ID(), err)
			}
		}
	}
	err := s.ISession.CommandCb(request)
	if request.Cmd == CmdEnd && err == nil {
		s.closeRecorder()
	}
	return err
}

func (s *debugSession) startRecording(config StartConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recorder != nil {
		s.recorder.Close()
		s.recorder = nil
	}
	recorder, err := newAudioRecorder(s.config, s.ID(), config)
	if err != nil {
		return err
	}
	s.recorder = recorder
	s.hasSeq = false
	s.nextSeq = 0
	s.packets = 0
	s.receivedBytes = 0
	s.lost = 0
	s.reordered = 0
	s.lowLevelWarned = false
	if setter, ok := s.ISession.(audioRecorderSetter); ok {
		setter.SetAudioRecorder(recorder)
	}
	log.Printf("debug session %d: recording to %s", s.ID(), recorder.dir)
	return nil
}

func (s *debugSession) DataCb(data []byte, seq uint32) {
	s.mu.Lock()
	recorder := s.recorder
	s.trackSequenceLocked(seq)
	s.receivedBytes += uint64(len(data))
	packets, receivedBytes, lost, reordered := s.packets, s.receivedBytes, s.lost, s.reordered
	checkLevel := packets == 100 && !s.lowLevelWarned
	if checkLevel {
		s.lowLevelWarned = true
	}
	s.mu.Unlock()

	if recorder != nil {
		recorder.WriteInput(data)
	}
	if checkLevel && recorder != nil {
		level := recorder.inputLevel()
		if level.RMSDBFS < -50 {
			log.Printf("debug session %d: CLIENT AUDIO WARNING input RMS=%.1fdBFS peak=%.1fdBFS; check microphone, gain and channel selection", s.ID(), level.RMSDBFS, level.PeakDBFS)
		}
	}
	if packets%100 == 0 {
		log.Printf("debug session %d: UDP packets=%d seq=%d bytes=%d lost=%d reordered=%d", s.ID(), packets, seq, receivedBytes, lost, reordered)
	}
	s.ISession.DataCb(data, seq)
}

func (s *debugSession) trackSequenceLocked(seq uint32) {
	s.packets++
	if !s.hasSeq {
		s.hasSeq = true
		s.nextSeq = seq + 1
		return
	}
	if seq > s.nextSeq {
		gap := uint64(seq - s.nextSeq)
		s.lost += gap
		log.Printf("debug session %d: UDP sequence gap expected=%d got=%d missing=%d", s.ID(), s.nextSeq, seq, gap)
		s.nextSeq = seq + 1
		return
	}
	if seq < s.nextSeq {
		s.reordered++
		log.Printf("debug session %d: UDP duplicate/out-of-order expected=%d got=%d", s.ID(), s.nextSeq, seq)
		return
	}
	s.nextSeq++
}

func (s *debugSession) CloseAll() {
	s.ISession.CloseAll()
	s.closeRecorder()
}

func (s *debugSession) closeRecorder() {
	s.mu.Lock()
	recorder := s.recorder
	if recorder == nil {
		s.mu.Unlock()
		return
	}
	s.recorder = nil
	packets, receivedBytes, lost, reordered := s.packets, s.receivedBytes, s.lost, s.reordered
	s.mu.Unlock()
	recorder.Close()
	log.Printf("debug session %d: UDP summary packets=%d bytes=%d lost=%d reordered=%d", s.ID(), packets, receivedBytes, lost, reordered)
}
