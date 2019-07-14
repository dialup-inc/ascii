package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"time"

	"github.com/dialupdotcom/ascii_roulette/term"
	"github.com/dialupdotcom/ascii_roulette/ui"
	"github.com/dialupdotcom/ascii_roulette/videos"
	"github.com/pion/webrtc/v2"
)

func main() {
	var (
		camID       = flag.Int("cam-id", 0, "cam-id used by OpenCV's VideoCapture.open()")
		signalerURL = flag.String("signaler-url", "asciirtc-signaler.pion.ly:8080", "host and port of the signaler")
		room        = flag.String("room", "main", "Name of room to join ")
	)
	flag.Parse()

	app, err := New()
	if err != nil {
		log.Fatal(err)
	}

	app.RTCConfig = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	checkWinSize := func() {
		winSize, err := term.GetWinSize()
		if err != nil {
			return
		}
		app.dispatch(ui.ResizeEvent{winSize})
	}
	checkWinSize()
	go func() {
		for range time.Tick(500 * time.Millisecond) {
			checkWinSize()
		}
	}()

	var skipIntro func()
	var nextPartner func()

	sendMessage := func() {
		if app.conn == nil || !app.conn.IsConnected() {
			return
		}

		msg := app.state.Input
		if err := app.conn.SendMessage(msg); err != nil {
			app.dispatch(ui.ErrorEvent{"sending message failed"})
		} else {
			app.dispatch(ui.SentMessageEvent{msg})
		}
	}

	quitChan := make(chan struct{})

	if err := term.CaptureStdin(func(c rune) {
		switch c {
		case 3: // ctrl-c
			quitChan <- struct{}{}
		case 4: // ctrl-d
			if nextPartner != nil {
				nextPartner()
				nextPartner = nil
			}
		case 127: // backspace
		app.dispatch(ui.BackspaceEvent{})
		case '\n', '\r':
			sendMessage()
		case ' ':
			if skipIntro != nil {
				skipIntro()
				skipIntro = nil
			}
			app.dispatch(ui.KeypressEvent{c})
		default:
			app.dispatch(ui.KeypressEvent{c})
		}
	}); err != nil {
		log.Fatal(err)
	}

	go func() {
		var introCtx context.Context
		introCtx, skipIntro = context.WithCancel(context.Background())

		// Play Dialup intro
		app.dispatch(ui.SetPageEvent(ui.GlobePage))

		player, err := videos.NewPlayer(videos.Globe())
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			app.dispatch(ui.FrameEvent{img})
		}
		player.Play(introCtx)

		// Play Pion intro
		app.dispatch(ui.SetPageEvent(ui.PionPage))

		player, err = videos.NewPlayer(videos.Pion())
		if err != nil {
			log.Fatal(err)
		}
		player.OnFrame = func(img image.Image) {
			app.dispatch(ui.FrameEvent{img})
		}
		player.Play(introCtx)

		// Attempt to find match
		app.dispatch(ui.SetPageEvent(ui.ChatPage))

		if err := app.capture.Start(*camID, 5); err != nil {
			msg := fmt.Sprintf("camera error: %v", err)
			app.dispatch(ui.ErrorEvent{msg})
			// TODO: show in ui and retry
			return
		}

		var backoff float64
		for {
			var connCtx context.Context
			connCtx, nextPartner = context.WithCancel(context.Background())

			skipReason, err := app.Connect(connCtx, *signalerURL, *room)
			if err != nil {
				app.dispatch(ui.ErrorEvent{err.Error()})

				sec := math.Pow(2, backoff) - 1
				time.Sleep(time.Duration(sec) * time.Second)
				if backoff < 4 {
					backoff++
				}
				continue
			}
			app.dispatch(ui.InfoEvent{skipReason})
			app.dispatch(ui.FrameEvent{nil})

			time.Sleep(100 * time.Millisecond)
		}
	}()

	<-quitChan
	if app.conn != nil && app.conn.IsConnected() {
		app.conn.SendBye()
	}
	app.Stop()
}
