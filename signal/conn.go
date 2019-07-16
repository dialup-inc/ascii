package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type msgType string

const (
	msgTypeAnswer       msgType = "answer"
	msgTypeAnswerAck    msgType = "answerAck"
	msgTypeOffer        msgType = "offer"
	msgTypeRequestOffer msgType = "requestOffer"
)

type msg struct {
	Type    msgType     `json:"type"`
	Payload interface{} `json:"payload"`
}

type connID uint64

type conn struct {
	ID connID

	wsMu sync.Mutex
	ws   *websocket.Conn
}

func newConn(id connID, ws *websocket.Conn) *conn {
	return &conn{
		ID: id,
		ws: ws,
	}
}

func (c *conn) Close(code int, reason string) error {
	deadline := time.Now().Add(100 * time.Millisecond)
	msg := websocket.FormatCloseMessage(code, reason)

	c.ws.WriteControl(websocket.CloseMessage, msg, deadline)

	return c.ws.Close()
}

func (c *conn) RequestOffer() (offer interface{}, err error) {
	req := msg{
		Type: msgTypeRequestOffer,
	}
	resp := &msg{
		Type: msgTypeOffer,
	}

	if err := c.rpc(req, resp); err != nil {
		return nil, fmt.Errorf("request offer: %v", err)
	}

	return resp.Payload, nil
}

func (c *conn) SendOffer(offer interface{}) (answer interface{}, err error) {
	req := msg{
		Type:    msgTypeOffer,
		Payload: offer,
	}
	resp := &msg{
		Type: msgTypeAnswer,
	}

	if err := c.rpc(req, resp); err != nil {
		return nil, fmt.Errorf("send offer: %v", err)
	}

	return resp.Payload, nil
}

func (c *conn) SendAnswer(answer interface{}) error {
	req := msg{
		Type:    msgTypeAnswer,
		Payload: answer,
	}
	resp := &msg{
		Type: msgTypeAnswerAck,
	}

	if err := c.rpc(req, resp); err != nil {
		return fmt.Errorf("send answer: %v", err)
	}

	return nil
}

func (c *conn) rpc(req msg, resp *msg) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()

	if resp == nil {
		return fmt.Errorf("rpc: nil passed as response")
	}
	respType := resp.Type

	if err := c.ws.WriteJSON(req); err != nil {
		return fmt.Errorf("rpc write: %v", err)
	}

	if err := c.ws.ReadJSON(resp); err != nil {
		return fmt.Errorf("rpc read: %v", err)
	}

	if resp.Type != respType {
		return fmt.Errorf("rpc: expected %q, got %q", respType, resp.Type)
	}

	return nil
}
