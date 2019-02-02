package main

import "github.com/pions/rtp"

type JitterBuffer struct {
	buffer [65536]*rtp.Packet

	lastSeq uint16

	// The number of packets before assuming a packet is lost
	size uint16

	cursor    uint16
	cursorSet bool
}

func (j *JitterBuffer) Push(p *rtp.Packet) {
	j.buffer[p.SequenceNumber] = p

	// TODO: handle wraparound, out of order packets
	j.lastSeq = p.SequenceNumber

	if !j.cursorSet {
		j.cursor = p.SequenceNumber
		j.cursorSet = true
	}
}

func (j *JitterBuffer) Pop() *rtp.Packet {
	if !j.cursorSet {
		return nil
	}

	p := j.buffer[j.cursor]
	if p == nil {
		size := j.size
		if size == 0 {
			size = 20
		}

		// It's been too long
		if j.lastSeq-j.cursor > size {
			j.seekNextPacket()
			return nil // TODO: return next packet
		}

		return nil
	}

	j.cursor++
	j.buffer[j.cursor] = nil

	return p
}

func (j *JitterBuffer) seekNextPacket() {
	for ; j.cursor < j.lastSeq; j.cursor++ {
		if j.buffer[j.cursor] != nil {
			break
		}
	}
}
