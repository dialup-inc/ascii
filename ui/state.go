package ui

import (
	"image"
)

type Page string

var (
	GlobePage Page = "globe"
	PionPage  Page = "pion"
	ChatPage  Page = "chat"
)

type State struct {
	Page Page

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
