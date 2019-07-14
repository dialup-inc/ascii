package yuv

import (
	"fmt"
	"image"
	"image/color"
)

// FromI420 decodes an i420-encoded YUV image into a Go Image.
//
// See https://www.fourcc.org/pixel-format/yuv-i420/
func FromI420(frame []byte, width, height int) (*image.YCbCr, error) {
	yi := width * height
	cbi := yi + width*height/4
	cri := cbi + width*height/4

	if cri > len(frame) {
		return nil, fmt.Errorf("frame length (%d) less than expected (%d)", len(frame), cri)
	}

	return &image.YCbCr{
		Y:              frame[:yi],
		YStride:        width,
		Cb:             frame[yi:cbi],
		Cr:             frame[cbi:cri],
		CStride:        width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}, nil
}

// FromNV21 decodes an NV21-encoded, big-endian YUV image into a Go Image.
//
// See https://www.fourcc.org/pixel-format/yuv-nv21/
func FromNV21(frame []byte, width, height int) (*image.YCbCr, error) {
	yi := width * height
	ci := yi + width*height/2

	if ci > len(frame) {
		return nil, fmt.Errorf("frame length (%d) less than expected (%d)", len(frame), ci)
	}

	var cb, cr []byte
	for i := yi; i < ci; i += 2 {
		cb = append(cb, frame[i])
		cr = append(cr, frame[i+1])
	}

	return &image.YCbCr{
		Y:              frame[:yi],
		YStride:        width,
		Cb:             cb,
		Cr:             cr,
		CStride:        width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, width, height),
	}, nil
}

func convertTo420(img image.Image) *image.YCbCr {
	bounds := img.Bounds()
	img420 := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio420)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			yy, cb, cr := color.RGBToYCbCr(uint8(r>>8), uint8(g>>8), uint8(b>>8))

			cy := img420.YOffset(x, y)
			ci := img420.COffset(x, y)
			img420.Y[cy] = yy
			img420.Cb[ci] = cb
			img420.Cr[ci] = cr
		}
	}

	return img420
}

// ToI420 converts a Go image into an I420-encoded YUV raw image slice
//
// See https://www.fourcc.org/pixel-format/yuv-i420/
func ToI420(img image.Image) (frame []byte, width, height int) {
	bounds := img.Bounds()

	var img420 *image.YCbCr
	if y, ok := img.(*image.YCbCr); ok && y.SubsampleRatio == image.YCbCrSubsampleRatio420 {
		// If the image is already I420, just use it
		img420 = y
	} else {
		// Otherwise convert it to I420
		img420 = convertTo420(img)
	}

	frame = append(frame, img420.Y...)
	frame = append(frame, img420.Cb...)
	frame = append(frame, img420.Cr...)

	return frame, bounds.Dx(), bounds.Dy()
}
