package videos

import (
	"context"
	"fmt"
	"image"
	"io"
	"sync"
	"time"

	"github.com/dialup-inc/ascii/vpx"
)

type Player struct {
	reader  *IVFReader
	decoder *vpx.Decoder

	OnFrame func(image.Image)

	ctxMu sync.Mutex
	ctx   context.Context
}

func (p *Player) Play(ctx context.Context) error {
	p.ctxMu.Lock()
	if p.ctx != nil {
		p.ctxMu.Unlock()
		return nil
	}
	p.ctx = ctx
	p.ctxMu.Unlock()

	defer func() {
		p.ctxMu.Lock()
		p.ctx = ctx
		p.ctxMu.Unlock()
	}()

	if err := p.reader.Rewind(); err != nil {
		return err
	}

	lastFrame := time.Now()

	hdr := p.reader.Header

	period := time.Duration(float64(hdr.FrameScale) / float64(hdr.FrameRate) * float64(time.Second))

	for {
		frame, _, err := p.reader.ReadFrame()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		img, err := p.decoder.Decode(frame)
		if err != nil {
			return err
		}

		waitTime := period - time.Since(lastFrame)
		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return ctx.Err()
		}

		p.OnFrame(img)

		lastFrame = time.Now()
	}
}

func NewPlayer(r io.ReadSeeker) (*Player, error) {
	ivf, err := NewIVFReader(r)
	if err != nil {
		return nil, err
	}
	if c := ivf.Codec(); c != "VP80" {
		return nil, fmt.Errorf("unknown codec %q", c)
	}

	decoder, err := vpx.NewDecoder(int(ivf.Header.Width), int(ivf.Header.Height))
	if err != nil {
		return nil, err
	}

	player := &Player{
		decoder: decoder,
		reader:  ivf,
		OnFrame: func(image.Image) {},
	}

	return player, nil
}
