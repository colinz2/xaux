package doa

import "io"

func MustTrue(m bool, msg string) {
	if m == false {
		panic(msg)
	}
}

func PanicExceptIOEOF(err error) {
	if err != io.EOF {
		panic(err)
	}
}
