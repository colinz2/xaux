/*
	Copyright (C) 2016 - 2018, Lefteris Zafiris <zaf@fastmail.com>

	This program is free software, distributed under the terms of
	the BSD 3-Clause License. See the LICENSE file
	at the top of the source tree.
*/

// Package resample provides PCM audio resampling.
package resample

/*
#cgo LDFLAGS: -L"../../../local/lib" -lsoxr
#cgo CFLAGS: -I"../../../local/include"

#include <stdlib.h>
#include "soxr.h"
*/
import "C"

import (
	"errors"
	"io"
	"runtime"
)

const (
	// Quality settings
	Quick     = 0 // Quick cubic interpolation
	LowQ      = 1 // LowQ 16-bit with larger rolloff
	MediumQ   = 2 // MediumQ 16-bit with medium rolloff
	HighQ     = 4 // High quality
	VeryHighQ = 6 // Very high quality

	// Input formats
	F32 = 0 // 32-bit floating point PCM
	F64 = 1 // 64-bit floating point PCM
	I32 = 2 // 32-bit signed linear PCM
	I16 = 3 // 16-bit signed linear PCM

	byteLen = 8
)

// Resampler resamples PCM sound data.
type Resampler struct {
	resampler   C.soxr_t
	inRate      float64
	outRate     float64
	channels    int
	frameSize   int
	destination io.Writer
}

var threads = runtime.NumCPU()

// New returns a pointer to a Resampler that implements an io.WriteCloser.
// It takes as parameters the destination data Writer, the input and output
// sampling rates, the number of channels of the input data, the input format
// and the quality setting.
func New(writer io.Writer, inputRate, outputRate float64, channels, format, quality int) (*Resampler, error) {
	if writer == nil {
		return nil, errors.New("io.Writer is nil")
	}
	if inputRate <= 0 || outputRate <= 0 {
		return nil, errors.New("invalid input or output sampling rates")
	}
	if channels <= 0 {
		return nil, errors.New("invalid channels number")
	}
	if quality < 0 || quality > 6 {
		return nil, errors.New("invalid quality setting")
	}

	var size int
	switch format {
	case F64:
		size = 64 / byteLen
	case F32, I32:
		size = 32 / byteLen
	case I16:
		size = 16 / byteLen
	default:
		return nil, errors.New("invalid format setting")
	}

	ioSpec := C.soxr_io_spec(C.soxr_datatype_t(format), C.soxr_datatype_t(format))
	qSpec := C.soxr_quality_spec(C.ulong(quality), 0)
	runtimeSpec := C.soxr_runtime_spec(C.uint(threads))
	var soxErr C.soxr_error_t
	soxr := C.soxr_create(C.double(inputRate), C.double(outputRate), C.uint(channels), &soxErr, &ioSpec, &qSpec, &runtimeSpec)
	if err := newSoxrError(soxErr); err != nil {
		if soxr != nil {
			C.soxr_delete(soxr)
		}
		return nil, err
	}
	if soxr == nil {
		return nil, errors.New("soxr_create returned nil")
	}

	return &Resampler{
		resampler:   soxr,
		inRate:      inputRate,
		outRate:     outputRate,
		channels:    channels,
		frameSize:   size,
		destination: writer,
	}, nil
}

// Reset permits reusing a Resampler rather than allocating a new one.
func (r *Resampler) Reset(writer io.Writer) error {
	if r.resampler == nil {
		return errors.New("soxr resampler is nil")
	}
	if writer == nil {
		return errors.New("io.Writer is nil")
	}
	r.destination = writer
	return newSoxrError(C.soxr_clear(r.resampler))
}

// Close clean-ups and frees memory. Should always be called when
// finished using the resampler.
func (r *Resampler) Close() error {
	if r.resampler == nil {
		return errors.New("soxr resampler is nil")
	}
	C.soxr_delete(r.resampler)
	r.resampler = nil
	return nil
}

// Write resamples PCM sound data. Writes len(p) bytes from p to
// the underlying data stream, returns the number of bytes written
// from p (0 <= n <= len(p)) and any error encountered that caused
// the write to stop early.
func (r *Resampler) Write(p []byte) (int, error) {
	if r.resampler == nil {
		return 0, errors.New("soxr resampler is nil")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if fragment := len(p) % (r.frameSize * r.channels); fragment != 0 {
		p = p[:len(p)-fragment]
	}
	framesIn := len(p) / r.frameSize / r.channels
	if framesIn == 0 {
		return 0, errors.New("incomplete input frame data")
	}
	framesOut := int(float64(framesIn) * (r.outRate / r.inRate))
	if framesOut == 0 {
		return 0, errors.New("not enough input to generate output")
	}

	dataIn := C.CBytes(p)
	defer C.free(dataIn)
	dataOut := C.malloc(C.size_t(framesOut * r.channels * r.frameSize))
	if dataOut == nil {
		return 0, errors.New("failed to allocate resampling output buffer")
	}
	defer C.free(dataOut)

	var read, done C.size_t
	for int(done) < framesOut {
		soxErr := C.soxr_process(r.resampler, C.soxr_in_t(dataIn), C.size_t(framesIn), &read, C.soxr_out_t(dataOut), C.size_t(framesOut), &done)
		if err := newSoxrError(soxErr); err != nil {
			return 0, err
		}
		if int(read) == framesIn && int(done) < framesOut {
			var flushed C.size_t
			soxErr = C.soxr_process(r.resampler, C.soxr_in_t(nil), 0, nil, C.soxr_out_t(dataOut), C.size_t(framesOut), &flushed)
			if err := newSoxrError(soxErr); err != nil {
				return 0, err
			}
			done += flushed
			break
		}
	}

	written, writeErr := r.destination.Write(C.GoBytes(dataOut, C.int(int(done)*r.channels*r.frameSize)))
	n := int(float64(written) * (r.inRate / r.outRate))
	if writeErr == nil && framesIn == int(read) && framesOut == int(done) {
		n = len(p)
	}
	return n, writeErr
}

func newSoxrError(soxErr C.soxr_error_t) error {
	if soxErr == nil {
		return nil
	}
	message := C.GoString(soxErr)
	if message == "" || message == "0" {
		return nil
	}
	return errors.New(message)
}
