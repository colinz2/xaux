package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	sc := NewSoundCap(ctx)
	go sc.Run()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGILL, syscall.SIGINT, syscall.SIGQUIT)

	sig := <-sigChan

	cancel()
	fmt.Println("sig = ", sig)
	sc.Dump("xx.pcm")
}
