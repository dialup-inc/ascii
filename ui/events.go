package ui

import (
	"image"

	"github.com/dialupdotcom/ascii_roulette/term"
)

type Event interface{}

type KeypressEvent rune

type SentMessageEvent string

type FrameEvent image.Image

type ReceivedChatEvent string

type BackspaceEvent struct{}

type ResizeEvent term.WinSize

type ErrorEvent string

type InfoEvent string

type SetPageEvent Page
