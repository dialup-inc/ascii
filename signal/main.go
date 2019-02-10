package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

const roomName = "seanTest"

type signalMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

var roomsMu sync.Mutex
var rooms = map[string]chan signalMsg{}

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

func main() {
	var upgrader websocket.Upgrader
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[error]", err)
			return
		}
		defer conn.Close()

		roomsMu.Lock()
		comChan, shouldAnswer := rooms[roomName]
		if !shouldAnswer {
			comChan = make(chan signalMsg)
			rooms[roomName] = comChan

			defer func() {
				roomsMu.Lock()
				delete(rooms, roomName)
				roomsMu.Unlock()
			}()
		}
		roomsMu.Unlock()

		if shouldAnswer {
			generateAnswer(comChan, conn)
		} else {
			generateOffer(comChan, conn)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
