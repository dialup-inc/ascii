package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/dialupdotcom/ascii_roulette/term"
	"github.com/dialupdotcom/ascii_roulette/vpx"
	"github.com/pion/webrtc/v2"
)

type demo struct {
	RTCConfig webrtc.Configuration

	width  int
	height int

	renderer *term.Renderer

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

	conn.OnMessage = func(m string) {
		d.renderer.Messages = append(d.renderer.Messages, term.Message{User: "Them", Text: m})
		d.renderer.RequestFrame()
	}

	go func() {
		time.Sleep(5 * time.Second)
		if err := d.capture.Start(camID, 5); err != nil {
			fmt.Println(err)
		}
		d.capture.RequestKeyframe()
	}()

	d.capture.SetTrack(conn.SendTrack)

	dec, err := vpx.NewDecoder()
	if err != nil {
		fmt.Println(err)
		cancel()
		return err
	}

	frameBuf := make([]byte, d.width*d.height*4)
	conn.OnFrame = func(frame []byte) {
		if err := d.decode(dec, frameBuf, frame); err != nil {
			conn.SendPLI()
		}
	}
	conn.OnPLI = func() {
		d.capture.RequestKeyframe()
	}

	if err := match(ctx, fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room), conn.pc); err != nil {
		cancel()
		return err
	}

	return err
}

func (d *demo) Stop() error {
	d.renderer.Stop()
	return nil
}

func (d *demo) decode(decoder *vpx.Decoder, frameBuf []byte, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}

	n, err := decoder.Decode(frameBuf, payload)
	if err != nil {
		return err
	}
	frame := frameBuf[:n]

	yi := d.width * d.height
	cbi := yi + d.width*d.height/4
	cri := cbi + d.width*d.height/4

	img := &image.YCbCr{
		Y:              frame[:yi],
		YStride:        d.width,
		Cb:             frame[yi:cbi],
		Cr:             frame[cbi:cri],
		CStride:        d.width / 2,
		SubsampleRatio: image.YCbCrSubsampleRatio420,
		Rect:           image.Rect(0, 0, d.width, d.height),
	}

	d.renderer.SetImage(img)

	return nil
}

func newDemo(width, height int) (*demo, error) {
	cap, err := NewCapture(width, height)
	if err != nil {
		return nil, err
	}

	d := &demo{
		width:    width,
		height:   height,
		renderer: term.NewRenderer(),
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

	fmt.Println("starting up...")

	demo, err := newDemo(640, 480)
	if err != nil {
		log.Fatal(err)
	}
	defer demo.Stop()
	demo.RTCConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	ansiRegex := regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

	var input string
	if err := CaptureStdin(func(c rune) {
		switch c {
		case 3: // ctrl-c
			os.Exit(0)
		case 127: // backspace
			if len(input) > 0 {
				input = input[:len(input)-1]
			}
			demo.renderer.SetInput(input)
		case '\n', '\r':
			demo.renderer.Messages = append(demo.renderer.Messages, term.Message{User: "You", Text: input})
			if demo.conn != nil {
				demo.conn.SendMessage(input)
			}
			input = ""
			demo.renderer.SetInput(input)
			// demo.renderer.SetMessages(messages)
		default:
			input += string(c)

			// Strip ansi codes
			input = ansiRegex.ReplaceAllString(input, "")

			demo.renderer.SetInput(input)
			// nothing for now
		}
	}); err != nil {
		log.Fatal(err)
	}

	if err := demo.Match(context.Background(), *camID, *signalerURL, *room); err != nil {
		fmt.Printf("Match error: %v\n", err)
		return
	}

	select {}
}
