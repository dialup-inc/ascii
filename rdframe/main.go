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
	frame, err := ioutil.ReadFile("frame.yuv")
	if err != nil {
		log.Fatal(err)
	}

	// frameSz := width * height * 3 / 2
	// y := frame[:frameSz]
	// cb := frame[:frameSz]
	// cr := frame[:frameSz]

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

	// stride := height

	// img := image.NewGray(image.Rect(0, 0, width, height))
	// i := 0
	// for y := 0; y < height; y++ {
	// 	for x := 0; x < width; x++ {
	// 		c := color.Gray{frame[i]}
	// 		img.SetGray(x, y, c)
	// 		i++
	// 	}
	// }

	// for y := 0; y < height/2; y++ {
	// 	for x := 0; x < width/2; x++ {
	// 		c := color.Gray{frame[i]}
	// 		img.SetGray(x, y, c)
	// 		i++
	// 	}
	// }

	// for y := 0; y < height/2; y++ {
	// 	for x := 0; x < width/2; x++ {
	// 		c := color.Gray{frame[i]}
	// 		img.SetGray(x+width/2, y, c)
	// 		i++
	// 	}
	// }

	f, err := os.Create("frame.png")
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}
