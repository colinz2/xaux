package ffaudio

/*
#include "ffaudio/audio.h"
#include "ffaudio/wasapi.c"

ffaudio_init_conf* NewConfigFile() {
	return (ffaudio_init_conf*)malloc(sizeof(ffaudio_init_conf));
}

void init(ffaudio_init_conf *conf) {
	ffaudio_default_interface()->init(conf);
}

void uninit() {
	ffaudio_default_interface()->uninit();
}

ffaudio_dev* dev_alloc(ffuint mode) {
	return ffaudio_default_interface()->dev_alloc(mode);
}

void dev_free(ffaudio_dev *d) {
	return ffaudio_default_interface()->dev_free(d);
}

const char* dev_error(ffaudio_dev *d) {
	return ffaudio_default_interface()->dev_error(d);
}

int dev_next(ffaudio_dev *d) {
	return ffaudio_default_interface()->dev_next(d);
}

const char* dev_info(ffaudio_dev *d, ffuint i) {
	return ffaudio_default_interface()->dev_info(d, i);
}

wchar_t* dev_info_DEV_ID(ffaudio_dev *d) {
	return (wchar_t*)(ffaudio_default_interface()->dev_info(d, FFAUDIO_DEV_ID));
}

ffuint dev_info_MIX_FORMAT_0(ffaudio_dev *d) {
	ffuint* a = (ffuint*)ffaudio_default_interface()->dev_info(d, FFAUDIO_DEV_MIX_FORMAT);
	return a[0];
}
ffuint dev_info_MIX_FORMAT_1(ffaudio_dev *d) {
	ffuint* a = (ffuint*)ffaudio_default_interface()->dev_info(d, FFAUDIO_DEV_MIX_FORMAT);
	return a[1];}
ffuint dev_info_MIX_FORMAT_2(ffaudio_dev *d) {
	ffuint* a = (ffuint*)ffaudio_default_interface()->dev_info(d, FFAUDIO_DEV_MIX_FORMAT);
	return a[2];}

*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

type DevInfo struct {
	Name       string
	IndexStr   string
	IsDefault  bool
	Format     int
	SampleRate int
	Channels   int
}

var (
	ErrFFAudio    = errors.New("ffAudio error")
	ErrFFAudioDev = fmt.Errorf("%w, dev error", ErrFFAudio)
)

func Init() {
	conf := &C.ffaudio_init_conf{}
	C.init(conf)
}

func UnInit() {
	C.uninit()
}

func ListDevPlayback() ([]DevInfo, error) {
	return ListDev(C.FFAUDIO_DEV_PLAYBACK)
}

func ListDevCapture() ([]DevInfo, error) {
	return ListDev(C.FFAUDIO_DEV_CAPTURE)
}

func ListDev(mode C.ffuint) ([]DevInfo, error) {
	var devs []DevInfo
	var err error
	d := C.dev_alloc(mode)
	if d == nil {
		return devs, ErrFFAudioDev
	}

	for {
		r := C.dev_next(d)
		if r > 0 {
			break
		} else if r < 0 {
			C.dev_free(d)
			var errStr string = C.GoString(C.dev_error(d))
			return nil, fmt.Errorf("%w,%s", ErrFFAudioDev, errStr)
		}
		indexStr := DevInfoFormat(unsafe.Pointer(C.dev_info_DEV_ID(d)))

		dev := DevInfo{
			Name:       C.GoString(C.dev_info(d, C.FFAUDIO_DEV_NAME)),
			IndexStr:   indexStr,
			Format:     int(C.dev_info_MIX_FORMAT_0(d)),
			SampleRate: int(C.dev_info_MIX_FORMAT_1(d)),
			Channels:   int(C.dev_info_MIX_FORMAT_2(d)),
		}
		devs = append(devs, dev)
	}

	C.dev_free(d)
	return devs, err
}
