package main

import (
	"image"
	"regexp"

	"github.com/dialupdotcom/ascii_roulette/term"
)

var ansiRegex = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

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

type Event interface {
	Do(State) State
}

type KeypressEvent struct {
	Char rune
}

func (e KeypressEvent) Do(s State) State {
	s.Input += string(e.Char)

	// Strip ansi codes
	s.Input = ansiRegex.ReplaceAllString(s.Input, "")

	return s
}

type SentMessageEvent struct {
	Text string
}

func (e SentMessageEvent) Do(s State) State {
	s.Messages = append(s.Messages, Message{
		User: "You",
		Text: e.Text,
	})
	s.Input = ""
	return s
}

type FrameEvent struct {
	Image image.Image
}

func (e FrameEvent) Do(s State) State {
	s.Image = e.Image
	return s
}

type ReceivedChatEvent struct {
	Text string
}

func (e ReceivedChatEvent) Do(s State) State {
	s.Messages = append(s.Messages, Message{
		User: "Them",
		Text: e.Text,
	})
	return s
}

type BackspaceEvent struct{}

func (e BackspaceEvent) Do(s State) State {
	if len(s.Input) == 0 {
		return s
	}
	s.Input = s.Input[:len(s.Input)-1]
	return s
}

type ResizeEvent struct {
	WinSize term.WinSize
}

func (e ResizeEvent) Do(s State) State {
	s.WindowWidth = e.WinSize.Width
	s.WindowHeight = e.WinSize.Height
	s.WindowCols = e.WinSize.Cols
	s.WindowRows = e.WinSize.Rows
	return s
}

type ErrorEvent struct {
	Text string
}

func (e ErrorEvent) Do(s State) State {
	s.Messages = append(s.Messages, Message{
		User: "Info",
		Text: e.Text,
	})
	return s
}

type InfoEvent struct {
	Text string
}

func (e InfoEvent) Do(s State) State {
	s.Messages = append(s.Messages, Message{
		User: "Info",
		Text: e.Text,
	})
	return s
}

type SetTitleEvent struct {
	Title string
}

func (e SetTitleEvent) Do(s State) State {
	s.Title = e.Title
	return s
}
