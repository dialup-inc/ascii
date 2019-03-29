package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	"github.com/pions/asciirtc/camera"
	"github.com/pions/asciirtc/vpx"
	"github.com/pions/rtcp"
	"github.com/pions/rtp/codecs"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/media"
	"github.com/pions/webrtc/pkg/media/samplebuilder"
)

type demo struct {
	RTCConfig webrtc.Configuration

	width  int
	height int

	printer *Printer

	connMu sync.Mutex
	conn   *webrtc.PeerConnection

	cam *camera.Camera
}

// mode for frames width per timestamp from a 30 second capture
const rtpAverageFrameWidth = 7

func (d *demo) newConn() (*webrtc.PeerConnection, error) {
	d.connMu.Lock()
	defer d.connMu.Unlock()

	if d.conn != nil {
		return nil, errors.New("another peer connection is connected")
	}

	conn, err := webrtc.NewPeerConnection(d.RTCConfig)
	if err != nil {
		return nil, err
	}
	d.conn = conn

	return conn, nil
}

func (d *demo) Match(ctx context.Context, camID int, signalerURL, room string) error {
	ctx, cancel := context.WithCancel(ctx)

	conn, err := d.newConn()
	if err != nil {
		cancel()
		return err
	}

	var track *webrtc.Track

	var frameMu sync.Mutex
	var pts int
	vpxBuf := make([]byte, 5*1024*1024)

	enc, err := vpx.NewEncoder(d.width, d.height)
	if err != nil {
		cancel()
		return err
	}

	cb := func(frame []byte) {
		frameMu.Lock()
		defer frameMu.Unlock()

		n, err := enc.Encode(vpxBuf, frame, pts, true)
		if err != nil {
			log.Fatal("encode: ", err)
		}
		pts++

		data := vpxBuf[:n]

		select {
		case <-ctx.Done():
			return
		default:
		}

		samp := media.Sample{Data: data, Samples: 1}

		if track == nil {
			return
		}

		if err := track.WriteSample(samp); err != nil {
			fmt.Println(err)
			return
		}
	}

	cam, err := camera.New(cb)
	if err != nil {
		cancel()
		return err
	}
	d.cam = cam

	if err := cam.Start(camID, d.width, d.height); err != nil {
		cancel()
		return err
	}

	conn.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		if s == webrtc.ICEConnectionStateClosed || s == webrtc.ICEConnectionStateFailed {
			d.connMu.Lock()
			if conn == d.conn {
				d.conn = nil
			}
			d.connMu.Unlock()

			cancel()

			// if err := capture.Stop(); err != nil {
			// 	fmt.Println(err)
			// }
		}
	})

	track, err = conn.NewTrack(webrtc.DefaultPayloadTypeVP8, 1234, "video", "asciirtc")
	if err != nil {
		cancel()
		return err
	}
	if _, err := conn.AddTrack(track); err != nil {
		cancel()
		return err
	}

	var once sync.Once
	conn.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver) {
		once.Do(func() {
			d.handleTrack(ctx, track)
		})
	})

	if err := match(ctx, fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room), conn); err != nil {
		cancel()
		return err
	}
	fmt.Println("CONNECTED")

	return err
}

func (d *demo) handleTrack(ctx context.Context, track *webrtc.Track) {
	// Send PLIs every once in a while
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				pli := &rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}
				if err := d.conn.SendRTCP(pli); err != nil {
					fmt.Println(err)
				}
			}
		}
	}()

	dec, err := vpx.NewDecoder()
	if err != nil {
		fmt.Println(err)
		return
	}

	// todo: less alloc
	frameBuf := make([]byte, d.width*d.height*4)

	builder := samplebuilder.New(rtpAverageFrameWidth*5, &codecs.VP8Packet{})
	for j := 0; ; j++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pkt, err := track.ReadRTP()
		if err != nil {
			fmt.Println(err)
			continue
		}

		builder.Push(pkt)

		for s := builder.Pop(); s != nil; s = builder.Pop() {
			if err := d.decode(dec, frameBuf, s.Data); err != nil {
				fmt.Println(err)
			}
		}
	}
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
		room        = flag.String("room", "", "Name of room to join ")
	)
	flag.Parse()

	if *room == "" {
		fmt.Println("No -room has been provided")
		return
	}

	demo := newDemo(640, 480)
	demo.RTCConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}
	demo.printer.Colored = *color

	if err := demo.Match(context.Background(), *camID, *signalerURL, *room); err != nil {
		fmt.Printf("Match error: %v\n", err)
		return
	}

	select {}
}
