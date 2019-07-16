package ui

import (
	"image"

	"github.com/dialup-inc/ascii/term"
)

// An Event represents a user action that cause changes in the UI state.
//
// They're processed by Renderer's Dispatch method.
type Event interface{}

// SentMessageEvent fires after a message has been sent to the current partner
type SentMessageEvent string

// FrameEvent is sent when the video decoder renders a new frame
type FrameEvent image.Image

// ReceivedChatEvent is fired when the user submits text in the chat input box.
type ReceivedChatEvent string

// KeypressEvent is fired when the user presses the keyboard.
type KeypressEvent rune

// BackspaceEvent is fired when the backspace button is pressed.
type BackspaceEvent struct{}

// ResizeEvent indicates that the terminal window's size has changed to the specified dimensions
type ResizeEvent term.WinSize

// ToggleHelpEvent toggles the help modal
type ToggleHelpEvent struct{}

// SkipEvent skips to the next match
type SkipEvent struct{}

// DataOpenedEvent fires when the text chat data channel opens and the user can begin typing
type DataOpenedEvent struct{}

// ConnStartedEvent fires when an ICE connection has been established and the call can begin
type ConnStartedEvent struct{}

// ConnEndedEvent fires when the connection with your partner has been lost
type ConnEndedEvent struct {
	// The reason for the disconnection
	Reason string
}

// SetPageEvent transitions to the specified page
type SetPageEvent Page

// LogLevel indicates the severity of a LogEvent message
type LogLevel int

const (
	// LogLevelInfo is for non-urgent, informational logs
	LogLevelInfo LogLevel = iota
	// LogLevelError is for logs that indicate problems
	LogLevelError
)

// A LogEvent prints a message to the console
type LogEvent struct {
	Text  string
	Level LogLevel
}
