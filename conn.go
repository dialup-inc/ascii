package main

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/samplebuilder"
)

// mode for frames width per timestamp from a 30 second capture
const rtpAverageFrameWidth = 7

func NewConn(config webrtc.Configuration) (*Conn, error) {
	conn := &Conn{
		OnPLI:   func() {},
		OnFrame: func([]byte) {},
	}

	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}
	conn.pc = pc

	if _, err = pc.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	track, err := pc.NewTrack(webrtc.DefaultPayloadTypeVP8, 1234, "video", "roulette")
	if err != nil {
		return nil, err
	}
	if _, err := pc.AddTrack(track); err != nil {
		return nil, err
	}
	conn.SendTrack = track

	pc.OnICEConnectionStateChange(conn.onICEConnectionStateChange)
	pc.OnTrack(conn.onTrack)

	return conn, nil
}

type Conn struct {
	SendTrack *webrtc.Track
	OnPLI     func()
	OnFrame   func([]byte)

	pc        *webrtc.PeerConnection
	recvTrack *webrtc.Track
	ssrc      uint32

	lastPLI time.Time
}

func (c *Conn) readRTCP(recv *webrtc.RTPReceiver) {
	for {
		rtcps, err := recv.ReadRTCP()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
		}

		for _, pkt := range rtcps {
			switch pkt.(type) {
			case *rtcp.PictureLossIndication:
				c.OnPLI()
			}
		}
	}
}

func (c *Conn) readRTP(track *webrtc.Track) {
	builder := samplebuilder.New(rtpAverageFrameWidth*5, &codecs.VP8Packet{})

	for {
		pkt, err := track.ReadRTP()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			continue
		}

		builder.Push(pkt)

		for s := builder.Pop(); s != nil; s = builder.Pop() {
			c.OnFrame(s.Data)
		}
	}
}

func (c *Conn) SendPLI() error {
	if time.Since(c.lastPLI) < 500*time.Millisecond {
		return nil
	}
	if c.recvTrack == nil {
		return nil
	}

	pli := &rtcp.PictureLossIndication{MediaSSRC: c.recvTrack.SSRC()}
	if err := c.pc.WriteRTCP([]rtcp.Packet{pli}); err != nil {
		return err
	}

	c.lastPLI = time.Now()
	return nil
}

func (c *Conn) onTrack(track *webrtc.Track, recv *webrtc.RTPReceiver) {
	if !atomic.CompareAndSwapUint32(&c.ssrc, 0, track.SSRC()) {
		return
	}
	c.recvTrack = track

	go c.readRTCP(recv)
	c.readRTP(track)
}

func (c *Conn) onICEConnectionStateChange(s webrtc.ICEConnectionState) {
	fmt.Println("ICEConnectionState", s)
}