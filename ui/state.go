package ui

import (
	"image"
)

type State struct {
	Page string

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
