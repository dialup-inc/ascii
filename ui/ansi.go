package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"

	"github.com/dialup-inc/ascii/term"
	"github.com/nfnt/resize"
)

var chars = []byte(" .,:;i1tfLCG08@")

func Image2ANSI(img image.Image, cols, rows int, aspect float64, lightBackground bool) []byte {
	buf := bytes.NewBuffer(nil)
	a := term.ANSI{buf}

	colors := term.ANSIPalette
	canvasRect := image.Rect(0, 0, cols, rows)
	canvas := image.NewPaletted(canvasRect, colors)

	// If there's an image, resize to fit inside canvas dimensions...
	if img != nil {
		imgRect := img.Bounds()
		imgW, imgH := float64(imgRect.Dx())*aspect, float64(imgRect.Dy())
		fitW, fitH := float64(cols)/imgW, float64(rows)/imgH

		var scaleW, scaleH uint
		if fitW < fitH {
			scaleW = uint(imgW * fitW)
			scaleH = uint(imgH * fitW)
		} else {
			scaleW = uint(imgW * fitH)
			scaleH = uint(imgH * fitH)
		}

		scaled := resize.Resize(scaleW, scaleH, img, resize.Bilinear)

		offsetW, offsetH := (cols-int(scaleW))/2, (rows-int(scaleH))/2
		fitRect := image.Rect(
			offsetW,
			offsetH,
			offsetW+int(scaleW),
			offsetH+int(scaleH),
		)
		draw.Draw(canvas, fitRect, scaled, image.ZP, draw.Over)
	}

	// Draw a character and colored ANSI escape sequence for each pixel...
	currentColor := -1
	for _, p := range canvas.Pix {
		pxColor := colors[p]

		if int(p) != currentColor {
			a.Foreground(pxColor)

			currentColor = int(p)
		}

		k, _, _, _ := color.GrayModel.Convert(pxColor).RGBA()
		chr := int(k) * (len(chars) - 1) / 0xffff

		if lightBackground {
			chr = len(chars) - chr - 1
		}

		buf.WriteByte(chars[chr])
	}

	return buf.Bytes()
}
