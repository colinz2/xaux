package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	proxyAddr = flag.String("aa", "192.168.1.100:11024", "asr proxy address")
)

func init() {
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	sc, err := NewSoundCap(ctx, *proxyAddr)
	if err != nil {
		panic(err)
	}
	go sc.Run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-sigChan

	cancel()
	fmt.Println("signal = ", sig)
	sc.Release()
	sc.Dump("xx.pcm")

}
