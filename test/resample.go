package main

import (
	"github.com/realzhangm/xaux/pkg/resample"
	"os"
)

func resampleTest() {
	buf48, err := os.ReadFile("48.pcm")
	if err != nil {
		panic(err)
	}
	buf16, err := resample.R48kTO16k(buf48)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("16.pcm", buf16, os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func main() {
	resampleTest()
}
