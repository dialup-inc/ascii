package main

import (
	"image"
	"regexp"

	"github.com/dialupdotcom/ascii_roulette/term"
)

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func StateReducer(s State, e Event) State {
	switch e.Type {
	case TypeResize:
		ws := e.Payload.(term.WinSize)
		s.WindowWidth = ws.Width
		s.WindowHeight = ws.Height
		s.WindowCols = ws.Cols
		s.WindowRows = ws.Rows

	case TypeBackspace:
		if len(s.Input) == 0 {
			return s
		}
		s.Input = s.Input[:len(s.Input)-1]

	case TypeSendMessage:
		s.Messages = append(s.Messages, Message{
			User: "You",
			Text: s.Input,
		})
		s.Input = ""

	case TypeKeypress:
		chr := e.Payload.(rune)
		s.Input += string(chr)

		// Strip ansi codes
		s.Input = ansiRegex.ReplaceAllString(s.Input, "")

	case TypeFrame:
		img := e.Payload.(image.Image)
		s.Image = img

	case TypeReceivedChat:
		str := e.Payload.(string)
		s.Messages = append(s.Messages, Message{User: "Them", Text: str})
	}
	return s
}

type EventType int

const (
	TypeKeypress EventType = iota
	TypeSendMessage
	TypeFrame
	TypeReceivedChat
	TypeBackspace
	TypeResize
)

type Event struct {
	Type    EventType
	Payload interface{}
}

type State struct {
	Input    string
	Messages []Message
	Image    image.Image

	WindowWidth  int
	WindowHeight int
	WindowCols   int
	WindowRows   int
}

type Message struct {
	User string
	Text string
}
