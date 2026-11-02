package x

import (
	"encoding/binary"
	"math"
)

const silenceDBFS = -160.0

type audioLevel struct {
	Samples  uint64  `json:"samples"`
	RMSDBFS  float64 `json:"rmsDBFS"`
	PeakDBFS float64 `json:"peakDBFS"`
}

type pcmStatistics struct {
	samples    uint64
	sumSquares float64
	peak       int32
}

func (s *pcmStatistics) add(data []byte) {
	for i := 0; i+1 < len(data); i += 2 {
		sample := int32(int16(binary.LittleEndian.Uint16(data[i : i+2])))
		absolute := sample
		if absolute < 0 {
			absolute = -absolute
		}
		if absolute > s.peak {
			s.peak = absolute
		}
		s.sumSquares += float64(sample) * float64(sample)
		s.samples++
	}
}

func (s *pcmStatistics) level() audioLevel {
	level := audioLevel{Samples: s.samples, RMSDBFS: silenceDBFS, PeakDBFS: silenceDBFS}
	if s.samples == 0 {
		return level
	}
	if s.sumSquares > 0 {
		rms := math.Sqrt(s.sumSquares / float64(s.samples))
		level.RMSDBFS = 20 * math.Log10(rms/32768)
	}
	if s.peak > 0 {
		level.PeakDBFS = 20 * math.Log10(float64(s.peak)/32768)
	}
	return level
}
