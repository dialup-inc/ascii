package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dialupdotcom/ascii_roulette/camera"
	"github.com/dialupdotcom/ascii_roulette/vpx"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media"
)

func NewCapture(width, height int) (*Capture, error) {
	cap := &Capture{
		vpxBuf: make([]byte, 5*1024*1024),
		width:  width,
		height: height,
	}

	enc, err := vpx.NewEncoder(width, height)
	if err != nil {
		return nil, err
	}
	cap.enc = enc

	cam, err := camera.New(cap.onFrame)
	if err != nil {
		return nil, err
	}
	cap.cam = cam

	return cap, nil
}

type Capture struct {
	enc *vpx.Encoder
	cam *camera.Camera

	width  int
	height int

	ptsMu sync.Mutex
	pts   int

	vpxBuf []byte

	forceKeyframe uint32
	encodeLock    uint32

	track *webrtc.Track
}

func (c *Capture) Start(camID int, frameRate float32) error {
	return c.cam.Start(camID, c.width, c.height, frameRate)
}

func (c *Capture) Stop() error {
	// TODO
	return nil
}

func (c *Capture) RequestKeyframe() {
	atomic.StoreUint32(&c.forceKeyframe, 1)
}

func (c *Capture) SetTrack(track *webrtc.Track) {
	c.track = track
}

func (c *Capture) onFrame(frame []byte) {
	if !atomic.CompareAndSwapUint32(&c.encodeLock, 0, 1) {
		return
	}
	defer atomic.StoreUint32(&c.encodeLock, 0)

	forceKeyframe := atomic.CompareAndSwapUint32(&c.forceKeyframe, 1, 0)

	n, err := c.enc.Encode(c.vpxBuf, frame, c.pts, forceKeyframe)
	if err != nil {
		fmt.Println("encode: ", err)
		return
	}
	c.pts++

	data := c.vpxBuf[:n]
	samp := media.Sample{Data: data, Samples: 1}

	if c.track == nil {
		return
	}

	if err := c.track.WriteSample(samp); err != nil {
		fmt.Println("write sample: ", err)
		return
	}
}
