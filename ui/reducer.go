package ui

import (
	"image"
	"regexp"
	"unicode"
	"unicode/utf8"

	"github.com/dialup-inc/ascii/term"
)

func StateReducer(s State, event Event) State {
	s.Image = imageReducer(s.Image, event)
	s.Input = inputReducer(s.Input, event)
	s.Messages = messagesReducer(s.Messages, event)
	s.Page = pageReducer(s.Page, event)
	s.WinSize = winSizeReducer(s.WinSize, event)

	return s
}

func pageReducer(s Page, event Event) Page {
	switch e := event.(type) {
	case SetPageEvent:
		return Page(e)
	default:
		return s
	}
}

func winSizeReducer(s term.WinSize, event Event) term.WinSize {
	switch e := event.(type) {
	case ResizeEvent:
		return term.WinSize(e)
	default:
		return s
	}
}

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

func inputReducer(s string, event Event) string {
	switch e := event.(type) {
	case KeypressEvent:
		s += string(e)

		// Strip ANSI escape codes
		s = ansiRegex.ReplaceAllString(s, "")

		// Strip unprintable characters
		var printable []rune
		for _, r := range s {
			if !unicode.IsPrint(r) {
				continue
			}
			printable = append(printable, r)
		}
		s = string(printable)

		return s

	case BackspaceEvent:
		if len(s) == 0 {
			return s
		}
		_, sz := utf8.DecodeLastRuneInString(s)
		return s[:len(s)-sz]

	case SentMessageEvent:
		return ""

	default:
		return s
	}
}

func imageReducer(s image.Image, event Event) image.Image {
	switch e := event.(type) {
	case FrameEvent:
		return image.Image(e)

	case SetPageEvent:
		return nil

	default:
		return s
	}
}

func messagesReducer(s []Message, event Event) []Message {
	switch e := event.(type) {
	case SentMessageEvent:
		return append(s, Message{
			User: "You",
			Text: string(e),
		})

	case ReceivedChatEvent:
		return append(s, Message{
			User: "Them",
			Text: string(e),
		})

	case LogEvent:
		user := "Log"
		switch e.Level {
		case LogLevelError:
			user = "Error"
		case LogLevelInfo:
			user = "Info"
		}

		return append(s, Message{
			User: user,
			Text: e.Text,
		})

	default:
		return s
	}
}
