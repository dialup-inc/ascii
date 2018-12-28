package main

import (
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
)

func main() {
	width, height := 640, 480

	data, err := ioutil.ReadFile("frame-1.pkt")
	if err != nil {
		log.Fatal(err)
	}

	if err := StartDecode(); err != nil {
		log.Fatal(err)
	}

	frame := make([]byte, 1<<22)
	if err := DecodeFrame(frame, data); err != nil {
		log.Fatal(err)
	}

	yi := width * height
	cbi := yi + width*height/4
	cri := cbi + width*height/4

	img := &image.YCbCr{
		Y:              frame[:yi],
		YStride:        width,
		Cb:             frame[yi:cbi],
		Cr:             frame[cbi:cri],
		CStride:        width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}

	f, err := os.Create("frame.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}

	if err := StopDecode(); err != nil {
		log.Fatal(err)
	}
}
