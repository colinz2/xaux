package resample

import (
	"log"
	"plugin"
	"runtime"
)

var R48kTO16k func([]byte) ([]byte, error)

func init() {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}

	p, err := plugin.Open("resample.so")
	if err != nil {
		log.Printf("resample plugin is unavailable: %v", err)
		return
	}
	symbol, err := p.Lookup("ReSample4816")
	if err != nil {
		log.Printf("resample plugin symbol is unavailable: %v", err)
		return
	}
	resampler, ok := symbol.(func([]byte) ([]byte, error))
	if !ok || resampler == nil {
		log.Printf("resample plugin has an invalid ReSample4816 symbol")
		return
	}
	R48kTO16k = resampler
}
