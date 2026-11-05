package doubao

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/realzhangm/xaux/pkg/ai_proxy/asr/doubao/protocol"
	"github.com/realzhangm/xaux/pkg/resample"
	"github.com/realzhangm/xaux/pkg/x"
)

const (
	DefaultWSURL      = protocol.DefaultWSURL
	DefaultResourceID = protocol.DefaultResourceID
	connectTimeout    = 20 * time.Second
	// audioQueueSize bounds the audio chunks waiting for the websocket
	// sender; at roughly 45 ms of 16 kHz audio per chunk this is ~11 s.
	audioQueueSize = 256
)

type Config = protocol.Config

var _ x.ISession = (*Session)(nil)
var _ x.ISessionMaker = (*SessionMaker)(nil)

var (
	ErrAlreadyStarted  = errors.New("doubao ASR session already started")
	ErrAlreadyFinished = errors.New("doubao ASR session is not started or already finished")
)

func ConfigFromEnv() Config {
	return Config{
		WSURL:      envOrDefault("DOUBAO_WS_URL", DefaultWSURL),
		APIKey:     os.Getenv("DOUBAO_API_KEY"),
		ResourceID: envOrDefault("DOUBAO_RESOURCE_ID", DefaultResourceID),
	}
}

func envOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

type SessionMaker struct {
	cnt    uint32
	config Config
}

func NewSessionMaker() (*SessionMaker, error) {
	return NewSessionMakerWithConfig(ConfigFromEnv())
}

func NewSessionMakerWithConfig(config Config) (*SessionMaker, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &SessionMaker{config: config}, nil
}

func (s *SessionMaker) MakeSession(response x.IResponse) (x.ISession, error) {
	if response == nil {
		return nil, errors.New("doubao ASR response writer is nil")
	}
	id := atomic.AddUint32(&s.cnt, 1) - 1
	return &Session{
		id:              id,
		netRsp:          response,
		config:          s.config,
		seenUtterances:  make(map[int]bool),
		finalUtterances: make(map[int]bool),
	}, nil
}

type sessionStatus uint8

const (
	statusNone sessionStatus = iota
	statusStarting
	statusStarted
	statusFinished
	statusFailed
	statusClosed
)

type Session struct {
	id     uint32
	netRsp x.IResponse
	config Config

	mu            sync.Mutex
	status        sessionStatus
	startConfig   x.StartConfig
	client        *protocol.AsrWsClient
	done          chan error
	sendDone      chan error
	audioCh       chan []byte
	droppedChunks uint64
	streamErr     error
	recorder      *x.AudioRecorder

	// Set before the worker goroutines start; only the websocket receive
	// goroutine accesses these fields afterwards.
	logDebug        bool
	seenUtterances  map[int]bool
	finalUtterances map[int]bool
	fallbackSeen    bool
	fallbackFinal   bool
	fallbackText    string
}

func (s *Session) ID() uint32 {
	return s.id
}

func (s *Session) SetAudioRecorder(recorder *x.AudioRecorder) {
	s.mu.Lock()
	s.recorder = recorder
	s.mu.Unlock()
}

// sendLoop forwards queued audio to Doubao from a single goroutine so a slow
// websocket write never blocks the shared UDP receive loop. The most recent
// chunk is held back so it can carry the negative final sequence number; when
// no audio arrived the final packet is empty.
func (s *Session) sendLoop(client *protocol.AsrWsClient, audioCh <-chan []byte, sendDone chan<- error, recorder *x.AudioRecorder) {
	var pending []byte
	for chunk := range audioCh {
		if pending != nil {
			if err := client.SendAudio(pending, false); err != nil {
				s.failStream(err)
				log.Printf("doubao session %d send error: %v", s.id, err)
				sendDone <- err
				return
			}
			if recorder != nil {
				recorder.WriteASR(pending)
			}
		}
		pending = chunk
	}

	// audioCh is closed: the held chunk carries the final sequence number.
	err := client.SendAudio(pending, true)
	if err != nil {
		s.failStream(err)
		log.Printf("doubao session %d send final packet: %v", s.id, err)
	} else if recorder != nil {
		recorder.WriteASR(pending)
	}
	sendDone <- err
}

func (s *Session) failStream(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.streamErr == nil {
		s.streamErr = err
	}
	if s.status == statusStarted {
		s.status = statusFailed
	}
}

