package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/realzhangm/xaux/pkg/doa"
	"github.com/realzhangm/xaux/pkg/x"
	"os"
	"os/exec"
)

type SoundCap struct {
	buff          *bytes.Buffer
	cmd           *exec.Cmd
	channelNum    int
	sampleRate    int
	bitsPerSample int
	asrClient     *x.Client
}

func toString(n int) string {
	return fmt.Sprintf("%d", n)
}

func (s *SoundCap) rspCallBack(rsp *x.AllResponse) error {
	fmt.Println(rsp.Result.Result)
	return nil
}

func NewSoundCap(ctx context.Context, proxyAddr string) (*SoundCap, error) {
	sc := &SoundCap{
		buff:          &bytes.Buffer{},
		channelNum:    1,
		sampleRate:    16000,
		bitsPerSample: 16,
	}

	var err error = nil
	sc.asrClient, err = x.NewClient(proxyAddr)
	if err != nil {
		return nil, err
	}
	err = sc.asrClient.Start(x.StartConfig{
		SampleRate:    int32(sc.sampleRate),
		BitsPerSample: int32(sc.bitsPerSample),
	}, sc.rspCallBack)
	if err != nil {
		return nil, err
	}

	//sc.cmd = exec.CommandContext(ctx, "winscap.exe",
	//	toString(sc.channelNum), toString(sc.sampleRate), toString(sc.bitsPerSample))
	sc.cmd = exec.CommandContext(ctx, "fmedia",
		"--dev-loopback=1",
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
	err := s.cmd.Run()
	if err != nil {
		return err
	}
	return err
}

func (s *SoundCap) Release() {
	s.asrClient.Close()
}

func (s *SoundCap) Dump(fileName string) {
	err := os.WriteFile(fileName, s.buff.Bytes(), os.ModePerm)
	if err != nil {
		panic(err)
	}
}
