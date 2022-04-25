package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"xaux/pkg/doa"
	"xaux/pkg/x"
	"xaux/pkg/x/client"
)

type SoundCap struct {
	buff          *bytes.Buffer
	cmd           *exec.Cmd
	channelNum    int
	sampleRate    int
	bitsPerSample int
	asrClient     *client.Client
}

func toString(n int) string {
	return fmt.Sprintf("%d", n)
}

func NewSoundCap(ctx context.Context, proxyAddr string) (*SoundCap, error) {
	sc := &SoundCap{
		buff:          &bytes.Buffer{},
		channelNum:    2,
		sampleRate:    48000,
		bitsPerSample: 16,
	}

	var err error = nil
	sc.asrClient, err = client.NewClient(proxyAddr)
	if err != nil {
		return nil, err
	}
	err = sc.asrClient.Start(x.StartConfig{
		SampleRate:    int32(sc.sampleRate),
		BitsPerSample: int32(sc.bitsPerSample),
	})
	if err != nil {
		return nil, err
	}

	sc.cmd = exec.CommandContext(ctx, "winscap.exe",
		toString(sc.channelNum), toString(sc.sampleRate), toString(sc.bitsPerSample))
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
	fmt.Println("duration=", s.getMillisecond(dataLen))
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
