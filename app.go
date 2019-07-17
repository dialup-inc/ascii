package ascii

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/dialup-inc/ascii/term"
	"github.com/dialup-inc/ascii/ui"
	"github.com/dialup-inc/ascii/videos"
	"github.com/dialup-inc/ascii/vpx"
	"github.com/pion/stun"
	"github.com/pion/webrtc/v2"
)

const defaultSTUNServer = "stun.l.google.com:19302"

type App struct {
	STUNServer string

	decoder *vpx.Decoder

	signalerURL string

	cancelMu    sync.Mutex
	quit        context.CancelFunc
	skipIntro   context.CancelFunc
	nextPartner context.CancelFunc
	startChat   context.CancelFunc

	renderer *ui.Renderer

	conn *Conn

	capture *Capture
}

func (a *App) run(ctx context.Context) error {
	a.cancelMu.Lock()
	if a.quit != nil {
		a.cancelMu.Unlock()
		return errors.New("app can only be run once")
	}
	ctx, cancel := context.WithCancel(ctx)
	a.quit = cancel
	a.cancelMu.Unlock()

	if err := term.CaptureStdin(a.onKeypress); err != nil {
		return err
	}

	winSize, _ := term.GetWinSize()
	if winSize.Rows < 15 || winSize.Cols < 50 {
		ansi := term.ANSI{os.Stdout}
		ansi.ResizeWindow(15, 50)
	}

	go a.watchWinSize(ctx)

	var introCtx context.Context
	introCtx, skipIntro := context.WithCancel(ctx)

	a.cancelMu.Lock()
	a.skipIntro = skipIntro
	a.cancelMu.Unlock()

	// Play Dialup intro
	a.renderer.Dispatch(ui.SetPageEvent(ui.GlobePage))

	player, err := videos.NewPlayer(videos.Globe())
	if err != nil {
		log.Fatal(err)
	}
	player.OnFrame = func(img image.Image) {
		a.renderer.Dispatch(ui.FrameEvent(img))
	}
	player.Play(introCtx)

	// Play Pion intro
	a.renderer.Dispatch(ui.SetPageEvent(ui.PionPage))

	player, err = videos.NewPlayer(videos.Pion())
	if err != nil {
		log.Fatal(err)
	}
	player.OnFrame = func(img image.Image) {
		a.renderer.Dispatch(ui.FrameEvent(img))
	}
	player.Play(introCtx)

	// Show confirmation page
	a.renderer.Dispatch(ui.SetPageEvent(ui.ConfirmPage))

	startCtx, startChat := context.WithCancel(ctx)

	a.cancelMu.Lock()
	a.startChat = startChat
	a.cancelMu.Unlock()

	<-startCtx.Done()

	// give user a chance to quit
	if err := ctx.Err(); err != nil {
		return nil
	}

	a.renderer.Dispatch(ui.SetPageEvent(ui.ChatPage))

	// Start up camera
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		err := a.capture.Start(0, 5)
		if err == nil {
			break
		}
		msg := fmt.Sprintf("camera error: %v", err)
		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelError,
			Text:  msg,
		})

		select {
		case <-time.After(1500 * time.Millisecond):
			continue
		case <-ctx.Done():
			return nil
		}
	}

	// Attempt to find match
	var backoff float64
	for {
		if err := ctx.Err(); err != nil {
			break
		}

		var connCtx context.Context
		connCtx, nextPartner := context.WithCancel(ctx)

		a.cancelMu.Lock()
		a.nextPartner = nextPartner
		a.cancelMu.Unlock()

		endReason, err := a.connect(connCtx)
		// HACK(maxhawkins): these errors get returned when the context passed
		// into match is canceled, so we ignore them. There's probably a more elegant
		// way to close the websocket without all this error munging.
		if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
			continue
		} else if err != nil && strings.Contains(err.Error(), "operation was canceled") {
			continue
		} else if err != nil {
			a.renderer.Dispatch(ui.LogEvent{
				Level: ui.LogLevelError,
				Text:  err.Error(),
			})

			sec := math.Pow(2, backoff) - 1
			if backoff < 4 {
				backoff++
			}

			select {
			case <-time.After(time.Duration(sec) * time.Second):
				continue
			case <-ctx.Done():
				return nil
			}
		}

		a.renderer.Dispatch(ui.ConnEndedEvent{endReason})
		a.renderer.Dispatch(ui.FrameEvent(nil))

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (a *App) catchError(msg interface{}, stack []byte) {
	buf := bytes.NewBuffer(nil)
	ansi := term.ANSI{buf}

	ansi.CursorPosition(1, 1)
	ansi.Reset()

	ansi.Bold()
	ansi.Foreground(color.RGBA{0xFF, 0x00, 0x00, 0xFF})
	buf.WriteString("Oops! ASCII Roulette hit a snag.\n")
	ansi.Normal()
	ansi.ForegroundReset()

	buf.WriteString("\n")
	buf.WriteString("Please report this error at https://github.com/dialup-inc/ascii/issues/new/choose\n")
	buf.WriteString("\n")
	buf.WriteString("\n")

	buf.WriteString(fmt.Sprintf("[panic] %v\n", msg))
	buf.WriteString("\n")
	buf.Write(stack)
	buf.WriteString("\n")

	data := bytes.ReplaceAll(buf.Bytes(), []byte("\n"), []byte("\r\n"))
	os.Stderr.Write(data)
}

