package ui

import (
	"regexp"
	"strings"
)

func StateReducer(s State, event Event) State {
	s.Input = inputReducer(s.Input, event)
	s.Messages = messagesReducer(s.Messages, event)

	switch e := event.(type) {
	case FrameEvent:
		s.Image = e.Image

	case ResizeEvent:
		s.WindowWidth = e.WinSize.Width
		s.WindowHeight = e.WinSize.Height
		s.WindowCols = e.WinSize.Cols
		s.WindowRows = e.WinSize.Rows

	case SetTitleEvent:
		s.Title = e.Title
	}

	return s
}

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func inputReducer(s string, event Event) string {
	switch e := event.(type) {
	case KeypressEvent:
		s += string(e.Char)

		// Strip ansi codes
		s = ansiRegex.ReplaceAllString(s, "")

		// Strip bell characters
		return strings.Replace(s, "\a", "", -1)

	case BackspaceEvent:
		if len(s) == 0 {
			return s
		}
		return s[:len(s)-1]

	case SentMessageEvent:
		return ""

	default:
		return s
	}
}

func messagesReducer(s []Message, event Event) []Message {
	switch e := event.(type) {
	case SentMessageEvent:
		return append(s, Message{
			User: "You",
			Text: e.Text,
		})

	case ReceivedChatEvent:
		return append(s, Message{
			User: "Them",
			Text: e.Text,
		})

	case ErrorEvent:
		return append(s, Message{
			User: "Info",
			Text: e.Text,
		})

	case InfoEvent:
		return append(s, Message{
			User: "Info",
			Text: e.Text,
		})

	default:
		return s
	}
}
