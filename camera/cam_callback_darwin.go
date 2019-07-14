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
var nextID handleID
var handles = make(map[handleID]func([]byte))

type handleID int

//export onFrame
func onFrame(userdata unsafe.Pointer, buf unsafe.Pointer, size C.int) {
	data := C.GoBytes(buf, size)

	handleNum := (*C.int)(userdata)

	cb := lookup(handleID(*handleNum))
	cb(data)
}

func register(fn func([]byte)) handleID {
	mu.Lock()
	defer mu.Unlock()

	nextID++
	for handles[nextID] != nil {
		nextID++
	}
	handles[nextID] = fn

	return nextID
}

func lookup(i handleID) func([]byte) {
	mu.Lock()
	defer mu.Unlock()

	return handles[i]
}

func unregister(i handleID) {
	mu.Lock()
	defer mu.Unlock()

	delete(handles, i)
}
