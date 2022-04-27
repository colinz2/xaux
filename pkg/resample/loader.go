package resample

import (
	"plugin"
	"runtime"
)

var (
	R48kTO16k func([]byte) ([]byte, error)
)

func init() {
	var goaway = false
	switch runtime.GOOS {
	case "linux", "darwin":
		goaway = false
	default:
		goaway = true
	}

	if goaway {
		return
	}

	p, err := plugin.Open("resample.so")
	if err != nil {
		panic(err)
	}
	s, err := p.Lookup("ReSample4816")
	if err != nil {
		panic(err)
	}
	var ok bool = false
	R48kTO16k, ok = s.(func([]byte) ([]byte, error))
	if !ok {
		panic("ReSample4816 load fail!")
	}
	if R48kTO16k == nil {
		panic("ReSample4816 is nil")
	}
}
