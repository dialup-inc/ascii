package main

import (
	"encoding/binary"
	"errors"
	"io"
)

type IVFReader struct {
	reader io.ReadSeeker
	Header IVFHeader
}

type IVFHeader struct {
	Signaure   [4]byte
	Version    uint16
	Size       uint16
	Codec      [4]byte
	Width      uint16
	Height     uint16
	FrameRate  uint32
	FrameScale uint32
	FrameCount uint32
	_          uint32
}

type IVFFrameHeader struct {
	Size uint32
	PTS  uint64
}

func NewIVFReader(r io.ReadSeeker) (*IVFReader, error) {
	var hdr IVFHeader
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		return nil, err
	}

	if string(hdr.Signaure[:]) != "DKIF" {
		return nil, errors.New("not a valid IVF file")
	}
	if hdr.Version != 0 {
		return nil, errors.New("unsupported IVF version")
	}

	return &IVFReader{
		reader: r,
		Header: hdr,
	}, nil
}

func (i *IVFReader) Codec() string {
	return string(i.Header.Codec[:])
}

func (i *IVFReader) ReadFrame() (data []byte, pts uint64, err error) {
	var hdr IVFFrameHeader
	if err := binary.Read(i.reader, binary.LittleEndian, &hdr); err != nil {
		return nil, 0, err
	}

	frame := make([]byte, hdr.Size)
	if _, err := io.ReadFull(i.reader, frame); err != nil {
		return nil, 0, err
	}

	return frame, hdr.PTS, nil
}

func (i *IVFReader) Rewind() error {
	_, err := i.reader.Seek(32, io.SeekStart)
	return err
}
