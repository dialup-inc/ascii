package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/pions/webrtc/pkg/media/samplebuilder"
	"github.com/pions/webrtc/pkg/rtp"
	"github.com/pions/webrtc/pkg/rtp/codecs"
)

func main() {
	builder := samplebuilder.New(256, &codecs.VP8Packet{})

	data, err := ioutil.ReadFile(fmt.Sprintf("packets/recv-%d.pkt", 16714))
	if err != nil {
		log.Fatal(err)
	}

	pkt := &rtp.Packet{}
	if err := pkt.Unmarshal(data); err != nil {
		log.Fatal(err)
	}
	pkt.SequenceNumber = 16711
	pkt.Timestamp -= 2
	builder.Push(pkt)

	pkt = &rtp.Packet{}
	if err := pkt.Unmarshal(data); err != nil {
		log.Fatal(err)
	}
	pkt.SequenceNumber = 16712
	pkt.Timestamp -= 2
	builder.Push(pkt)

	pkt = &rtp.Packet{}
	if err := pkt.Unmarshal(data); err != nil {
		log.Fatal(err)
	}
	pkt.SequenceNumber = 16713
	pkt.Timestamp -= 1
	builder.Push(pkt)

	for s := builder.Pop(); s != nil; s = builder.Pop() {
		fmt.Println("BEFORE")
	}

	for i := 16714; i <= 16762; i++ {
		data, err = ioutil.ReadFile(fmt.Sprintf("packets/recv-%d.pkt", i))
		if err != nil {
			log.Fatal(err)
		}

		pkt = &rtp.Packet{}
		if err := pkt.Unmarshal(data); err != nil {
			log.Fatal(err)
		}

		// fmt.Println(pkt)

		builder.Push(pkt)
	}

	for s := builder.Pop(); s != nil; s = builder.Pop() {
		fmt.Println("POP")
	}

	pkt = &rtp.Packet{}
	if err := pkt.Unmarshal(data); err != nil {
		log.Fatal(err)
	}
	pkt.SequenceNumber = 16763
	pkt.Timestamp = 1004
	builder.Push(pkt)

	for s := builder.Pop(); s != nil; s = builder.Pop() {
		fmt.Println("WRITE")
		ioutil.WriteFile("sample.pkt", s.Data, 0777)
	}
}
