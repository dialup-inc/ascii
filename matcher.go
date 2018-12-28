package main

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/pions/webrtc"
)

type SignalMsg struct {
	Type    string                       `json:"type"`
	Payload webrtc.RTCSessionDescription `json:"payload"`
}

func Match(ctx context.Context, wsURL string, conn *webrtc.RTCPeerConnection) error {
	signalConn, _, err := websocket.DefaultDialer.DialContext(ctx, "ws://localhost:8080/ws", nil)
	if err != nil {
		return err
	}
	defer signalConn.Close()

	offer, err := conn.CreateOffer(nil)
	if err != nil {
		return err
	}

	if err := signalConn.WriteJSON(SignalMsg{
		Type:    "offer",
		Payload: offer,
	}); err != nil {
		return err
	}

	msg := &SignalMsg{}
	if err := signalConn.ReadJSON(msg); err != nil {
		return err
	}

	switch msg.Type {
	case "offer":
		if err := conn.SetRemoteDescription(msg.Payload); err != nil {
			return err
		}
		answer, err := conn.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err := conn.SetLocalDescription(answer); err != nil {
			return err
		}
		if err := signalConn.WriteJSON(SignalMsg{
			Type:    "answer",
			Payload: answer,
		}); err != nil {
			return err
		}
	case "answer":
		if err := conn.SetLocalDescription(offer); err != nil {
			return err
		}
		if err := conn.SetRemoteDescription(msg.Payload); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown signaling message %v", msg.Type)
	}

	return nil
}
