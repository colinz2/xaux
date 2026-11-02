package doa

import (
	"errors"
	"io"
)

func MustTrue(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}

func PanicExceptIOEOF(err error) {
	if err != nil && !errors.Is(err, io.EOF) {
		panic(err)
	}
}
