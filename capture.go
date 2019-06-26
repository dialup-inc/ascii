package main

import (
	"fmt"
	"sync"

	"github.com/pions/asciirtc/camera"
	"github.com/pions/asciirtc/vpx"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/media"
)

func NewCapture(width, height int) (*Capture, error) {
	cap := &Capture{
		vpxBuf: make([]byte, 5*1024*1024),
		width: width,
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

	width int
	height int

	ptsMu sync.Mutex
	pts   int

	vpxBuf []byte

	track *webrtc.Track
}

func (c *Capture) Start(camID int, frameRate float32) error {
	return c.cam.Start(camID, c.width, c.height, frameRate)
}

func (c *Capture) Stop() error {
	// TODO
	return nil
}

func (c *Capture) SetTrack(track *webrtc.Track) {
	c.track = track
}

func (c *Capture) onFrame(frame []byte) {
	c.ptsMu.Lock()
	defer c.ptsMu.Unlock()

	n, err := c.enc.Encode(c.vpxBuf, frame, c.pts, c.pts%10 == 0)
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
