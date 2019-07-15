package ui

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dialupdotcom/ascii_roulette/term"
)

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

func (r *Renderer) GetState() State {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()

	return r.state
}

func (r *Renderer) Dispatch(e Event) {
	r.stateMu.Lock()
	newState := StateReducer(r.state, e)
	var changed bool
	if !reflect.DeepEqual(r.state, newState) {
		changed = true
	}
	r.state = newState
	r.stateMu.Unlock()

	if changed {
		r.RequestFrame()
	}
}

func (r *Renderer) RequestFrame() {
	select {
	case r.requestFrame <- struct{}{}:
	default:
	}
}

// pixels are rectangular, not square in the terminal. add a scale factor to account for this
func getAspect(w term.WinSize) float64 {
	if w.Width == 0 || w.Height == 0 || w.Rows == 0 || w.Cols == 0 {
		return 2.0
	}
	return float64(w.Height) * float64(w.Cols) / float64(w.Rows) / float64(w.Width)
}

func (r *Renderer) drawVideo(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	vidW, vidH := s.WinSize.Cols, s.WinSize.Rows-chatHeight

	a.CursorPosition(1, 1)
	a.Background(color.Black)
	a.Bold()

	aspect := getAspect(s.WinSize)
	imgANSI := Image2ANSI(s.Image, vidW, vidH, aspect, false)
	buf.Write(imgANSI)
}

func (r *Renderer) drawChat(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	width := s.WinSize.Cols
	chatTop := s.WinSize.Rows - chatHeight + 1
	logTop := chatTop + 1
	chatBottom := logTop + 3

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
		} else if m.User == "Them" {
			a.Foreground(color.RGBA{0xFF, 0, 0, 0xFF})
		} else {
			a.Foreground(color.RGBA{0x99, 0x99, 0x99, 0xFF})
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
	for i, m := range msgs {
		a.CursorPosition(logTop+i, 0)
		drawChatLine(m)
	}
	// blank if there arent enough messages
	for i := len(msgs); i < 3; i++ {
		a.CursorPosition(logTop+i, 0)
		buf.WriteString(strings.Repeat(" ", width))
	}

	input := s.Input

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Foreground(color.White)
	a.Bold()

	// First clear
	a.CursorPosition(chatBottom, 0)
	textLen = utf8.RuneCountInString(input) + 4
	if width > textLen {
		buf.WriteString(strings.Repeat(" ", width))
	}

	// Then write line
	a.CursorPosition(chatBottom, 0)
	inputLine := " > " + input
	if len(inputLine) >= width {
		inputLine = inputLine[:width-1] + "…"
	}
	buf.WriteString(inputLine)

	// Add blinking cursor where you're supposed to type
	if len(inputLine) < width {
		a.Blink()
		buf.WriteString("_")
	}

	a.BlinkOff()
}

func (r *Renderer) drawTitle(buf *bytes.Buffer, line1, line2 string) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	// Draw background
	a.Bold()

	a.Background(color.RGBA{0x00, 0x00, 0x00, 0xFF})
	buf.WriteString(strings.Repeat(" ", s.WinSize.Cols*chatHeight))

	a.Foreground(color.RGBA{0x00, 0xff, 0xff, 0xff})
	a.CursorPosition(s.WinSize.Rows-2, (s.WinSize.Cols-len(line1))/2+1)
	buf.WriteString(line1)

	a.Foreground(color.RGBA{0x00, 0x55, 0x55, 0xff})
	a.CursorPosition(s.WinSize.Rows-1, (s.WinSize.Cols-len(line2))/2+1)
	buf.WriteString(line2)
}

func (r *Renderer) drawBlank(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	a.Background(color.RGBA{0x00, 0x00, 0x00, 0xFF})

	a.CursorPosition(1, 1)
	buf.WriteString(strings.Repeat(" ", s.WinSize.Cols*s.WinSize.Rows))
}

func (r *Renderer) draw() {
	buf := bytes.NewBuffer(nil)

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	switch s.Page {
	case GlobePage:
		r.drawTitle(buf, "Presented by dialup.com", "(we're hiring!)")
		r.drawVideo(buf)

	case PionPage:
		r.drawTitle(buf, "Powered by Pion", "")
		r.drawVideo(buf)

	case ChatPage:
		r.drawChat(buf)
		r.drawVideo(buf)

	default:
		r.drawBlank(buf)
	}

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
	buf.WriteString(strings.Repeat(" ", s.WinSize.Cols*s.WinSize.Rows))
	a.CursorPosition(1, 1)

	io.Copy(os.Stdout, buf)
}
