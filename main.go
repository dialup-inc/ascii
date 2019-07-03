package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/dialupdotcom/ascii_roulette/term"
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

func (d *demo) Match(ctx context.Context, camID int, signalerURL, room string) error {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := NewConn(d.RTCConfig)
	if err != nil {
		cancel()
		return err
	}
	d.conn = conn

	conn.OnBye = func() {
		d.dispatch(TypeInfo, "Partner left")
	}
	conn.OnMessage = func(s string) {
		d.dispatch(TypeReceivedChat, s)
	}
	conn.OnICEConnectionStateChange = func(s webrtc.ICEConnectionState) {
		switch s {
		case webrtc.ICEConnectionStateChecking:
			d.dispatch(TypeInfo, "Connecting...")

		case webrtc.ICEConnectionStateConnected:
			d.capture.RequestKeyframe()
			d.dispatch(TypeInfo, "Connected")

		case webrtc.ICEConnectionStateDisconnected:
			d.dispatch(TypeInfo, "Reconnecting...")

		case webrtc.ICEConnectionStateFailed:
			d.dispatch(TypeInfo, "Lost connection")
		}
	}

	go func() {
		if err := d.capture.Start(camID, 5); err != nil {
			d.dispatch(TypeError, fmt.Sprintf("camera error: %v", err))
			return
		}
	}()

	d.capture.SetTrack(conn.SendTrack)

	dec, err := NewDecoder(320, 240)
	if err != nil {
		cancel()
		return err
	}
	conn.OnFrame = func(frame []byte) {
		img, err := dec.Decode(frame)
		if err != nil {
			conn.SendPLI()
			return
		}
		d.dispatch(TypeFrame, img)
	}
	conn.OnPLI = func() {
		d.capture.RequestKeyframe()
	}

	d.dispatch(TypeInfo, "Searching for match...")
	if err := match(ctx, fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room), conn.pc); err != nil {
		cancel()
		return err
	}

	d.dispatch(TypeInfo, "Found match")

	cancel()
	return err
}

func (d *demo) Stop() error {
	d.renderer.Stop()
	return nil
}

func (d *demo) dispatch(t EventType, payload interface{}) {
	newState := StateReducer(d.state, Event{t, payload})
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
		room        = flag.String("room", "pion5", "Name of room to join ")
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
		ws, err := term.GetWinSize()
		if err != nil {
			return
		}
		demo.dispatch(TypeResize, ws)
	}
	checkWinSize()
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			checkWinSize()
		}
	}()

	sendMessage := func() {
		if demo.conn == nil || !demo.conn.IsConnected() {
			return
		}

		msg := demo.state.Input
		if err := demo.conn.SendMessage(msg); err != nil {
			demo.dispatch(TypeError, "sending message failed")
		} else {
			demo.dispatch(TypeSentMessage, msg)
		}
	}

	quitChan := make(chan struct{})

	if err := CaptureStdin(func(c rune) {
		switch c {
		case 3, 4: // ctrl-c, ctrl-d
			quitChan <- struct{}{}
		case 14: // ctrl-n

		case 127: // backspace
			demo.dispatch(TypeBackspace, nil)
		case '\n', '\r':
			sendMessage()
		default:
			demo.dispatch(TypeKeypress, c)
		}
	}); err != nil {
		log.Fatal(err)
	}

	go func() {
		demo.dispatch(TypeSetTitle, "Presented by dialup.com\n(we're hiring!)")

		f, err := os.Open("videos/globe.ivf")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		player, err := NewPlayer(f)
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			demo.dispatch(TypeFrame, img)
		}
		if err := player.Play(context.Background()); err != nil {
			demo.dispatch(TypeError, err.Error())
		}

		demo.dispatch(TypeSetTitle, "Powered by Pion")

		f, err = os.Open("videos/pion.ivf")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		player, err = NewPlayer(f)
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			demo.dispatch(TypeFrame, img)
		}
		if err := player.Play(context.Background()); err != nil {
			demo.dispatch(TypeError, err.Error())
		}

		demo.dispatch(TypeSetTitle, "")

		if err := demo.Match(context.Background(), *camID, *signalerURL, *room); err != nil {
			demo.dispatch(TypeError, fmt.Sprintf("match error: %v", err))
			return
		}
	}()

	<-quitChan
	if demo.conn != nil && demo.conn.IsConnected() {
		demo.conn.SendBye()
	}
	demo.Stop()
}
