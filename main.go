package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"sync"
	"time"
	"log"

	"github.com/pions/asciirtc/term"
	"github.com/pions/asciirtc/vpx"
	"github.com/pions/rtcp"
	"github.com/pions/rtp/codecs"
	"github.com/pions/webrtc"
	"github.com/pions/webrtc/pkg/media/samplebuilder"
)

type demo struct {
	RTCConfig webrtc.Configuration

	width  int
	height int

	renderer *term.Renderer

	connMu sync.Mutex
	conn   *webrtc.PeerConnection

	capture *Capture
}

// mode for frames width per timestamp from a 30 second capture
const rtpAverageFrameWidth = 7

func (d *demo) Match(ctx context.Context, camID int, signalerURL, room string) error {
	ctx, cancel := context.WithCancel(ctx)

	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	conn, err := api.NewPeerConnection(d.RTCConfig)
	if err != nil {
		cancel()
		return err
	}
	d.conn = conn


	go func() {
		time.Sleep(5 * time.Second)
		if err := d.capture.Start(camID, 5); err != nil {
			fmt.Println(err)
		}
	}()

	conn.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		fmt.Println("ICEConnectionState", s)
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

	var once sync.Once
	conn.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver) {
		once.Do(func() {
			d.handleTrack(ctx, track)
		})
	})

	track, err := conn.NewTrack(webrtc.DefaultPayloadTypeVP8, 1234, "video", "asciirtc")
	if err != nil {
		cancel()
		return err
	}
	if _, err := conn.AddTrack(track); err != nil {
		cancel()
		return err
	}
	d.capture.SetTrack(track)

	if err := match(ctx, fmt.Sprintf("ws://%s/ws?room=%s", signalerURL, room), conn); err != nil {
		cancel()
		return err
	}

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
		capture: cap,
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
	demo.RTCConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	if err := demo.Match(context.Background(), *camID, *signalerURL, *room); err != nil {
		fmt.Printf("Match error: %v\n", err)
		return
	}

	select {}
}
