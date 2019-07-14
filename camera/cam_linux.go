package camera

import (
	"fmt"
	"os"
	"strings"

	"github.com/blackjack/webcam"
)

const webcamReadTimeout = 5

type Camera struct {
	callback func([]byte)
}

func (c *Camera) Start(camID, width, height int) error {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		return err
	}

	var selectedFormat webcam.PixelFormat
	for v, k := range cam.GetSupportedFormats() {
		if strings.HasPrefix(k, "YUYV") {
			selectedFormat = v
			break
		}
	}

	if selectedFormat == 0 {
		return fmt.Errorf("Only YUYV supported")
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
				panic(err.Error())
			}

			frame, err := cam.ReadFrame()
			if len(frame) != 0 {
				c.callback(frame)
			} else if err != nil {
				panic(err.Error())
			}
		}
	}()
	return nil
}

func (c *Camera) Close() error {
	// TODO
	return nil
}

func New(cb func([]byte)) (*Camera, error) {
	return &Camera{callback: cb}, nil
}
