package ui

import (
	"bytes"
	"image/color"
	"io"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/dialup-inc/ascii/term"
)

const (
	chatHeight = 5
)

func NewRenderer() *Renderer {
	return &Renderer{
		requestFrame: make(chan struct{}),
	}
}

type Renderer struct {
	requestFrame chan struct{}

	stateMu sync.Mutex
	state   State

	start time.Time
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

func (r *Renderer) drawVideo(buf *bytes.Buffer, headHeight int) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	vidW, vidH := s.WinSize.Cols, s.WinSize.Rows-chatHeight-headHeight

	a.CursorPosition(1+headHeight, 1)
	a.Background(color.Black)
	a.Bold()

	aspect := getAspect(s.WinSize)
	imgANSI := Image2ANSI(s.Image, vidW, vidH, aspect, false)
	buf.Write(imgANSI)
}

func (r *Renderer) drawHead(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	line1 := " dialup.com/ascii"
	line2 := " (we're hiring!)"

	a.CursorPosition(1, 1)

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})

	a.Foreground(color.RGBA{0x00, 0xFF, 0xFF, 0xFF})
	buf.WriteString(line1)

	a.Foreground(color.RGBA{0x00, 0x44, 0x44, 0xFF})
	buf.WriteString(line2)

	remaining := s.WinSize.Cols - len(line1) - len(line2)
	if remaining > 0 {
		buf.WriteString(strings.Repeat(" ", remaining))
	}
}

func (r *Renderer) drawPrompt(buf *bytes.Buffer, s State) {
	a := term.ANSI{buf}

	prompt := " > "
	input := s.Input

	width := s.WinSize.Cols
	row := s.WinSize.Rows

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	a.Bold()

	// Clear what's there
	a.CursorPosition(row, 1)
	buf.WriteString(strings.Repeat(" ", width))

	// Then write the line
	a.CursorPosition(row, 1)

	var lineLen int

	if !s.ChatActive {
		a.Foreground(color.RGBA{0x33, 0x33, 0x33, 0XFF})
		buf.WriteString(prompt)
		return
	}

	// Add prompt
	a.Foreground(color.White)
	buf.WriteString(prompt)
	lineLen += len(prompt)

	// Add input
	input = truncate(input, width-lineLen, "…")
	buf.WriteString(input)
	lineLen += len(input)

	// Add blinking cursor where you're supposed to type
	cursor := "_"
	if lineLen+len(cursor) < width {
		a.Foreground(color.White)
		a.Blink()
		buf.WriteString(cursor)
		a.BlinkOff()
		lineLen += len(cursor)
	}

	// add label
	label := " Send a message."
	label = truncate(label, width-lineLen, "")
	if input == "" {
		a.Foreground(color.RGBA{0x33, 0x33, 0x33, 0xFF})
		buf.WriteString(label)
		lineLen += len(label)
	}
}

func truncate(s string, n int, ellipsis string) string {
	if len(s) <= n {
		return s
	}

	maxLen := n - len(ellipsis)
	if len(s) > maxLen {
		s = s[:maxLen]
	}

	return s + ellipsis
}

