package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/pions/asciirtc/capture"
	"github.com/pions/rtcp"
	"github.com/pions/rtp/codecs"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
	"github.com/pions/webrtc/pkg/media"
	"github.com/pions/webrtc/pkg/media/samplebuilder"
)

type demo struct {
	RTCConfig webrtc.RTCConfiguration

	width  int
	height int

	printer *Printer

	connMu sync.Mutex
	conn   *webrtc.RTCPeerConnection
}

func (d *demo) newConn() (*webrtc.RTCPeerConnection, error) {
	d.connMu.Lock()
	defer d.connMu.Unlock()

	if d.conn != nil {
		return nil, errors.New("another peer connection is connected")
	}

	conn, err := webrtc.New(d.RTCConfig)
	if err != nil {
		return nil, err
	}
	d.conn = conn

	return conn, nil
}

func (d *demo) Match(ctx context.Context, camID int, signalerURL string) error {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := d.newConn()
	if err != nil {
		cancel()
		return err
	}

	if err := capture.Start(camID, d.width, d.height); err != nil {
		cancel()
		return err
	}

	conn.OnICEConnectionStateChange(func(s ice.ConnectionState) {
		if s == ice.ConnectionStateClosed || s == ice.ConnectionStateFailed {
			d.connMu.Lock()
			if conn == d.conn {
				d.conn = nil
			}
			d.connMu.Unlock()

			cancel()

			if err := capture.Stop(); err != nil {
				fmt.Println(err)
			}
		}
	})

	track, err := conn.NewRTCSampleTrack(webrtc.DefaultPayloadTypeVP8, "video", "asciirtc")
	if err != nil {
		cancel()
		return err
	}
	if _, err := conn.AddTrack(track); err != nil {
		cancel()
		return err
	}

	var once sync.Once
	conn.OnTrack(func(track *webrtc.RTCTrack) {
		once.Do(func() {
			d.handleTrack(ctx, track)
		})
	})

	if err := match(ctx, fmt.Sprintf("ws://%s/ws", signalerURL), conn); err != nil {
		cancel()
		return err
	}
	fmt.Println("CONNECTED")

	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("Sending now...")

		ticker := time.Tick(40 * time.Millisecond)

		buf := make([]byte, 1<<24)
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := capture.ReadFrame(buf, true)
			if err != nil {
				fmt.Println(err)
				continue
			}

			samp := media.RTCSample{Data: buf[:n], Samples: 1}

			<-ticker

			track.Samples <- samp
		}
	}()

	return err
}

func (d *demo) handleTrack(ctx context.Context, track *webrtc.RTCTrack) {
	// Send PLIs every once in a while
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				pli := &rtcp.PictureLossIndication{MediaSSRC: track.Ssrc}
				if err := d.conn.SendRTCP(pli); err != nil {
					fmt.Println(err)
				}
			}
		}
	}()

	if err := capture.StartDecode(); err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := capture.StopDecode(); err != nil {
			fmt.Println(err)
		}
	}()

	builder := samplebuilder.New(256, &codecs.VP8Packet{})
	for j := 0; ; j++ {
		select {
		case <-ctx.Done():
			return
		case newPkt := <-track.Packets:
			builder.Push(newPkt)

			for s := builder.Pop(); s != nil; s = builder.Pop() {
				if err := d.decode(s.Data); err != nil {
					fmt.Println(err)
				}
			}

		}
	}
}

func (d *demo) decode(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}

	stride := d.width * 4
	frame := make([]byte, stride*d.height) // todo: less alloc

	if err := capture.DecodeFrame(frame, payload); err != nil {
		return err
	}

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

	d.printer.SetImage(img)

	return nil
}

func newDemo(width, height int) *demo {
	printer := NewPrinter()
	d := &demo{
		width:   width,
		height:  height,
		printer: printer,
	}
	printer.Start()
	return d
}

func main() {
	var (
		color       = flag.Bool("color", true, "whether to render image with colors")
		camID       = flag.Int("cam-id", 0, "cam-id used by OpenCV's VideoCapture.open()")
		signalerURL = flag.String("signaler-url", "asciirtc-signaler.pion.ly:8080", "host and port of the signaler")
	)

	flag.Parse()

	webrtc.RegisterDefaultCodecs()

	demo := newDemo(640, 480)
	demo.RTCConfig = webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	demo.printer.Colored = *color

	if err := demo.Match(context.Background(), *camID, *signalerURL); err != nil {
		fmt.Printf("Match error: %v\n", err)
	}

	select {}
}
