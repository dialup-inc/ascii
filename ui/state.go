package ui

import (
	"image"

	"github.com/dialup-inc/ascii/term"
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

	HelpOn bool

	Input      string
	ChatActive bool

	Messages []Message
	Image    image.Image
	WinSize  term.WinSize
}

type MessageType int

const (
	MessageTypeIncoming MessageType = iota
	MessageTypeOutgoing
	MessageTypeInfo
	MessageTypeError
)

type Message struct {
	Type MessageType
	User string
	Text string
}

type EndConnReason int

const (
	// User closed connection
	EndConnNormal EndConnReason = iota
	// Error during connection setup
	EndConnSetupError
	// Error during matching
	EndConnMatchError
	// Connection timed out
	EndConnTimedOut
	// Lost connection with partner
	EndConnDisconnected
	// Partner left
	EndConnGone
)