func (r *Renderer) drawChat(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	width := s.WinSize.Cols
	chatTop := s.WinSize.Rows - chatHeight + 1
	logTop := chatTop + 1

	a.CursorPosition(chatTop, 1)
	// Draw background
	a.Normal()

	a.Background(color.RGBA{0x12, 0x12, 0x12, 0xFF})
	label := "ASCII Roulette"
	link := "hit ctrl-t for help"
	buf.WriteString(" ")
	a.Foreground(color.RGBA{0x00, 0xff, 0xff, 0xff})
	buf.WriteString(label)
	textLen := len(label) + len(link) + 2
	if width > textLen {
		buf.WriteString(strings.Repeat(" ", width-textLen))
	}
	a.Foreground(color.RGBA{0x00, 0x99, 0x99, 0xff})
	buf.WriteString(link)
	buf.WriteString(" ")

	a.Background(color.RGBA{0x22, 0x22, 0x22, 0xFF})

	drawChatLine := func(m Message) {
		switch m.Type {
		case MessageTypeIncoming:
			a.Foreground(color.RGBA{0xFF, 0, 0, 0xFF})
			buf.WriteString(" ")
			buf.WriteString(m.User)
			buf.WriteString(": ")
			a.Foreground(color.RGBA{0x99, 0x99, 0x99, 0xFF})
			buf.WriteString(m.Text)

			textLen := utf8.RuneCountInString(m.User) + utf8.RuneCountInString(m.Text) + 3
			if width > textLen {
				buf.WriteString(strings.Repeat(" ", width-textLen))
			}

		case MessageTypeOutgoing:
			a.Foreground(color.RGBA{0xFF, 0xFF, 0, 0xFF})
			buf.WriteString(" ")
			buf.WriteString(m.User)
			buf.WriteString(": ")
			a.Foreground(color.RGBA{0x99, 0x99, 0x99, 0xFF})
			buf.WriteString(m.Text)

			textLen := utf8.RuneCountInString(m.User) + utf8.RuneCountInString(m.Text) + 3
			if width > textLen {
				buf.WriteString(strings.Repeat(" ", width-textLen))
			}

		case MessageTypeInfo:
			a.Foreground(color.RGBA{0x99, 0x99, 0x99, 0xFF})
			buf.WriteString(" ")
			buf.WriteString(m.Text)

			textLen := utf8.RuneCountInString(m.Text) + 1
			if width > textLen {
				buf.WriteString(strings.Repeat(" ", width-textLen))
			}

		case MessageTypeError:
			a.Foreground(color.RGBA{0xAA, 0x00, 0x00, 0xFF})
			buf.WriteString(" ")
			buf.WriteString(m.Text)

			textLen := utf8.RuneCountInString(m.Text) + 1
			if width > textLen {
				buf.WriteString(strings.Repeat(" ", width-textLen))
			}

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

	r.drawPrompt(buf, s)
}

func (r *Renderer) drawTitle(buf *bytes.Buffer, line string) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	// Draw background
	a.Bold()

	a.Background(color.RGBA{0x00, 0x00, 0x00, 0xFF})
	buf.WriteString(strings.Repeat(" ", s.WinSize.Cols*chatHeight))

	a.Foreground(color.RGBA{0x00, 0xff, 0xff, 0xff})
	a.CursorPosition(s.WinSize.Rows-2, (s.WinSize.Cols-len(line))/2+1)
	buf.WriteString(line)
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

func (r *Renderer) drawConfirm(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	// Blank background
	a.Background(color.RGBA{0x00, 0x00, 0x00, 0xFF})
	a.CursorPosition(1, 1)
	buf.WriteString(strings.Repeat(" ", s.WinSize.Cols*s.WinSize.Rows))

	// Draw title
	if s.WinSize.Rows > 6 {
		line := "ASCII Roulette"
		if s.WinSize.Cols > 25 {
			line = "Welcome to " + line
		}

		timeOffset := float64(time.Since(r.start)/time.Millisecond) / 2000.0

		a.Bold()
		a.CursorPosition(2, (s.WinSize.Cols-len(line))/2+1)
		for i, r := range line {
			t := float64(i)/float64(len(line)) + timeOffset
			a.Foreground(rainbow(t))
			buf.WriteRune(r)
		}
	}

	// Draw description
	descWidth := 40
	maxWidth := s.WinSize.Cols - 2
	if maxWidth < descWidth {
		descWidth = maxWidth
	}

	desc := "This program connects you in a video chat with a random person!\n🎥  Your webcam will activate\n🔉  There is no audio\nClicking, you agree to the TOS: dialup.com/terms"
	var descSections [][]string
	for _, line := range strings.Split(desc, "\n") {
		descSections = append(descSections, wordWrap(line, descWidth))
	}

	// Hide parts of the description if they're too long
	var totalLength int
	for i, lines := range descSections {
		if totalLength+len(lines) > s.WinSize.Rows-8 {
			descSections = descSections[:i]
			break
		}

		totalLength += len(lines)
		if i > 0 {
			totalLength++ // for newline
		}
	}

	a.Normal()
	a.Foreground(color.RGBA{0xAA, 0xAA, 0xAA, 0xFF})

	descOffset := 4
	for _, lines := range descSections {
		// Don't display if it'll clip the button

		for i, line := range lines {
			a.CursorPosition((s.WinSize.Rows-totalLength-8)/2+i+descOffset, (s.WinSize.Cols-len(line))/2+1)
			buf.WriteString(line)
		}

		descOffset += len(lines) + 1
	}

	// Draw button
	a.Bold()
	a.Background(color.RGBA{0x11, 0x11, 0x11, 0xFF})
	a.Foreground(color.White)

	line := "  Press Enter to Start Camera  "
	if s.WinSize.Cols <= 25 {
		line = "  Press Enter  "
	}

	a.CursorPosition(s.WinSize.Rows-3, (s.WinSize.Cols-len(line))/2+1)
	if len(line) > 0 {
		buf.WriteString(strings.Repeat(" ", len(line)))
	}

	a.CursorPosition(s.WinSize.Rows-2, (s.WinSize.Cols-len(line))/2+1)
	buf.WriteString(line)

	a.CursorPosition(s.WinSize.Rows-1, (s.WinSize.Cols-len(line))/2+1)
	if len(line) > 0 {
		buf.WriteString(strings.Repeat(" ", len(line)))
	}
}

func (r *Renderer) drawHelp(buf *bytes.Buffer) {
	a := term.ANSI{buf}

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	rows := []string{
		"                 ",
		"  Skip   ctrl-d  ",
		"  Help   ctrl-t  ",
		"  Quit   ctrl-c  ",
		"                 ",
	}

	var boxWidth int
	for _, r := range rows {
		if len(r) > boxWidth {
			boxWidth = len(r)
		}
	}

	boxTop := (s.WinSize.Rows-len(rows)-1)/2 + 1
	boxLeft := (s.WinSize.Cols-boxWidth)/2 + 1

	a.CursorPosition(boxTop, boxLeft)
	a.Bold()
	a.Background(color.Black)
	a.Foreground(color.White)
	buf.WriteString("  Shortcuts      ")

	a.Normal()
	a.Foreground(color.Black)
	a.Background(color.White)
	for i, line := range rows {
		a.CursorPosition(boxTop+i+1, boxLeft)
		buf.WriteString(line)
	}
}

func wordWrap(s string, lineLen int) []string {
	var lines []string

	var line string
	for _, word := range strings.Split(s, " ") {
		if len(line)+len(word)+1 > lineLen {
			lines = append(lines, line)
			line = ""
		}
		line += " " + word
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}

	return lines
}

func rainbow(t float64) *color.RGBA {
	const freq = math.Pi
	r := math.Sin(freq*t)*127 + 128
	g := math.Sin(freq*t+2*math.Pi/3)*127 + 128
	b := math.Sin(freq*t+4*math.Pi/3)*127 + 128

	return &color.RGBA{uint8(r), uint8(g), uint8(b), 0xFF}
}

func (r *Renderer) draw() {
	buf := bytes.NewBuffer(nil)

	r.stateMu.Lock()
	s := r.state
	r.stateMu.Unlock()

	switch s.Page {
	case GlobePage:
		r.drawTitle(buf, "Presented by dialup.com")
		r.drawVideo(buf, 0)

	case PionPage:
		r.drawTitle(buf, "Powered by Pion")
		r.drawVideo(buf, 0)

	case ChatPage:
		r.drawHead(buf)
		r.drawChat(buf)
		r.drawVideo(buf, 1)
		if s.HelpOn {
			r.drawHelp(buf)
		}

	case ConfirmPage:
		r.drawConfirm(buf)

	default:
		r.drawBlank(buf)
	}

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
