package vpx

/*
#cgo pkg-config: --static vpx

#include "vpx/vpx_encoder.h"

int vpx_init_enc(vpx_codec_ctx_t *ctx, vpx_image_t **raw, int width, int height);
int vpx_encode(vpx_codec_ctx_t *ctx, vpx_image_t *raw, const char* yv12_frame, int yv12_len, char* encoded, int encoded_cap, int* size, int pts, int force_key_frame);
int vpx_cleanup_enc(vpx_codec_ctx_t *ctx, vpx_image_t *raw);
*/
import "C"

import (
	"unsafe"
)

type Encoder struct {
	ctx C.vpx_codec_ctx_t
	img *C.vpx_image_t
}

func NewEncoder(width, height int) (*Encoder, error) {
	e := &Encoder{}
	ret := C.vpx_init_enc(&e.ctx, &e.img, C.int(width), C.int(height))
	if ret != 0 {
		return nil, VPXCodecErr(ret)
	}
	return e, nil
}

func (e *Encoder) Close() error {
	ret := C.vpx_cleanup_enc(&e.ctx, e.img)
	if ret != 0 {
		return VPXCodecErr(ret)
	}
	return nil
}

func (e *Encoder) Encode(vpxFrame, yuvFrame []byte, pts int, forceKeyframe bool) (n int, err error) {
	inP := (*C.char)(unsafe.Pointer(&yuvFrame[0]))
	inL := C.int(len(yuvFrame))

	outP := (*C.char)(unsafe.Pointer(&vpxFrame[0]))
	outCap := C.int(cap(vpxFrame))
	outL := (*C.int)(unsafe.Pointer(&n))

	forceKeyframeB := C.int(0)
	if forceKeyframe {
		forceKeyframeB = C.int(1)
	}

	ret := C.vpx_encode(&e.ctx, e.img, inP, inL, outP, outCap, outL, C.int(pts), forceKeyframeB)
	if ret != 0 {
		return n, VPXCodecErr(ret)
	}

	return n, nil
}
