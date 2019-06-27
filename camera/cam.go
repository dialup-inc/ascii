package camera

/*
#cgo LDFLAGS: -L. -lcam -lc++ -framework Foundation -framework AVFoundation -framework CoreVideo -framework CoreMedia

#include "cam.h"

extern void onFrame(void *userdata, void *buf, int len);
void onFrame_cgo(void *userdata, void *buf, int len) {
	onFrame(userdata, buf, len);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Camera struct {
	c C.Camera

	callback int
}

func (c *Camera) Start(camID, width, height int) error {
	if ret := C.cam_start(c.c, C.int(camID), C.int(width), C.int(height)); ret != 0 {
		return fmt.Errorf("error %d", ret)
	}
	return nil
}

func (c *Camera) Close() error {
	// TODO
	return nil
}

func New(cb func([]byte)) (*Camera, error) {
	cam := &Camera{}

	cam.callback = register(cb)

	if ret := C.cam_init(&cam.c, (C.FrameCallback)(unsafe.Pointer(C.onFrame_cgo)), unsafe.Pointer(&cam.callback)); ret != 0 {
		return nil, fmt.Errorf("error %d", ret)
	}
	return cam, nil
}
