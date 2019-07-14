package ui

import (
	"image"

	"github.com/dialupdotcom/ascii_roulette/term"
)

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

type SetPageEvent Page
