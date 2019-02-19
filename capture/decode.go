package capture

/*
#cgo pkg-config: --static vpx

#include "decode.h"
*/
import "C"
import (
	"unsafe"
)

func StartDecode() error {
	ret := C.vpx_init_dec()
	if ret != 0 {
		return CaptureError(ret)
	}
	return nil
}

func StopDecode() error {
	ret := C.vpx_cleanup_dec()
	if ret != 0 {
		return CaptureError(ret)
	}
	return nil
}

func DecodeFrame(out, in []byte) error {
	outP := (*C.char)(unsafe.Pointer(&out[0]))
	outL := C.int(len(out))

	inP := (*C.char)(unsafe.Pointer(&in[0]))
	inL := C.int(len(in))

	ret := C.vpx_decode(inP, inL, outP, outL)
	if ret != 0 {
		return CaptureError(ret)
	}

	return nil
}
