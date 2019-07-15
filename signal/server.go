package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func NewServer() *Server {
	s := &Server{
		lobby: make(map[connID]*conn),
	}
	go s.doMatching()
	return s
}

type Server struct {
	lobbyMu sync.Mutex
	lobby   map[connID]*conn

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
		"info":        "https://dialup.com/ascii",
		"source":      "https://github.com/dialupdotcom/ascii_roulette",
		"description": "This is the WebRTC signaling server for ASCII Roulette.",
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

	nextID := atomic.AddUint64((*uint64)(&s.nextID), 1)
	id := connID(nextID)

	c := &conn{
		ID: id,
		ws: ws,
	}

	s.lobbyMu.Lock()
	s.lobby[id] = c
	s.lobbyMu.Unlock()

	log.Info().Uint64("id", nextID).Msg("new conn")
}

func (s *Server) connComplete(c *conn) {
	c.Close(websocket.CloseNormalClosure, "")

	s.lobbyMu.Lock()
	delete(s.lobby, c.ID)
	s.lobbyMu.Unlock()

	log.Info().
		Uint64("id", uint64(c.ID)).
		Str("state", "complete").
		Msg("conn closed")
}

func (s *Server) connErr(c *conn, err error) {
	c.Close(websocket.CloseInternalServerErr, err.Error())

	s.lobbyMu.Lock()
	delete(s.lobby, c.ID)
	s.lobbyMu.Unlock()

	log.Info().
		Uint64("id", uint64(c.ID)).
		Err(err).
		Str("state", "failed").
		Msg("conn closed")
}

func (s *Server) match(a, b *conn) {
	offer, err := a.RequestOffer()
	if err != nil {
		s.connErr(a, err)
		return
	}
	answer, err := b.SendOffer(offer)
	if err != nil {
		s.connErr(a, err)
		s.connErr(b, err)
		return
	}
	if err := a.SendAnswer(answer); err != nil {
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
		s.lobbyMu.Lock()
		var lobby []*conn
		for _, c := range s.lobby {
			lobby = append(lobby, c)
		}
		s.lobbyMu.Unlock()

		var partner *conn
		for _, i := range rand.Perm(len(lobby)) {
			c := lobby[i]

			if partner == nil {
				partner = c
				continue
			}

			go s.match(c, partner)
			partner = nil
		}
	}
}
