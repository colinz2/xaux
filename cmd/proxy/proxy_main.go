package main

import (
	"github.com/realzhangm/xaux/pkg/ai_proxy"
)

func init() {
}

func main() {
	proxy, err := ai_proxy.NewProxy()
	if err != nil {
		panic(err)
	}
	err = proxy.Start()
	if err != nil {
		panic(err)
	}
}
