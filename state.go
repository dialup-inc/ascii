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

	case TypeSentMessage:
		msg := e.Payload.(string)
		s.Messages = append(s.Messages, Message{
			User: "You",
			Text: msg,
		})
		s.Input = ""

	case TypeError:
		msg := e.Payload.(string)
		s.Messages = append(s.Messages, Message{
			User: "Error",
			Text: msg,
		})

	case TypeInfo:
		msg := e.Payload.(string)
		s.Messages = append(s.Messages, Message{
			User: "Info",
			Text: msg,
		})

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

	case TypeSetTitle:
		str := e.Payload.(string)
		s.Title = str
	}
	return s
}

type EventType int

const (
	TypeKeypress EventType = iota
	TypeSentMessage
	TypeFrame
	TypeReceivedChat
	TypeBackspace
	TypeResize
	TypeError
	TypeInfo
	TypeSetTitle
)

type Event struct {
	Type    EventType
	Payload interface{}
}

type State struct {
	Input    string
	Messages []Message
	Image    image.Image

	Title string

	WindowWidth  int
	WindowHeight int
	WindowCols   int
	WindowRows   int
}

type Message struct {
	User string
	Text string
}
