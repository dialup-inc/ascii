package capture

/*
#cgo pkg-config: opencv vpx

#include "capture.h"
*/
import "C"
import (
	"unsafe"
)

func Start(width, height int) error {
	ret := C.capture_start(C.int(width), C.int(height))
	if ret != 0 {
		return CaptureError(ret)
	}
	return nil
}

func Stop() error {
	ret := C.capture_stop()
	if ret != 0 {
		return CaptureError(ret)
	}
	return nil
}

func ReadFrame(buf []byte, forceKeyframe bool) (int, error) {
	ptr := (*C.char)(unsafe.Pointer(&buf[0]))
	l := C.int(len(buf))

	var kf C.int
	if forceKeyframe {
		kf = C.int(1)
	}

	n := C.capture_read(ptr, l, kf)
	if n < 0 {
		return 0, CaptureError(n)
	}

	return int(n), nil
}
