package camera

/*
#cgo LDFLAGS: -L. -lcam -lc++ -framework Foundation -framework AVFoundation -framework CoreVideo -framework CoreMedia

#include "cam_avfoundation.h"

extern void onFrame(void *userdata, void *buf, int len);
void onFrame_cgo(void *userdata, void *buf, int len) {
	onFrame(userdata, buf, len);
}
*/
import "C"
import (
	"fmt"
	"image"
	"unsafe"

	"github.com/dialup-inc/ascii/yuv"
)

type Camera struct {
	c C.Camera

	width int
	height int

	handleID handleID
	callback func(image.Image)
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
	c.width = width
	c.height = height

	if ret := C.cam_start(c.c, C.int(camID), C.int(width), C.int(height)); ret != 0 {
		return CamError(ret)
	}
	return nil
}

func (c *Camera) Close() error {
	// TODO
	return nil
}

func (c *Camera) onFrame(data []byte) {
	img, err := yuv.FromI420(data, c.width, c.height)
	if err != nil {
		panic(err)
	}
	c.callback(img)
}

func New(cb func(image.Image)) (*Camera, error) {
	cam := &Camera{}

	cam.callback = cb
	cam.handleID = register(func(data []byte) {
		cam.onFrame(data)
	})

	if ret := C.cam_init(&cam.c, (C.FrameCallback)(unsafe.Pointer(C.onFrame_cgo)), unsafe.Pointer(&cam.handleID)); ret != 0 {
		return nil, fmt.Errorf("error %d", CamError(ret))
	}
	return cam, nil
}