func (s *Session) onCmdStart(startConfig x.StartConfig) error {
	if err := s.beginStart(startConfig); err != nil {
		return err
	}
	s.mu.Lock()
	recorder := s.recorder
	s.mu.Unlock()
	debug := recorder != nil
	if debug {
		log.Printf("doubao session %d: connecting ASR, sample_rate=%d bits=%d", s.id, startConfig.SampleRate, startConfig.BitsPerSample)
	}

	client := protocol.NewAsrWsClient(s.config.WSURL)
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	payload := s.requestPayload()
	if err := client.Start(ctx, s.config, payload); err != nil {
		s.mu.Lock()
		if s.status == statusStarting {
			s.status = statusNone
		}
		s.mu.Unlock()
		return err
	}

	done := make(chan error, 1)
	sendDone := make(chan error, 1)
	audioCh := make(chan []byte, audioQueueSize)
	s.mu.Lock()
	if s.status == statusClosed {
		s.mu.Unlock()
		client.Close()
		return errors.New("doubao ASR session closed while connecting")
	}
	s.client = client
	s.done = done
	s.sendDone = sendDone
	s.audioCh = audioCh
	s.logDebug = debug
	s.status = statusStarted
	s.mu.Unlock()

	go s.sendLoop(client, audioCh, sendDone, recorder)
	go s.receive(client, done)
	if debug {
		log.Printf("doubao session %d: ASR connected", s.id)
	}
	return nil
}

func (s *Session) beginStart(config x.StartConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != statusNone {
		return ErrAlreadyStarted
	}
	if err := validateStartConfig(config); err != nil {
		return err
	}
	s.status = statusStarting
	s.startConfig = config
	return nil
}

func validateStartConfig(config x.StartConfig) error {
	if config.SampleRate != 16000 && config.SampleRate != 48000 {
		return fmt.Errorf("unsupported sample rate %d: expected 16000 or 48000", config.SampleRate)
	}
	if config.BitsPerSample != 0 && config.BitsPerSample != 16 {
		return fmt.Errorf("unsupported bits per sample %d: expected 16", config.BitsPerSample)
	}
	if config.SampleRate == 48000 && resample.R48kTO16k == nil {
		return errors.New("48 kHz audio requires the resample plugin")
	}
	return nil
}

func (s *Session) requestPayload() protocol.AsrRequestPayload {
	return protocol.AsrRequestPayload{
		User: protocol.UserMeta{Uid: fmt.Sprintf("xaux-%d", s.id)},
		Audio: protocol.AudioMeta{
			Format:  "pcm",
			Codec:   "raw",
			Rate:    protocol.DefaultSampleRate,
			Bits:    16,
			Channel: 1,
		},
		Request: protocol.RequestMeta{
			ModelName:       "bigmodel",
			EnableITN:       true,
			EnablePUNC:      true,
			EnableDDC:       true,
			ShowUtterances:  true,
			EnableNonstream: true,
			ResultType:      "full",
			EndWindowSize:   800,
		},
	}
}

func (s *Session) receive(client *protocol.AsrWsClient, done chan<- error) {
	err := client.Receive(s.handleResponse)
	if err != nil {
		s.mu.Lock()
		if s.streamErr == nil {
			s.streamErr = err
		}
		if s.status == statusStarted {
			s.status = statusFailed
		}
		shouldNotify := s.status != statusClosed
		s.mu.Unlock()
		if shouldNotify {
			log.Printf("doubao session %d receive error: %v", s.id, err)
			stop := &x.StopResponse{Type: x.TypeStop, Error: x.Error{Msg: err.Error()}}
			if writeErr := s.writeJSON(stop); writeErr != nil {
				log.Printf("doubao session %d: notify client of ASR failure: %v", s.id, writeErr)
			}
		}
	}
	done <- err
}

func (s *Session) onCmdEnd() error {
	s.mu.Lock()
	if s.status == statusFailed {
		err := s.streamErr
		client := s.client
		s.mu.Unlock()
		if client != nil {
			_ = client.Close()
		}
		return err
	}
	if s.status != statusStarted {
		s.mu.Unlock()
		return ErrAlreadyFinished
	}
	s.status = statusFinished
	client := s.client
	done := s.done
	sendDone := s.sendDone
	// DataCb stops queueing once the status leaves statusStarted, so closing
	// the channel here cannot race with a send into it.
	close(s.audioCh)
	s.mu.Unlock()
	defer func() { _ = client.Close() }()

	select {
	case err := <-sendDone:
		if err != nil {
			return err
		}
	case <-time.After(connectTimeout):
		return errors.New("timed out sending the final Doubao ASR audio packet")
	}

	select {
	case err := <-done:
		return err
	case <-time.After(connectTimeout):
		return errors.New("timed out waiting for the final Doubao ASR response")
	}
}

func (s *Session) CloseAll() {
	s.mu.Lock()
	if s.status == statusClosed {
		s.mu.Unlock()
		return
	}
	if s.status == statusStarted {
		// Let sendLoop exit; its final write fails once the client closes.
		close(s.audioCh)
	}
	s.status = statusClosed
	client := s.client
	s.mu.Unlock()
	if client != nil {
		_ = client.Close()
	}
}

