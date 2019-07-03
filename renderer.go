package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dialupdotcom/ascii_roulette/term"
	"github.com/nfnt/resize"
)

var chars = []byte(" .,:;i1tfLCG08@")

const chatHeight = 5

func NewRenderer() *Renderer {
	return &Renderer{
		requestFrame: make(chan struct{}),
	}
}

type Renderer struct {
	requestFrame chan struct{}

	stateMu sync.Mutex
	state   State
}

func (r *Renderer) SetState(s State) {
	r.stateMu.Lock()
	r.state = s
	r.stateMu.Unlock()

	r.RequestFrame()
}

func (r *Renderer) RequestFrame() {
	select {
	case r.requestFrame <- struct{}{}:
	default:
	}
}

func (r *Renderer) drawVideo(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	vidW, vidH := s.WindowCols, s.WindowRows-chatHeight

	// pixels are rectangular, not square in the terminal. add a scale factor to account for this
	windowWidth, windowHeight := s.WindowWidth, s.WindowHeight
	windowRows, windowCols := s.WindowRows, s.WindowCols

	winAspect := 2.0
	if windowWidth > 0 && windowHeight > 0 && windowRows > 0 && windowCols > 0 {
		winAspect = float64(windowHeight) * float64(windowCols) / float64(windowRows) / float64(windowWidth)
	}

	winRect := image.Rect(0, 0, vidW, vidH)
	colors := term.ANSIPalette
	canvas := image.NewPaletted(winRect, colors)

	if s.Image != nil {
		imgRect := s.Image.Bounds()
		imgW, imgH := float64(imgRect.Dx())*winAspect, float64(imgRect.Dy())

		fitW, fitH := float64(vidW)/imgW, float64(vidH)/imgH
		var scaleW, scaleH uint
		if fitW < fitH {
			scaleW = uint(imgW * fitW)
			scaleH = uint(imgH * fitW)
		} else {
			scaleW = uint(imgW * fitH)
			scaleH = uint(imgH * fitH)
		}

		scaled := resize.Resize(scaleW, scaleH, s.Image, resize.Bilinear)

		offsetW, offsetH := (vidW-int(scaleW))/2, (vidH-int(scaleH))/2
		rect := image.Rect(
			offsetW,
			offsetH,
			offsetW+int(scaleW),
			offsetH+int(scaleH),
		)
		draw.Draw(canvas, rect, scaled, image.ZP, draw.Over)
	}

	a.CursorPosition(1, 1)
	a.Background(color.Black)
	a.Bold()

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
}

func (r *Renderer) drawChat(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	width := s.WindowCols

	// Draw background
	a.Normal()

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Foreground(color.RGBA{0x00, 0xff, 0xff, 0xff})
	label := "ASCII Roulette"
	link := "dialup.com/ascii"
	buf.WriteString(" ")
	buf.WriteString(label)
	textLen := len(label) + len(link) + 2
	if width > textLen {
		buf.WriteString(strings.Repeat(" ", width-textLen))
	}
	buf.WriteString(link)
	buf.WriteString(" ")

	a.Background(color.RGBA{0x22, 0x22, 0x22, 0xFF})

	drawChatLine := func(m Message) {
		if m.User == "You" {
			a.Foreground(color.RGBA{0xFF, 0xFF, 0, 0xFF})
		} else {
			a.Foreground(color.RGBA{0xFF, 0, 0, 0xFF})
		}

		buf.WriteString(" ")
		buf.WriteString(m.User)
		buf.WriteString(": ")
		a.Foreground(color.RGBA{0x99, 0x99, 0x99, 0xFF})
		buf.WriteString(m.Text)
		textLen := utf8.RuneCountInString(m.User) + utf8.RuneCountInString(m.Text) + 3
		if width > textLen {
			buf.WriteString(strings.Repeat(" ", width-textLen))
		}
	}

	msgs := s.Messages
	if len(s.Messages) > 3 {
		msgs = msgs[len(s.Messages)-3:]
	}
	for _, m := range msgs {
		drawChatLine(m)
	}
	// blank if there arent enough messages
	for i := len(msgs); i < 3; i++ {
		buf.WriteString(strings.Repeat(" ", width))
	}

	input := s.Input

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Foreground(color.White)
	a.Bold()
	buf.WriteString(" > " + input)
	a.Blink()
	buf.WriteString("_")
	textLen = utf8.RuneCountInString(input) + 4
	if width > textLen {
		buf.WriteString(strings.Repeat(" ", width-textLen))
	}
	a.BlinkOff()
}

func (r *Renderer) draw() {
	buf := bytes.NewBuffer(nil)

	r.drawChat(buf)
	r.drawVideo(buf)

	io.Copy(os.Stdout, buf)
}

func (r *Renderer) loop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	for {
		r.draw()

		select {
		case <-r.requestFrame:
		case <-ticker.C:
		}
	}
}

func (r *Renderer) Start() {
	a := term.ANSI{os.Stdout}
	a.HideCursor()

	go r.loop()
}

func (r *Renderer) Stop() {
	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	buf := bytes.NewBuffer(nil)
	a := term.ANSI{buf}

	a.ShowCursor()
	a.Reset()
	a.BackgroundReset()
	a.ForegroundReset()
	a.Normal()
	a.CursorPosition(1, 1)
	buf.WriteString(strings.Repeat(" ", s.WindowCols*s.WindowRows))
	a.CursorPosition(1, 1)

	io.Copy(os.Stdout, buf)
}
