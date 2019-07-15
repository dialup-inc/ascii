package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func NewServer() *Server {
	s := &Server{
		lobby: NewLobby(),
	}
	go s.doMatching()
	return s
}

type Server struct {
	lobby *Lobby

	nextID connID
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
		"app":         "ascii_roulette",
		"service":     "signaler",
		"description": "This is the WebRTC signaling server for ASCII Roulette.",
		"info":        "https://dialup.com/ascii",
		"source":      "https://github.com/dialup-inc/ascii",
		"contact":     "webmaster@dialup.com",
	})
}

func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	var upgrader websocket.Upgrader
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("websocket error")
		return
	}

	id := atomic.AddUint64((*uint64)(&s.nextID), 1)

	s.lobby.Add(&conn{
		ID: connID(id),
		ws: ws,
	})

	log.Info().
		Uint64("id", id).
		Msg("new conn")
}

func (s *Server) connComplete(c *conn) {
	log.Info().
		Uint64("id", uint64(c.ID)).
		Str("state", "complete").
		Msg("conn closed")

	c.Close(websocket.CloseNormalClosure, "")
}

func (s *Server) connErr(c *conn, err error) {
	log.Info().
		Uint64("id", uint64(c.ID)).
		Err(err).
		Str("state", "failed").
		Msg("conn closed")

	c.Close(websocket.CloseInternalServerErr, err.Error())
}

func (s *Server) handshake(a, b *conn) error {
	offer, err := a.RequestOffer()
	if err != nil {
		return err
	}
	answer, err := b.SendOffer(offer)
	if err != nil {
		return err
	}
	if err := a.SendAnswer(answer); err != nil {
		return err
	}
	return nil
}

func (s *Server) match(a, b *conn) {
	if err := s.handshake(a, b); err != nil {
		s.connErr(a, err)
		s.connErr(b, err)
		return
	}

	log.Info().
		Uint64("a", uint64(a.ID)).
		Uint64("b", uint64(b.ID)).
		Msg("matched conns")

	s.connComplete(a)
	s.connComplete(b)
}

// runs in own goroutine
func (s *Server) doMatching() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		candidates := s.lobby.Pop()

		var partner *conn
		for _, i := range rand.Perm(len(candidates)) {
			c := candidates[i]

			if partner == nil {
				partner = c
				continue
			}

			go s.match(c, partner)
			partner = nil
		}

		if partner != nil {
			s.lobby.Add(partner)
		}
	}
}
