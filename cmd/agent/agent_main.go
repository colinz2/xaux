package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/realzhangm/xaux/pkg/x"
	"os"
	"os/signal"
	"syscall"
)

import (
	"github.com/realzhangm/xaux/pkg/ffaudio"
	"github.com/realzhangm/xaux/pkg/sound_cap"
)

var (
	proxyAddr   = flag.String("aa", "127.0.0.1:11024", "asr ai_proxy address")
	exeDevParam = ""
)

func init() {
	flag.Parse()
	ffaudio.Init()
	devCap, err := ffaudio.GetDevPlaybackAndCapture()
	if err != nil {
		panic(err)
	}
	index, devType := devCap.FindIndex(devCap.PlayBackDefault)
	if index < 0 {
		panic("index < 0")
	}
	exeDevParam = sound_cap.TransFFMediaDevParam(devType, index)
}

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	sc, err := sound_cap.NewSoundCap(ctx, &sound_cap.Config{
		ProxyAddr:      *proxyAddr,
		ExeDevParam:    exeDevParam,
		RecordFilePath: "",
	}, func(rsp *x.AllResponse) error {
		return nil
	})
	if err != nil {
		panic(err)
	}
	go func() {
		err = sc.Run()
		if err != nil {
			panic(err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-sigChan

	cancel()
	fmt.Println("signal = ", sig)
	sc.Close()
	sc.DumpRecordAudio()
}
