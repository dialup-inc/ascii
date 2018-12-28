package main

import (
	"image"
	"sync"
	"time"

	"github.com/buger/goterm"
	"github.com/qeesung/image2ascii/convert"
)

type Printer struct {
	imgMu sync.Mutex
	img   image.Image

	Colored bool

	conv *convert.ImageConverter

	startOnce sync.Once

	stopCh chan struct{}
}

func NewPrinter() *Printer {
	return &Printer{
		stopCh: make(chan struct{}),
		conv:   convert.NewImageConverter(),
	}
}

func (p *Printer) SetImage(img image.Image) {
	p.imgMu.Lock()
	p.img = img
	p.imgMu.Unlock()
}

func (p *Printer) draw(t time.Duration) {
	if t == 0 {
		goterm.Clear()
	}

	opts := convert.DefaultOptions
	opts.FixedWidth = goterm.Width()
	opts.FixedHeight = goterm.Height() - 1
	opts.Colored = p.Colored

	p.imgMu.Lock()
	if p.img == nil {
		p.imgMu.Unlock()
		return
	}
	ascii := p.conv.Image2ASCIIString(p.img, &opts)
	p.imgMu.Unlock()

	goterm.MoveCursor(1, 1)
	goterm.Print(ascii)
	goterm.Flush()
}

func (p *Printer) loop() {
	p.draw(0)

	start := time.Now()
	tick := time.NewTicker(time.Second / 5)
	for {
		select {
		case <-tick.C:
			p.draw(time.Since(start))
		case <-p.stopCh:
			break
		}
	}
}

func (p *Printer) Start() {
	p.startOnce.Do(func() {
		go p.loop()
	})
}

func (p *Printer) Stop() {
	select {
	case <-p.stopCh:
		return
	default:
		close(p.stopCh)
	}
}