func (a *App) Run(ctx context.Context) error {
	// Show a nice error page if there's a panic somewhere in the code
	defer func() {
		if r := recover(); r != nil {
			a.catchError(r, debug.Stack())
		}
	}()

	err := a.run(ctx)

	// Clean up:
	a.renderer.Stop()
	if a.conn != nil && a.conn.IsConnected() {
		a.conn.SendBye()
	}

	return err
}

func (a *App) watchWinSize(ctx context.Context) error {
	checkWinSize := func() {
		winSize, err := term.GetWinSize()
		if err != nil {
			return
		}
		a.renderer.Dispatch(ui.ResizeEvent(winSize))
	}

	checkWinSize()

	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick:
			checkWinSize()
		}
	}
}

func (a *App) sendMessage() {
	if a.conn == nil || !a.conn.IsConnected() {
		return
	}

	msg := a.renderer.GetState().Input
	msg = strings.TrimSpace(msg)

	// Don't send empty messages
	if len(msg) == 0 {
		return
	}

	if err := a.conn.SendMessage(msg); err != nil {
		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelError,
			Text:  fmt.Sprintf("sending failed: %v", err),
		})
	} else {
		a.renderer.Dispatch(ui.SentMessageEvent(msg))
	}
}

func (a *App) checkConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	c, err := stun.Dial("udp", a.STUNServer)
	if err != nil {
		return errors.New("connection error: check your firewall or network")
	}
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	errChan := make(chan error)

	go c.Do(message, func(res stun.Event) {
		errChan <- res.Error
	})

	select {
	case err := <-errChan:
		if err != nil {
			return errors.New("binding request failed: firewall may be blocking connections")
		}
		return nil

	case <-ctx.Done():
		switch ctx.Err() {
		case context.Canceled:
			return err
		case context.DeadlineExceeded:
			return errors.New("binding request failed: firewall may be blocking connections")
		default:
			return nil
		}
	}
}

