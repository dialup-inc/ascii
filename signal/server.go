package signal

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v2"
)

type signalMsg struct {
	Type    string                    `json:"type"`
	Payload webrtc.SessionDescription `json:"payload"`
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
	var upgrader websocket.Upgrader
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[error]", err)
		return
	}
	defer conn.Close()

	roomName := r.URL.Query().Get("room")
	if roomName == "" {
		log.Println("No room name provided", err)
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
		log.Println("write:", err)
		return
	}

	offerMsg := &signalMsg{}
	if err := conn.ReadJSON(offerMsg); err != nil {
		log.Println("read:", err)
		return
	} else if offerMsg.Type != "offer" {
		log.Println("expected offer from 'requestOffer' got:", offerMsg.Type)
		return
	}
	comChan <- *offerMsg

	answerMsg := <-comChan
	if err := conn.WriteJSON(answerMsg); err != nil {
		log.Println("write:", err)
		return
	}
}

func generateAnswer(comChan chan signalMsg, conn *websocket.Conn) {
	offer := <-comChan
	if err := conn.WriteJSON(offer); err != nil {
		log.Println("write:", err)
		return
	}

	answerMsg := &signalMsg{}
	if err := conn.ReadJSON(answerMsg); err != nil {
		log.Println("read:", err)
		return
	} else if answerMsg.Type != "answer" {
		log.Println("expected answer from 'offer' got:", answerMsg.Type)
		return
	}
	comChan <- *answerMsg
}
