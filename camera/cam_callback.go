package camera

import (
	"sync"
	"unsafe"
)

/*
extern void onFrame(void *userdata, void *buf, int len);
*/
import "C"

var mu sync.Mutex
var index int
var fns = make(map[int]func([]byte))

//export onFrame
func onFrame(userdata unsafe.Pointer, buf unsafe.Pointer, size C.int) {
	frame := C.GoBytes(buf, size)

	cbNum := (*C.int)(userdata)

	cb := lookup(int(*cbNum))

	if cb == nil {
		return
	}
	cb(frame)
}

func register(fn func([]byte)) int {
	mu.Lock()
	defer mu.Unlock()
	index++
	for fns[index] != nil {
		index++
	}
	fns[index] = fn
	return index
}

func lookup(i int) func([]byte) {
	mu.Lock()
	defer mu.Unlock()
	return fns[i]
}

func unregister(i int) {
	mu.Lock()
	defer mu.Unlock()
	delete(fns, i)
}
