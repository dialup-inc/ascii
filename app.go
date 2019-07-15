package roulette

import (
	"context"
	"errors"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dialupdotcom/ascii_roulette/signal"
	"github.com/dialupdotcom/ascii_roulette/term"
	"github.com/dialupdotcom/ascii_roulette/ui"
	"github.com/dialupdotcom/ascii_roulette/videos"
	"github.com/dialupdotcom/ascii_roulette/vpx"
	"github.com/pion/stun"
	"github.com/pion/webrtc/v2"
)

const defaultSTUNServer = "stun.l.google.com:19302"

type App struct {
	STUNServer string

	decoder *vpx.Decoder

	signalerURL string
	room        string

	cancelMu    sync.Mutex
	quit        context.CancelFunc
	skipIntro   context.CancelFunc
	nextPartner context.CancelFunc

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

	// Attempt to find match
	a.renderer.Dispatch(ui.SetPageEvent(ui.ChatPage))

	if err := a.capture.Start(0, 5); err != nil {
		msg := fmt.Sprintf("camera error: %v", err)
		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelError,
			Text:  msg,
		})
		// TODO: show in ui and retry
		return err
	}

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

		skipReason, err := a.connect(connCtx, a.signalerURL, a.room)
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
			time.Sleep(time.Duration(sec) * time.Second)
			if backoff < 4 {
				backoff++
			}
			continue
		}

		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelInfo,
			Text:  skipReason,
		})
		a.renderer.Dispatch(ui.FrameEvent(nil))

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (a *App) Run(ctx context.Context) error {
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
	if err := a.conn.SendMessage(msg); err != nil {
		a.renderer.Dispatch(ui.LogEvent{
			Level: ui.LogLevelError,
			Text:  "sending message failed",
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

func (a *App) connect(ctx context.Context, signalerURL, room string) (endReason string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	frameTimeout := time.NewTimer(999 * time.Hour)
	frameTimeout.Stop()

	connectTimeout := time.NewTimer(999 * time.Hour)
	connectTimeout.Stop()

	if err := a.checkConnection(ctx); err != nil {
		return "", err
	}

	ended := make(chan string)

	conn, err := NewConn(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{fmt.Sprintf("stun:%s", a.STUNServer)}},
		},
	})
	if err != nil {
		return "", err
	}
	a.conn = conn

	defer func() {
		// Turn off callbacks
		conn.OnBye = func() {}
		conn.OnMessage = func(string) {}
		conn.OnICEConnectionStateChange = func(webrtc.ICEConnectionState) {}
		conn.OnFrame = func([]byte) {}
		conn.OnPLI = func() {}

		// Send Goodbye packet
		if conn.IsConnected() {
			conn.SendBye()
		}
	}()

	conn.OnBye = func() {
		a.renderer.Dispatch(ui.FrameEvent(nil))
		ended <- "Partner left"
	}
	conn.OnMessage = func(s string) {
		a.renderer.Dispatch(ui.ReceivedChatEvent(s))
	}
	conn.OnICEConnectionStateChange = func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateConnected:
			a.capture.RequestKeyframe()
			connectTimeout.Stop()
			a.renderer.Dispatch(ui.LogEvent{
				Level: ui.LogLevelInfo,
				Text:  "Connected (type ctrl-d to skip)",
			})

		case webrtc.ICEConnectionStateDisconnected:
			a.renderer.Dispatch(ui.LogEvent{
				Level: ui.LogLevelInfo,
				Text:  "Reconnecting...",
			})

		case webrtc.ICEConnectionStateFailed:
			ended <- "Lost connection"
		}
	}

	a.capture.SetTrack(conn.SendTrack)

	dec, err := vpx.NewDecoder(320, 240)
	if err != nil {
		return "", err
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
	wsURL := fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room)
	if err := signal.Match(ctx, wsURL, conn.pc); err != nil {
		return "", err
	}

	connectTimeout.Reset(10 * time.Second)

	a.renderer.Dispatch(ui.LogEvent{
		Level: ui.LogLevelInfo,
		Text:  "Found match. Connecting...",
	})

	select {
	case <-ctx.Done():
		return "", nil
	case <-connectTimeout.C:
		return "Connection timed out", nil
	case <-frameTimeout.C:
		return "Lost connection", nil
	case reason := <-ended:
		return reason, nil
	}
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

	case 127: // backspace
		a.renderer.Dispatch(ui.BackspaceEvent{})

	case '\n', '\r':
		a.sendMessage()

	case ' ':
		a.cancelMu.Lock()
		if a.skipIntro != nil {
			a.skipIntro()
			a.skipIntro = nil
		}
		a.cancelMu.Unlock()

		a.renderer.Dispatch(ui.KeypressEvent(c))

	default:
		a.renderer.Dispatch(ui.KeypressEvent(c))
	}
}

func New(signalerURL, room string) (*App, error) {
	cap, err := NewCapture(320, 240)
	if err != nil {
		return nil, err
	}

	a := &App{
		signalerURL: signalerURL,
		room:        room,
		STUNServer:  defaultSTUNServer,

		renderer: ui.NewRenderer(),
		capture:  cap,
	}
	a.renderer.Start()

	return a, nil
}
