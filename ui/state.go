package ui

import (
	"image"

	"github.com/dialupdotcom/ascii_roulette/term"
)

type Page string

var (
	GlobePage   Page = "globe"
	PionPage    Page = "pion"
	ConfirmPage Page = "confirm"
	ChatPage    Page = "chat"
)

type State struct {
	Page Page

	Input    string
	Messages []Message
	Image    image.Image
	WinSize  term.WinSize
}

type Message struct {
	User string
	Text string
}
