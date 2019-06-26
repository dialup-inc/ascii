package term

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/nfnt/resize"
)

var chars = []byte(" .,:;i1tfLCG08@")

func NewRenderer() *Renderer {
	return &Renderer{
		requestFrame: make(chan struct{}),
	}
}

type Renderer struct {
	requestFrame chan struct{}

	img   image.Image
	imgMu sync.Mutex
}

func (r *Renderer) SetImage(i image.Image) {
	r.imgMu.Lock()
	r.img = i
	r.imgMu.Unlock()

	select {
	case r.requestFrame <- struct{}{}:
	default:
	}
}

func (r *Renderer) draw() {
	var buf bytes.Buffer
	a := ANSI{&buf}

	winsize, err := GetWinSize()
	if err != nil {
		log.Fatal(err)
	}

	r.imgMu.Lock()
	img := r.img
	r.imgMu.Unlock()

	if r.img == nil {
		return
	}

	scaled := resize.Resize(uint(winsize.Cols), uint(winsize.Rows), img, resize.Bilinear)

	winRect := image.Rect(0, 0, winsize.Cols, winsize.Rows)
	colors := ANSIPalette
	canvas := image.NewPaletted(winRect, colors)

	draw.Draw(canvas, scaled.Bounds(), scaled, image.ZP, draw.Over)

	a.CursorPosition(1, 1)

	currentColor := -1
	for _, p := range canvas.Pix {
		pxColor := colors[p]

		if int(p) != currentColor {
			a.Foreground(pxColor)

			currentColor = int(p)
		}

		k, _, _, _ := color.GrayModel.Convert(pxColor).RGBA()
		chr := int(k) * (len(chars) - 1) / 0xffff

		buf.WriteByte(chars[chr])
	}

	a.CursorPosition(1, 1)

	io.Copy(os.Stdout, &buf)
}

func (r *Renderer) loop() {
	for range r.requestFrame {
		r.draw()
	}
}

func (r *Renderer) Start() {
	var buf bytes.Buffer

	a := ANSI{&buf}
	a.Clear()
	a.HideCursor()
	a.Bold()
	a.Background(color.Black)

	io.Copy(os.Stdout, &buf)

	go r.loop()
}

func (r *Renderer) Stop() {
	a := ANSI{os.Stdout}
	a.ShowCursor()
	a.Reset()
}
