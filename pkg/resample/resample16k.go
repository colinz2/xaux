package resample

import (
	"bytes"
	"errors"
	"sync"

	"github.com/realzhangm/xaux/pkg/resample/resample"
)

type ReSampler struct {
	res  *resample.Resampler
	buff *bytes.Buffer
}

var res4816Pool = sync.Pool{New: func() any {
	resampler := &ReSampler{buff: new(bytes.Buffer)}
	var err error
	resampler.res, err = resample.New(resampler.buff,
		48000, 16000, 2, resample.I16, resample.HighQ)
	if err != nil {
		return nil
	}
	return resampler
}}

func getRes4816() *ReSampler {
	value := res4816Pool.Get()
	if value == nil {
		return nil
	}
	return value.(*ReSampler)
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
	if res == nil {
		return nil, errors.New("resampler is nil")
	}
	defer putRes4816(res)
	_, err := res.res.Write(input)
	if err != nil {
		return nil, err
	}
	output := make([]byte, res.buff.Len())
	copy(output, res.buff.Bytes())
	return output, nil
}
