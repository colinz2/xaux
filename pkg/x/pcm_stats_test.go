package x

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestPCMStatistics(t *testing.T) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], uint16(0))
	binary.LittleEndian.PutUint16(data[2:4], uint16(32767))
	negative := int16(-32768)
	binary.LittleEndian.PutUint16(data[4:6], uint16(negative))
	binary.LittleEndian.PutUint16(data[6:8], uint16(100))

	var stats pcmStatistics
	stats.add(data)
	level := stats.level()
	if level.Samples != 4 {
		t.Fatalf("samples = %d, want 4", level.Samples)
	}
	if math.Abs(level.PeakDBFS) > 0.001 {
		t.Fatalf("peak = %f dBFS, want approximately 0", level.PeakDBFS)
	}
	if level.RMSDBFS < -4 || level.RMSDBFS > -2 {
		t.Fatalf("RMS = %f dBFS", level.RMSDBFS)
	}
}

func TestPCMStatisticsSilence(t *testing.T) {
	var stats pcmStatistics
	stats.add([]byte{0, 0, 0, 0})
	level := stats.level()
	if level.Samples != 2 || level.RMSDBFS != silenceDBFS || level.PeakDBFS != silenceDBFS {
		t.Fatalf("silence level = %+v", level)
	}
}