func (a *App) connect(ctx context.Context) (ui.EndConnReason, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	frameTimeout := time.NewTimer(999 * time.Hour)
	frameTimeout.Stop()

	connectTimeout := time.NewTimer(999 * time.Hour)
	connectTimeout.Stop()

	if err := a.checkConnection(ctx); err != nil {
		return ui.EndConnSetupError, err
	}

	ended := make(chan ui.EndConnReason)

	conn, err := NewConn(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{fmt.Sprintf("stun:%s", a.STUNServer)}},
		},
	})
	if err != nil {
		return ui.EndConnSetupError, err
	}
	a.conn = conn

	defer func() {
		// Turn off callbacks
		conn.OnBye = func() {}
		conn.OnMessage = func(string) {}
		conn.OnICEConnectionStateChange = func(webrtc.ICEConnectionState) {}
		conn.OnFrame = func([]byte) {}
		conn.OnPLI = func() {}
		conn.OnDataOpen = func() {}

		// Send Goodbye packet
		if conn.IsConnected() {
			conn.SendBye()
		}
	}()

	conn.OnBye = func() {
		a.renderer.Dispatch(ui.FrameEvent(nil))
		ended <- ui.EndConnGone
	}
	conn.OnMessage = func(s string) {
		a.renderer.Dispatch(ui.ReceivedChatEvent(s))
	}
	conn.OnICEConnectionStateChange = func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateConnected:
			a.capture.RequestKeyframe()
			connectTimeout.Stop()
			a.renderer.Dispatch(ui.ConnStartedEvent{})

		case webrtc.ICEConnectionStateDisconnected:
			a.renderer.Dispatch(ui.LogEvent{
				Level: ui.LogLevelInfo,
				Text:  "Reconnecting...",
			})

		case webrtc.ICEConnectionStateFailed:
			ended <- ui.EndConnDisconnected
		}
	}
	conn.OnDataOpen = func() {
		a.renderer.Dispatch(ui.DataOpenedEvent{})
	}

	a.capture.SetTrack(conn.SendTrack)

	dec, err := vpx.NewDecoder(320, 240)
	if err != nil {
		return ui.EndConnSetupError, err
	}
	conn.OnFrame = func(frame []byte) {
		frameTimeout.Reset(5 * time.Second)
		connectTimeout.Stop()

		img, err := dec.Decode(frame)
		if err != nil {
			conn.SendPLI()
			return
		}
		a.renderer.Dispatch(ui.FrameEvent(img))
	}
	conn.OnPLI = func() {
		a.capture.RequestKeyframe()
	}

	a.renderer.Dispatch(ui.LogEvent{
		Level: ui.LogLevelInfo,
		Text:  "Searching for match...",
	})

	err = Match(ctx, a.signalerURL, conn.pc)
	if err == errMatchFailed {
		return ui.EndConnMatchError, nil
	}
	if err != nil {
		return ui.EndConnMatchError, err
	}

	connectTimeout.Reset(10 * time.Second)

	a.renderer.Dispatch(ui.LogEvent{
		Level: ui.LogLevelInfo,
		Text:  "Found match. Connecting...",
	})

	var reason ui.EndConnReason
	select {
	case <-ctx.Done():
		reason = ui.EndConnNormal
	case <-connectTimeout.C:
		reason = ui.EndConnTimedOut
	case <-frameTimeout.C:
		reason = ui.EndConnDisconnected
	case r := <-ended:
		reason = r
	}

	return reason, nil
}

func (a *App) onKeypress(c rune) {
	switch c {
	case 3: // ctrl-c

		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelInfo,
			Text:  "Quitting...",
		})

		a.cancelMu.Lock()
		if a.quit != nil {
			a.quit()
		}
		a.cancelMu.Unlock()

	case 4: // ctrl-d
		a.cancelMu.Lock()
		if a.nextPartner != nil {
			a.nextPartner()
			a.nextPartner = nil
		}
		a.cancelMu.Unlock()

		a.renderer.Dispatch(ui.SkipEvent{})

	case 20: // ctrl-t
		a.renderer.Dispatch(ui.ToggleHelpEvent{})

	case 127: // backspace
		a.renderer.Dispatch(ui.BackspaceEvent{})

	case '\n', '\r':
		a.cancelMu.Lock()
		if a.startChat != nil {
			a.startChat()
			a.startChat = nil
		}
		a.cancelMu.Unlock()

		a.sendMessage()

	case ' ':
		a.renderer.Dispatch(ui.KeypressEvent(c))

		a.cancelMu.Lock()
		if a.skipIntro != nil {
			a.skipIntro()
			a.skipIntro = nil
		}
		if a.startChat != nil {
			a.startChat()
			a.startChat = nil
		}
		a.cancelMu.Unlock()

	default:
		a.renderer.Dispatch(ui.KeypressEvent(c))
	}
}

func New(signalerURL string) (*App, error) {
	cap, err := NewCapture(320, 240)
	if err != nil {
		return nil, err
	}

	a := &App{
		signalerURL: signalerURL,
		STUNServer:  defaultSTUNServer,

		renderer: ui.NewRenderer(),
		capture:  cap,
	}
	a.renderer.Start()

	return a, nil
}