func (s *Session) CommandCb(request *x.AllRequest) error {
	var response interface{}
	switch request.Cmd {
	case x.CmdStart:
		err := s.onCmdStart(request.Config)
		startResponse := x.StartResponse{Type: x.TypeRspStart}
		if err != nil {
			startResponse.Error = x.Error{Msg: err.Error()}
		} else {
			startResponse.SessionID = s.id
			startResponse.UDPPort = x.UDPPort
		}
		response = &startResponse
	case x.CmdEnd:
		err := s.onCmdEnd()
		endResponse := x.EndResponse{Type: x.TypeRspEnd}
		if err != nil {
			endResponse.Error = x.Error{Msg: err.Error()}
		}
		response = &endResponse
	default:
		return fmt.Errorf("unsupported command %q", request.Cmd)
	}
	return s.writeJSON(response)
}

// DataCb runs on the shared UDP receive loop, so it only converts and queues
// the audio; the blocking websocket write happens in sendLoop.
func (s *Session) DataCb(data []byte, _ uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.status != statusStarted {
		return
	}

	pcm16k := data
	if s.startConfig.SampleRate == 48000 {
		var err error
		pcm16k, err = resample.R48kTO16k(data)
		if err != nil {
			log.Printf("doubao session %d resample error: %v", s.id, err)
			return
		}
	}

	// Copy because the caller reuses its UDP read buffer.
	chunk := append([]byte(nil), pcm16k...)
	select {
	case s.audioCh <- chunk:
	default:
		s.droppedChunks++
		if s.droppedChunks == 1 || s.droppedChunks%100 == 0 {
			log.Printf("doubao session %d: audio queue full, dropped %d chunks", s.id, s.droppedChunks)
		}
	}
}

func (s *Session) handleResponse(response *protocol.AsrResponse) error {
	if response.PayloadMsg == nil {
		return nil
	}
	result := response.PayloadMsg.Result
	if s.logDebug {
		log.Printf("doubao session %d: ASR response last=%t text_bytes=%d utterances=%d log_id=%s", s.id, response.IsLastPackage, len(result.Text), len(result.Utterances), result.Additions.LogID)
	}
	if len(result.Utterances) == 0 {
		return s.handleFallbackResult(response, result.Text)
	}

	for index, utterance := range result.Utterances {
		if !s.seenUtterances[index] {
			s.seenUtterances[index] = true
			start := &x.SentenceStartResponse{Type: x.TypeSentenceStart}
			start.StartResult.Index = index
			start.StartResult.Time = utterance.StartTime
			if err := s.writeJSON(start); err != nil {
				return err
			}
		}
		if utterance.Definite && s.finalUtterances[index] {
			continue
		}

		responseType := x.TypeRecognizing
		if utterance.Definite {
			responseType = x.TypeSentenceEnd
			s.finalUtterances[index] = true
		}
		recognition := &x.RecognizingResponse{Type: responseType}
		recognition.Result.Interim = !utterance.Definite
		recognition.Result.Index = index
		recognition.Result.Time = utterance.EndTime
		recognition.Result.Result = utterance.Text
		recognition.Result.Words = make([]x.Words, 0, len(utterance.Words))
		for _, word := range utterance.Words {
			recognition.Result.Words = append(recognition.Result.Words, x.Words{
				Text:      word.Text,
				StartTime: word.StartTime,
				EndTime:   word.EndTime,
			})
		}
		if len(recognition.Result.Words) == 0 {
			recognition.Result.Words = nil
		}
		if err := s.writeJSON(recognition); err != nil {
			return err
		}
	}
	return nil
}

// handleFallbackResult supports responses that only contain aggregate text.
func (s *Session) handleFallbackResult(response *protocol.AsrResponse, text string) error {
	if text != "" {
		s.fallbackText = text
	}
	if s.fallbackText == "" || s.fallbackFinal {
		return nil
	}
	if !s.fallbackSeen {
		s.fallbackSeen = true
		start := &x.SentenceStartResponse{Type: x.TypeSentenceStart}
		if err := s.writeJSON(start); err != nil {
			return err
		}
	}

	responseType := x.TypeRecognizing
	if response.IsLastPackage {
		responseType = x.TypeSentenceEnd
		s.fallbackFinal = true
	}
	recognition := &x.RecognizingResponse{Type: responseType}
	recognition.Result.Interim = !response.IsLastPackage
	recognition.Result.Result = s.fallbackText
	return s.writeJSON(recognition)
}

func (s *Session) writeJSON(value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.netRsp.Write(data)
	return err
}
