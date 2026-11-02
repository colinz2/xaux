package main

import "flag"

var proxyAddr = flag.String("aa", "127.0.0.1:11024", "asr ai_proxy address")

func main() {
	flag.Parse()
}
