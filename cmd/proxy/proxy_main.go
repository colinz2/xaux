package main

import (
	"flag"
	"log"

	"github.com/realzhangm/xaux/pkg/ai_proxy"
)

var (
	asrProvider = flag.String("asr_provider", ai_proxy.ASRProviderDoubao, "ASR provider: aliyun or doubao")
	debug       = flag.Bool("debug", false, "enable debug logs and per-session PCM recording")
	recordDir   = flag.String("record_dir", "recordings", "directory used for PCM recordings in debug mode")
)

func main() {
	flag.Parse()
	proxy, err := ai_proxy.NewProxyWithOptions(*asrProvider, ai_proxy.ProxyOptions{
		Debug:     *debug,
		RecordDir: *recordDir,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := proxy.Start(); err != nil {
		log.Fatal(err)
	}
}
