package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v2"
	"github.com/pion/webrtc/v2/pkg/media/samplebuilder"
)

// mode for frames width per timestamp from a 30 second capture
const rtpAverageFrameWidth = 7

type DCMessage struct {
	Event   string
	Payload []byte
}

func NewConn(config webrtc.Configuration) (*Conn, error) {
	conn := &Conn{
		OnPLI:                      func() {},
		OnFrame:                    func([]byte) {},
		OnMessage:                  func(string) {},
		OnBye:                      func() {},
		OnICEConnectionStateChange: func(webrtc.ICEConnectionState) {},
	}

	m := webrtc.MediaEngine{}
	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	pc, err := api.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}
	conn.pc = pc

	pc.OnICEConnectionStateChange(conn.onICEConnectionStateChange)
	pc.OnTrack(conn.onTrack)

	if _, err = pc.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		return nil, err
	}

	track, err := pc.NewTrack(webrtc.DefaultPayloadTypeVP8, rand.Uint32(), "video", "roulette")
	if err != nil {
		return nil, err
	}
	if _, err := pc.AddTrack(track); err != nil {
		return nil, err
	}
	conn.SendTrack = track

	dc, err := pc.CreateDataChannel("chat", nil)
	if err != nil {
		return nil, err
	}
	conn.dc = dc

	dc.OnMessage(conn.onMessage)

	return conn, nil
}

type Conn struct {
	SendTrack                  *webrtc.Track
	OnPLI                      func()
	OnFrame                    func([]byte)
	OnMessage                  func(string)
	OnICEConnectionStateChange func(webrtc.ICEConnectionState)
	OnBye                      func()

	pc        *webrtc.PeerConnection
	recvTrack *webrtc.Track
	ssrc      uint32

	dc *webrtc.DataChannel

	lastPLI time.Time
}

func (c *Conn) readRTCP(recv *webrtc.RTPReceiver) {
	for {
		rtcps, err := recv.ReadRTCP()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
			// fmt.Println(err)
		}

		for _, pkt := range rtcps {
			switch p := pkt.(type) {
			case *rtcp.PictureLossIndication:
				c.OnPLI()
			case *rtcp.Goodbye:
				for _, ssrc := range p.Sources {
					if ssrc == recv.Track().SSRC() {
						c.OnBye()
						break
					}
				}
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
			// fmt.Println(err)
			continue
		}

		builder.Push(pkt)

		for s := builder.Pop(); s != nil; s = builder.Pop() {
			c.OnFrame(s.Data)
		}
	}
}

func (c *Conn) onMessage(msg webrtc.DataChannelMessage) {
	var dcm DCMessage
	if err := json.Unmarshal(msg.Data, &dcm); err != nil {
		// TODO
	}
	if dcm.Event == "chat" {
		c.OnMessage(string(dcm.Payload))
	}
}

func (c *Conn) onICEConnectionStateChange(s webrtc.ICEConnectionState) {
	c.OnICEConnectionStateChange(s)
}

func (c *Conn) SendMessage(m string) error {
	data, err := json.Marshal(DCMessage{
		Event:   "chat",
		Payload: []byte(m),
	})
	if err != nil {
		return err
	}
	return c.dc.Send(data)
}

func (c *Conn) SendPLI() error {
	if time.Since(c.lastPLI) < 500*time.Millisecond {
		return nil
	}
	if c.recvTrack == nil {
		return nil
	}

	pli := &rtcp.PictureLossIndication{MediaSSRC: c.SendTrack.SSRC()}
	if err := c.pc.WriteRTCP([]rtcp.Packet{pli}); err != nil {
		return err
	}

	c.lastPLI = time.Now()
	return nil
}

func (c *Conn) SendBye() error {
	if c.SendTrack == nil {
		return nil
	}

	bye := &rtcp.Goodbye{Sources: []uint32{c.SendTrack.SSRC()}}
	if err := c.pc.WriteRTCP([]rtcp.Packet{bye}); err != nil {
		return err
	}

	return nil
}

func (c *Conn) IsConnected() bool {
	switch c.pc.ICEConnectionState() {
	case webrtc.ICEConnectionStateCompleted, webrtc.ICEConnectionStateConnected:
		return true
	default:
		return false
	}
}

func (c *Conn) onTrack(track *webrtc.Track, recv *webrtc.RTPReceiver) {
	if !atomic.CompareAndSwapUint32(&c.ssrc, 0, track.SSRC()) {
		return
	}
	c.recvTrack = track

	go c.readRTCP(recv)
	c.readRTP(track)
}
