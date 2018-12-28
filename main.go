package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/maxhawkins/asciirtc/capture"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/ice"
	"github.com/pions/webrtc/pkg/media"
	"github.com/pions/webrtc/pkg/media/samplebuilder"
	"github.com/pions/webrtc/pkg/rtcp"
	"github.com/pions/webrtc/pkg/rtp"
	"github.com/pions/webrtc/pkg/rtp/codecs"
)

var saver = flag.Bool("saver", false, "")

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
		if !*saver {
			for {
				time.Sleep(1 * time.Second)
			}
		}

		time.Sleep(3 * time.Second)
		fmt.Println("Sending now...")

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

			// if *saver {
			// 	// fmt.Println("SAVE", n)
			// 	ioutil.WriteFile(fmt.Sprintf("frame-%d.vp8", i), buf[:n], 0777)
			// }

			track.Samples <- samp

			// if i > 2 {
			// 	time.Sleep(100 * time.Second)
			// 	os.Exit(0)
			// }

			// select {
			// case track.Samples <- samp:
			// default:
			// }
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

	packetQ := make(chan *rtp.Packet)
	go func() {
		for p := range track.Packets {
			go func(pkt *rtp.Packet) {
				packetQ <- pkt
			}(p)
		}
	}()

	i := 0
	stride := d.width * 4
	frame := make([]byte, stride*d.height)
	for j := 0; ; j++ {
		select {
		case <-ctx.Done():
			return
		case pkt := <-packetQ:

			if *saver {
				continue
			}

			// if !*saver {
			// 	fmt.Println("in", i)
			// 	d, _ := pkt.Marshal()
			// 	ioutil.WriteFile(fmt.Sprintf("in-%d.pkt", i), d, 0777)
			// }

			builder.Push(pkt)
			for s := builder.Pop(); s != nil; s = builder.Pop() {

				i++
				if !*saver {
					fmt.Println("RECV", len(s.Data))
					ioutil.WriteFile(fmt.Sprintf("frame-%d.pkt", i), s.Data, 0777)
				}

				if err := capture.DecodeFrame(frame, s.Data); err != nil {
					fmt.Println(err)
					continue
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

				f, err := os.Create(fmt.Sprintf("%02d.png", i))
				if err != nil {
					log.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, img); err != nil {
					log.Fatal(err)
				}
			}
		}
	}
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
