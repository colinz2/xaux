package sound_cap

import (
	"bytes"
	"context"
	"fmt"
	"github.com/realzhangm/xaux/pkg/common/doa"
	"github.com/realzhangm/xaux/pkg/common/path"
	"github.com/realzhangm/xaux/pkg/x"
	"os"
	"os/exec"
	"sync/atomic"
)

type AsrResultCallBack func(rsp *x.AllResponse) error

type SoundCap struct {
	asrClient     *x.Client
	cmd           *exec.Cmd
	buff          *bytes.Buffer
	channelNum    int
	sampleRate    int
	bitsPerSample int
	isClosed      int32
	Config
}

type Config struct {
	ProxyAddr      string
	ExeDevParam    string // --dev-loopback=1
	RecordFilePath string
}

func NewSoundCap(ctx context.Context, config *Config, asrCb AsrResultCallBack) (*SoundCap, error) {
	if len(config.ProxyAddr) == 0 {
		panic("len of config ProxyAddr == 0")
	}
	if len(config.ExeDevParam) == 0 {
		panic("len of config ExeDevParam == 0")
	}

	sc := &SoundCap{
		buff:          &bytes.Buffer{},
		channelNum:    1,
		sampleRate:    16000,
		bitsPerSample: 16,
		isClosed:      0,
		Config:        *config,
	}

	var err error = nil
	sc.asrClient, err = x.NewClient(sc.ProxyAddr)
	if err != nil {
		return nil, err
	}
	err = sc.asrClient.Start(x.StartConfig{
		SampleRate:    int32(sc.sampleRate),
		BitsPerSample: int32(sc.bitsPerSample),
	}, asrCb)
	if err != nil {
		return nil, err
	}

	//sc.cmd = exec.CommandContext(ctx, "winscap.exe",
	//	toString(sc.channelNum), toString(sc.sampleRate), toString(sc.bitsPerSample))
	sc.cmd = exec.CommandContext(ctx, "fmedia",
		fmt.Sprintf("%s", sc.ExeDevParam),
		"--record", "-o", "@stdout.wav",
		"--format=int16",
		"--channels=mono",
		"--capture-buffer=64",
		fmt.Sprintf("--rate=%d", sc.sampleRate),
	)

	doa.MustTrue(sc.cmd != nil, "sc.cmd is nil")

	sc.cmd.Stdout = sc
	return sc, nil
}

func (s SoundCap) getMillisecond(len int) int {
	bytesPerMilli := (s.sampleRate / 1000) * (s.bitsPerSample / 8) * s.channelNum
	if bytesPerMilli > len {
		return 0
	}
	return len / bytesPerMilli
}

func (s *SoundCap) Write(p []byte) (n int, err error) {
	dataLen := len(p)
	doa.MustTrue(dataLen%2 == 0, "sample not even")
	//fmt.Println("duration=", s.getMillisecond(dataLen))
	if s.channelNum == 2 {
		for i := 0; i < dataLen; i += 4 {
			monoData := p[i : i+2]
			_, err := s.buff.Write(monoData)
			if err != nil {
				panic(err)
			}
			err = s.asrClient.Send(monoData)
			if err != nil {
				panic(err)
			}
		}
	} else {
		_, err := s.buff.Write(p)
		if err != nil {
			panic(err)
		}
		err = s.asrClient.Send(p)
		if err != nil {
			panic(err)
		}
	}
	return len(p), nil
}

func (s *SoundCap) Run() error {
	defer s.close()
	err := s.cmd.Run()
	if err != nil {
		return err
	}
	return err
}

func (s *SoundCap) close() {
	if !atomic.CompareAndSwapInt32(&s.isClosed, 0, 1) {
		return
	}
	if s.cmd != nil && s.cmd.Process != nil && s.cmd.ProcessState == nil {
		s.cmd.Process.Kill()
	}
	if s.asrClient != nil {
		s.asrClient.Close()
	}
}

func (s *SoundCap) IsClose() bool {
	return atomic.LoadInt32(&s.isClosed) == 1
}

func (s *SoundCap) Close() {
	s.close()
}

func (s *SoundCap) DumpRecordAudio() {
	if path.Exists(s.RecordFilePath) != nil {
		return
	}

	if err := os.WriteFile(s.RecordFilePath, s.buff.Bytes(), os.ModePerm); err != nil {
		panic(err)
	}
}