package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"xaux/pkg/doa"
)

type SoundCap struct {
	buff         *bytes.Buffer
	cmd          *exec.Cmd
	channelNum   int
	sampleRate   int
	bitPerSample int
}

func NewSoundCap(ctx context.Context) *SoundCap {
	sc := &SoundCap{
		buff:         &bytes.Buffer{},
		channelNum:   2,
		sampleRate:   48000,
		bitPerSample: 16,
	}
	sc.cmd = exec.CommandContext(ctx, "winscap.exe", "2", "48000", "16")
	doa.MustTrue(sc.cmd != nil, "sc.cmd is nil")

	sc.cmd.Stdout = sc
	return sc
}

func (s SoundCap) getMillisecond(len int) int {
	bytesPerMilli := (s.sampleRate / 1000) * s.bitPerSample * s.channelNum
	if bytesPerMilli > len {
		return 0
	}
	return len / bytesPerMilli
}

func (s *SoundCap) Write(p []byte) (n int, err error) {
	dataLen := len(p)
	doa.MustTrue(dataLen%2 == 0, "sample not even")
	fmt.Println("len=", s.getMillisecond(dataLen))
	for i := 0; i < dataLen; i += 4 {
		s.buff.Write(p[i : i+2])
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

func (s *SoundCap) Dump(fileName string) {
	err := os.WriteFile(fileName, s.buff.Bytes(), os.ModePerm)
	if err != nil {
		panic(err)
	}
}
