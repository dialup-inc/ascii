package main

import (
	"image"

	"github.com/dialupdotcom/ascii_roulette/vpx"
)

type Decoder struct {
	buf []byte

	width  int
	height int

	decoder *vpx.Decoder
}

func NewDecoder(width, height int) (*Decoder, error) {
	dec, err := vpx.NewDecoder()
	if err != nil {
		return nil, err
	}

	return &Decoder{
		buf:    make([]byte, width*height*4),
		width:  width,
		height: height,

		decoder: dec,
	}, nil
}

// can't use concurrently
func (d *Decoder) Decode(data []byte) (image.Image, error) {
	var err error
	var n int
	if len(data) > 0 {
		n, err = d.decoder.Decode(d.buf, data)
		if err != nil {
			return nil, err
		}
	}

	frame := d.buf[:n]

	yi := d.width * d.height
	cbi := yi + d.width*d.height/4
	cri := cbi + d.width*d.height/4

	return &image.YCbCr{
		Y:              frame[:yi],
		YStride:        d.width,
		Cb:             frame[yi:cbi],
		Cr:             frame[cbi:cri],
		CStride:        d.width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, d.width, d.height),
	}, nil
}
