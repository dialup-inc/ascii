package vpx

/*
#cgo pkg-config: --static vpx

#include "vpx/vpx_decoder.h"

int vpx_init_dec(vpx_codec_ctx_t *ctx);
int vpx_decode(vpx_codec_ctx_t *ctx, const char* frame, int frame_len, char* yv12_frame, int yv12_len, int *decoded_len);
int vpx_cleanup_dec(vpx_codec_ctx_t *ctx);
*/
import "C"
import (
	"unsafe"
)

type Decoder struct {
	ctx C.vpx_codec_ctx_t
}

func NewDecoder() (*Decoder, error) {
	d := &Decoder{}
	ret := C.vpx_init_dec(&d.ctx)
	if ret != 0 {
		return nil, VPXCodecErr(ret)
	}
	return d, nil
}

func (d *Decoder) Close() error {
	ret := C.vpx_cleanup_dec(&d.ctx)
	if ret != 0 {
		return VPXCodecErr(ret)
	}
	return nil
}

func (d *Decoder) Decode(out, in []byte) (n int, err error) {
	inP := (*C.char)(unsafe.Pointer(&in[0]))
	inL := C.int(len(in))

	outP := (*C.char)(unsafe.Pointer(&out[0]))
	outCap := C.int(cap(out))
	outL := (*C.int)(unsafe.Pointer(&n))

	ret := C.vpx_decode(&d.ctx, inP, inL, outP, outCap, outL)
	if ret != 0 {
		return n, VPXCodecErr(ret)
	}

	return n, nil
}
