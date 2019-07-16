package camera

/*
#cgo LDFLAGS: -L. -lcam -lc++ -framework Foundation -framework AVFoundation -framework CoreVideo -framework CoreMedia

#include "cam_avfoundation.h"

extern void onFrame(void *userdata, void *buf, int len, int width, int height);
void onFrame_cgo(void *userdata, void *buf, int len, int width, int height) {
	onFrame(userdata, buf, len, width, height);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/dialup-inc/ascii/yuv"
)
type Camera struct {
	c C.Camera

	handleID handleID
	callback FrameCallback
}

type CamError int

const (
	ErrCamOK CamError = 0
	ErrCamInitFailed CamError = -1
	ErrCamOpenFailed = -2
	ErrCamNotFound = -3
)

func (c CamError) Error() string {
	switch c {
	case ErrCamInitFailed:
		return "init failed"
	case ErrCamOpenFailed:
		return "open failed"
	case ErrCamNotFound:
		return "not found"
	default:
		return fmt.Sprintf("error %d", c)
	}
}

func (c *Camera) Start(camID, width, height int) error {
	if ret := C.cam_start(c.c, C.int(camID), C.int(width), C.int(height)); ret != 0 {
		return CamError(ret)
	}
	return nil
}

func (c *Camera) Close() error {
	// TODO
	return nil
}

func (c *Camera) onFrame(data []byte, width, height int) {
	c.callback(yuv.FromI420(data, width, height))
}

func New(cb FrameCallback) (*Camera, error) {
	cam := &Camera{}

	cam.callback = cb
	cam.handleID = register(func(data []byte, width, height int) {
		cam.onFrame(data, width, height)
	})

	if ret := C.cam_init(&cam.c, (C.FrameCallback)(unsafe.Pointer(C.onFrame_cgo)), unsafe.Pointer(&cam.handleID)); ret != 0 {
		return nil, fmt.Errorf("error %d", CamError(ret))
	}
	return cam, nil
}
