package camera

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"os"
	"strings"

	"github.com/blackjack/webcam"
)

const webcamReadTimeout = 5

type Camera struct {
	callback FrameCallback
}

func (c *Camera) Start(camID, width, height int) error {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		return err
	}

	var selectedFormat webcam.PixelFormat
	for v, k := range cam.GetSupportedFormats() {
		if strings.HasPrefix(k, "Motion-JPEG") {
			selectedFormat = v
			break
		}
	}

	if selectedFormat == 0 {
		return fmt.Errorf("Only Motion-JPEG supported")
	}

	if _, _, _, err = cam.SetImageFormat(selectedFormat, uint32(width), uint32(height)); err != nil {
		return err
	}

	if err = cam.StartStreaming(); err != nil {
		return err
	}

	go func() {
		for {
			err = cam.WaitForFrame(webcamReadTimeout)
			switch err.(type) {
			case nil:
			case *webcam.Timeout:
				fmt.Fprint(os.Stderr, err.Error())
				continue
			default:
				c.callback(nil, err)
				return
			}

			frame, err := cam.ReadFrame()
			if len(frame) != 0 {
				img, err := jpeg.Decode(bytes.NewReader(frame))
				c.callback(img, err)
			} else if err != nil {
				c.callback(nil, err)
			}
		}
	}()
	return nil
}

func (c *Camera) Close() error {
	// TODO
	return nil
}

func New(cb FrameCallback) (*Camera, error) {
	return &Camera{callback: cb}, nil
}
