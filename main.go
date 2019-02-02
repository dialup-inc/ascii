package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"sync"
	"time"

	"github.com/maxhawkins/asciirtc/capture"
	"github.com/pions/rtcp"
	"github.com/pions/rtp/codecs"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
	"github.com/pions/webrtc/pkg/media"
	"github.com/pions/webrtc/pkg/media/samplebuilder"
)

type Demo struct {
	RTCConfig webrtc.RTCConfiguration

	width  int
	height int

	printer *Printer

	connMu sync.Mutex
	conn   *webrtc.RTCPeerConnection
}

func (d *Demo) newConn() (*webrtc.RTCPeerConnection, error) {
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

func (d *Demo) Match(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := d.newConn()
	if err != nil {
		cancel()
		return err
	}

	if err := capture.Start(d.width, d.height); err != nil {
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

	if err := Match(ctx, "ws://localhost:8080/ws", conn); err != nil {
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

func (d *Demo) handleTrack(ctx context.Context, track *webrtc.RTCTrack) {
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
	defer capture.StopDecode()

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

func (d *Demo) decode(payload []byte) error {
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

func NewDemo(width, height int) *Demo {
	printer := NewPrinter()
	d := &Demo{
		width:   width,
		height:  height,
		printer: printer,
	}
	printer.Start()
	return d
}

func main() {
	var (
		color = flag.Bool("color", true, "whether to render image with colors")
	)
	flag.Parse()

	webrtc.RegisterDefaultCodecs()

	demo := NewDemo(640, 480)
	demo.RTCConfig = webrtc.RTCConfiguration{
		IceServers: []webrtc.RTCIceServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	demo.printer.Colored = *color

	if err := demo.Match(context.Background()); err != nil {
		fmt.Printf("Match error: %v\n", err)
	}

	select {}
}
