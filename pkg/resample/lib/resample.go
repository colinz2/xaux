package main

import (
	"bytes"
	"errors"
	"github.com/zaf/resample"
	"sync"
)

type ReSampler struct {
	res  *resample.Resampler
	buff *bytes.Buffer
}

var (
	res4816Pool = sync.Pool{New: func() any {
		var err error = nil
		res4816 := new(ReSampler)
		res4816.buff = new(bytes.Buffer)
		res4816.res, err = resample.New(res4816.buff,
			48000, 16000, 1, resample.I16, resample.HighQ)
		if err != nil {
			return nil
		}
		return res4816
	}}
)

func getRes4816() *ReSampler {
	return res4816Pool.Get().(*ReSampler)
}

func putRes4816(res *ReSampler) {
	res.buff.Reset()
	if res.res.Reset(res.buff) != nil {
		res.res.Close()
		return
	}
	res4816Pool.Put(res)
}

func ReSample4816(input []byte) ([]byte, error) {
	res := getRes4816()
	defer putRes4816(res)
	if res == nil {
		return nil, errors.New("res is nil")
	}
	_, err := res.res.Write(input)
	if err != nil {
		return nil, err
	}
	output := make([]byte, res.buff.Len())
	copy(output, res.buff.Bytes())
	return output, nil
}
