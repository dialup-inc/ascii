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
	"time"
	"unicode/utf8"

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

	img   image.Image
	imgMu sync.Mutex

	input    string
	Messages []Message

	start time.Time
}

type Message struct {
	User string
	Text string
}

func (r *Renderer) SetImage(i image.Image) {
	r.imgMu.Lock()
	r.img = i
	r.imgMu.Unlock()

	r.RequestFrame()
}

func (r *Renderer) SetInput(text string) {
	r.input = text
	r.RequestFrame()
}

func (r *Renderer) RequestFrame() {
	select {
	case r.requestFrame <- struct{}{}:
	default:
	}
}

func (r *Renderer) drawVideo(buf *bytes.Buffer, winSize WinSize, t time.Duration) {
	a := ANSI{buf}

	r.imgMu.Lock()
	img := r.img
	r.imgMu.Unlock()

	width, height := winSize.Cols, winSize.Rows-chatHeight

	winRect := image.Rect(0, 0, width, height)
	colors := ANSIPalette
	canvas := image.NewPaletted(winRect, colors)

	if img != nil {
		scaled := resize.Resize(uint(width), uint(height), img, resize.Bilinear)
		draw.Draw(canvas, scaled.Bounds(), scaled, image.ZP, draw.Over)
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

func (r *Renderer) drawChat(buf *bytes.Buffer, winSize WinSize, t time.Duration) {
	a := ANSI{buf}

	width := winSize.Cols

	// Draw background
	a.Normal()

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Foreground(color.RGBA{0x00, 0xff, 0xff, 0xff})
	label := "ASCII Roulette"
	link := "dialup.com/ascii"
	buf.WriteString(" ")
	buf.WriteString(label)
	buf.WriteString(strings.Repeat(" ", width-len(label)-len(link)-2))
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
		buf.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(m.User)-utf8.RuneCountInString(m.Text)-3))
	}

	msgs := r.Messages
	if len(r.Messages) > 3 {
		msgs = msgs[len(r.Messages)-3:]
	}
	for _, m := range msgs {
		drawChatLine(m)
	}
	// blank if there arent enough messages
	for i := len(msgs); i < 3; i++ {
		buf.WriteString(strings.Repeat(" ", width))
	}

	input := r.input // fmt.Sprintf("%q", r.input)

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Foreground(color.White)
	a.Bold()
	buf.WriteString(" > " + input)
	a.Blink()
	buf.WriteString("_")
	buf.WriteString(strings.Repeat(" ", width-utf8.RuneCountInString(input)-4))
	a.BlinkOff()
}

func (r *Renderer) draw() {
	winSize, err := GetWinSize()
	if err != nil {
		log.Fatal(err)
	}

	buf := bytes.NewBuffer(nil)

	t := time.Since(r.start)

	r.drawChat(buf, winSize, t)
	r.drawVideo(buf, winSize, t)

	io.Copy(os.Stdout, buf)
}

func (r *Renderer) loop() {
	r.start = time.Now()
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
	a := ANSI{os.Stdout}
	a.HideCursor()

	go r.loop()
}

func (r *Renderer) Stop() {
	a := ANSI{os.Stdout}
	a.ShowCursor()
	a.Reset()
}
