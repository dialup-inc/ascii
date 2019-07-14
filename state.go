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

type Event interface{}

type KeypressEvent struct {
	Char rune
}

type SentMessageEvent struct {
	Text string
}

type FrameEvent struct {
	Image image.Image
}

type ReceivedChatEvent struct {
	Text string
}

type BackspaceEvent struct{}

type ResizeEvent struct {
	WinSize term.WinSize
}

type ErrorEvent struct {
	Text string
}

type InfoEvent struct {
	Text string
}

type SetTitleEvent struct {
	Title string
}
