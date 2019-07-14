package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/dialupdotcom/ascii_roulette/signal"
	"github.com/dialupdotcom/ascii_roulette/ui"
	"github.com/dialupdotcom/ascii_roulette/vpx"
	"github.com/pion/webrtc/v2"
)

type App struct {
	RTCConfig webrtc.Configuration

	decoder *vpx.Decoder

	renderer *ui.Renderer
	state    ui.State

	conn *Conn

	capture *Capture
}

func (a *App) Connect(ctx context.Context, signalerURL, room string) (endReason string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	frameTimeout := time.NewTimer(999 * time.Hour)
	frameTimeout.Stop()

	connectTimeout := time.NewTimer(999 * time.Hour)
	connectTimeout.Stop()

	ended := make(chan string)

	conn, err := NewConn(a.RTCConfig)
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
		a.dispatch(ui.FrameEvent{nil})
		ended <- "Partner left"
	}
	conn.OnMessage = func(s string) {
		a.dispatch(ui.ReceivedChatEvent{s})
	}
	conn.OnICEConnectionStateChange = func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateConnected:
			a.capture.RequestKeyframe()
			connectTimeout.Stop()
			a.dispatch(ui.InfoEvent{"Connected (type ctrl-d to skip)"})

		case webrtc.ICEConnectionStateDisconnected:
			a.dispatch(ui.InfoEvent{"Reconnecting..."})

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
		a.dispatch(ui.FrameEvent{img})
	}
	conn.OnPLI = func() {
		a.capture.RequestKeyframe()
	}

	a.dispatch(ui.InfoEvent{"Searching for match..."})
	wsURL := fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room)
	if err := signal.Match(ctx, wsURL, conn.pc); err != nil {
		return "", err
	}

	connectTimeout.Reset(10 * time.Second)

	a.dispatch(ui.InfoEvent{"Found match. Connecting..."})

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

func (a *App) Stop() error {
	a.renderer.Stop()
	return nil
}

func (a *App) dispatch(e ui.Event) {
	newState := ui.StateReducer(a.state, e)
	if !reflect.DeepEqual(a.state, newState) {
		a.renderer.SetState(newState)
	}
	a.state = newState
}

func New() (*App, error) {
	cap, err := NewCapture(320, 240)
	if err != nil {
		return nil, err
	}

	a := &App{
		renderer: ui.NewRenderer(),
		capture:  cap,
	}
	a.renderer.Start()

	return a, nil
}
