package camera

import (
	"sync"
	"unsafe"
)

/*
extern void onFrame(void *userdata, void *buf, int len, int width, int height);
*/
import "C"

type frameCb func(frame []byte, width, height int)

var mu sync.Mutex
var nextID handleID
var handles = make(map[handleID]frameCb)

type handleID int

//export onFrame
func onFrame(userdata unsafe.Pointer, buf unsafe.Pointer, size, cWidth, cHeight C.int) {
	data := C.GoBytes(buf, size)
	width, height := int(cWidth), int(cHeight)


	handleNum := (*C.int)(userdata)

	cb := lookup(handleID(*handleNum))
	cb(data, width, height)
}

func register(fn frameCb) handleID {
	mu.Lock()
	defer mu.Unlock()

	nextID++
	for handles[nextID] != nil {
		nextID++
	}
	handles[nextID] = fn

	return nextID
}

func lookup(i handleID) frameCb {
	mu.Lock()
	defer mu.Unlock()

	return handles[i]
}

func unregister(i handleID) {
	mu.Lock()
	defer mu.Unlock()

	delete(handles, i)
}
