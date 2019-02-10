package main

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/pions/webrtc"
)

type signalMsg struct {
	Type    string                       `json:"type"`
	Payload webrtc.RTCSessionDescription `json:"payload"`
}

func match(ctx context.Context, wsURL string, conn *webrtc.RTCPeerConnection) error {
	signalConn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer signalConn.Close()

	msg := &signalMsg{}
	for {
		if err := signalConn.ReadJSON(msg); err != nil {
			return err
		}

		switch msg.Type {
		case "requestOffer":
			offer, err := conn.CreateOffer(nil)
			if err != nil {
				return err
			}
			if err := signalConn.WriteJSON(signalMsg{
				Type:    "offer",
				Payload: offer,
			}); err != nil {
				return err
			}

		case "offer":
			if err := conn.SetRemoteDescription(msg.Payload); err != nil {
				return err
			}
			answer, err := conn.CreateAnswer(nil)
			if err != nil {
				return err
			}
			if err := signalConn.WriteJSON(signalMsg{
				Type:    "answer",
				Payload: answer,
			}); err != nil {
				return err
			}

			return nil
		case "answer":
			if err := conn.SetRemoteDescription(msg.Payload); err != nil {
				return err
			}

			return nil
		default:
			return fmt.Errorf("unknown signaling message %v", msg.Type)
		}

	}
}
