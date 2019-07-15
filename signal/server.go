package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type signalMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func NewServer() *Server {
	return &Server{
		rooms: make(map[string]chan signalMsg),
	}
}

type Server struct {
	roomsMu sync.Mutex
	rooms   map[string]chan signalMsg
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		s.HandleStatus(w, r)
	case "/ws":
		s.HandleWS(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]string{
		"app":     "ascii_roulette",
		"service": "signaler",
		"info":     "https://dialup.com/ascii",
		"source":     "https://github.com/dialupdotcom/ascii_roulette",
		"description": "This is the WebRTC signaling server for ASCII Roulette.",
		"contact": "webmaster@dialup.com",
	})
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	var upgrader websocket.Upgrader
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket.Upgrader error")
		return
	}
	defer conn.Close()

	roomName := r.URL.Query().Get("room")
	if roomName == "" {
		log.Warn().Msg("no room name provided")
		http.Error(w, "no room name provided", http.StatusBadRequest)
		return
	}

	s.roomsMu.Lock()
	comChan, shouldAnswer := s.rooms[roomName]
	if !shouldAnswer {
		comChan = make(chan signalMsg)
		s.rooms[roomName] = comChan

		defer func() {
			s.roomsMu.Lock()
			delete(s.rooms, roomName)
			s.roomsMu.Unlock()
		}()
	}
	s.roomsMu.Unlock()

	if shouldAnswer {
		generateAnswer(comChan, conn)
	} else {
		generateOffer(comChan, conn)
	}
}

func generateOffer(comChan chan signalMsg, conn *websocket.Conn) {
	if err := conn.WriteJSON(signalMsg{
		Type: "requestOffer",
	}); err != nil {
		log.Warn().Err(err).Msg("requestOffer write failed")
		return
	}

	offerMsg := &signalMsg{}
	if err := conn.ReadJSON(offerMsg); err != nil {
		log.Warn().Err(err).Msg("requestOffer read failed")
		return
	} else if offerMsg.Type != "offer" {
		msg := fmt.Sprint("expected offer from 'requestOffer' got:", offerMsg.Type)
		log.Warn().Msg(msg)
		return
	}
	comChan <- *offerMsg

	answerMsg := <-comChan
	if err := conn.WriteJSON(answerMsg); err != nil {
		log.Warn().Err(err).Msg("requestOffer reply failed")
		return
	}
}

func generateAnswer(comChan chan signalMsg, conn *websocket.Conn) {
	offer := <-comChan
	if err := conn.WriteJSON(offer); err != nil {
		log.Warn().Err(err).Msg("offer write failed")
		return
	}

	answerMsg := &signalMsg{}
	if err := conn.ReadJSON(answerMsg); err != nil {
		log.Warn().Err(err).Msg("answer read failed")
		return
	} else if answerMsg.Type != "answer" {
		msg := fmt.Sprint("expected answer from 'offer' got:", answerMsg.Type)
		log.Warn().Msg(msg)
		return
	}
	comChan <- *answerMsg
}
