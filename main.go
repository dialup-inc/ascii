package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"reflect"
	"time"

	"github.com/dialupdotcom/ascii_roulette/term"
	"github.com/dialupdotcom/ascii_roulette/videos"
	"github.com/pion/webrtc/v2"
)

type demo struct {
	RTCConfig webrtc.Configuration

	decoder Decoder

	renderer *Renderer
	state    State

	conn *Conn

	capture *Capture
}

func (d *demo) Connect(ctx context.Context, signalerURL, room string) (endReason string, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	frameTimeout := time.NewTimer(999 * time.Hour)
	frameTimeout.Stop()

	connectTimeout := time.NewTimer(999 * time.Hour)
	connectTimeout.Stop()

	ended := make(chan string)

	conn, err := NewConn(d.RTCConfig)
	if err != nil {
		return "", err
	}
	d.conn = conn

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
		d.dispatch(FrameEvent{nil})
		ended <- "Partner left"
	}
	conn.OnMessage = func(s string) {
		fmt.Print("\a") // beep
		d.dispatch(ReceivedChatEvent{s})
	}
	conn.OnICEConnectionStateChange = func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateConnected:
			d.capture.RequestKeyframe()
			connectTimeout.Stop()
			d.dispatch(InfoEvent{"Connected (type ctrl-d to skip)"})

		case webrtc.ICEConnectionStateDisconnected:
			d.dispatch(InfoEvent{"Reconnecting..."})

		case webrtc.ICEConnectionStateFailed:
			ended <- "Lost connection"
		}
	}

	d.capture.SetTrack(conn.SendTrack)

	dec, err := NewDecoder(320, 240)
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
		d.dispatch(FrameEvent{img})
	}
	conn.OnPLI = func() {
		d.capture.RequestKeyframe()
	}

	d.dispatch(InfoEvent{"Searching for match..."})
	wsURL := fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room)
	if err := match(ctx, wsURL, conn.pc); err != nil {
		return "", err
	}

	connectTimeout.Reset(10 * time.Second)

	d.dispatch(InfoEvent{"Found match. Connecting..."})

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

func (d *demo) Stop() error {
	d.renderer.Stop()
	return nil
}

func (d *demo) dispatch(e Event) {
	newState := e.Do(d.state)
	if !reflect.DeepEqual(d.state, newState) {
		d.renderer.SetState(newState)
	}
	d.state = newState
}

func newDemo() (*demo, error) {
	cap, err := NewCapture(320, 240)
	if err != nil {
		return nil, err
	}

	d := &demo{
		renderer: NewRenderer(),
		capture:  cap,
	}
	d.renderer.Start()

	return d, nil
}

func main() {
	var (
		camID       = flag.Int("cam-id", 0, "cam-id used by OpenCV's VideoCapture.open()")
		signalerURL = flag.String("signaler-url", "asciirtc-signaler.pion.ly:8080", "host and port of the signaler")
		room        = flag.String("room", "main", "Name of room to join ")
	)
	flag.Parse()

	demo, err := newDemo()
	if err != nil {
		log.Fatal(err)
	}

	demo.RTCConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	checkWinSize := func() {
		winSize, err := term.GetWinSize()
		if err != nil {
			return
		}
		demo.dispatch(ResizeEvent{winSize})
	}
	checkWinSize()
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			checkWinSize()
		}
	}()

	var skipIntro func()
	var nextPartner func()

	sendMessage := func() {
		if demo.conn == nil || !demo.conn.IsConnected() {
			return
		}

		msg := demo.state.Input
		if err := demo.conn.SendMessage(msg); err != nil {
			demo.dispatch(ErrorEvent{"sending message failed"})
		} else {
			demo.dispatch(SentMessageEvent{msg})
		}
	}

	quitChan := make(chan struct{})

	if err := CaptureStdin(func(c rune) {
		switch c {
		case 3: // ctrl-c
			quitChan <- struct{}{}
		case 4: // ctrl-d
			if nextPartner != nil {
				nextPartner()
				nextPartner = nil
			}
		case 127: // backspace
			demo.dispatch(BackspaceEvent{})
		case '\n', '\r':
			sendMessage()
		case ' ':
			if skipIntro != nil {
				skipIntro()
				skipIntro = nil
			}
			demo.dispatch(KeypressEvent{c})
		default:
			demo.dispatch(KeypressEvent{c})
		}
	}); err != nil {
		log.Fatal(err)
	}

	go func() {
		var introCtx context.Context
		introCtx, skipIntro = context.WithCancel(context.Background())

		// Play Dialup intro
		demo.dispatch(SetTitleEvent{"Presented by dialup.com\n(we're hiring!)"})

		player, err := NewPlayer(videos.Globe())
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			demo.dispatch(FrameEvent{img})
		}
		player.Play(introCtx)

		// Play Pion intro
		demo.dispatch(SetTitleEvent{"Powered by Pion"})

		player, err = NewPlayer(videos.Pion())
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			demo.dispatch(FrameEvent{img})
		}
		player.Play(introCtx)

		// Attempt to find match
		demo.dispatch(FrameEvent{nil})
		demo.dispatch(SetTitleEvent{""})

		if err := demo.capture.Start(*camID, 5); err != nil {
			msg := fmt.Sprintf("camera error: %v", err)
			demo.dispatch(ErrorEvent{msg})
			// TODO: show in ui and retry
			return
		}

		var backoff float64
		for {
			var connCtx context.Context
			connCtx, nextPartner = context.WithCancel(context.Background())

			skipReason, err := demo.Connect(connCtx, *signalerURL, *room)
			if err != nil {
				demo.dispatch(ErrorEvent{err.Error()})

				sec := math.Pow(2, backoff) - 1
				time.Sleep(time.Duration(sec) * time.Second)
				if backoff < 4 {
					backoff++
				}
				continue
			}
			demo.dispatch(InfoEvent{skipReason})
			demo.dispatch(FrameEvent{nil})

			time.Sleep(100 * time.Millisecond)
		}
	}()

	<-quitChan
	if demo.conn != nil && demo.conn.IsConnected() {
		demo.conn.SendBye()
	}
	demo.Stop()
}
