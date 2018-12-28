package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type SignalMsg struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func main() {
	var upgrader websocket.Upgrader

	var busMu sync.Mutex
	bus := make(map[int]chan *SignalMsg)

	var lobbyMu sync.Mutex
	lobby := -1
	lobbyCh := make(chan int)

	match := func(id int) (match int, first bool) {
		lobbyMu.Lock()
		if lobby < 0 {
			lobby = id
			lobbyMu.Unlock()
			match = <-lobbyCh
			first = false
		} else {
			match = lobby
			lobby = -1
			lobbyMu.Unlock()
			lobbyCh <- id
			first = true
		}

		return match, first
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[error]", err)
			return
		}
		defer conn.Close()

		busMu.Lock()
		recv := make(chan *SignalMsg)
		id := len(bus)
		bus[id] = recv
		busMu.Unlock()

		partner, first := match(id)

		busMu.Lock()
		send := bus[partner]
		busMu.Unlock()

		go func() {
			for r := range recv {
				if err := conn.WriteJSON(r); err != nil {
					log.Println("write:", err)
					return
				}
			}
		}()

		msg := &SignalMsg{}
		for {
			if err := conn.ReadJSON(msg); err != nil {
				log.Println("read:", err)
				return
			}

			switch msg.Type {
			case "offer":
				if first {
					send <- msg
				}

			case "answer":
				send <- msg

			case "exit":
				break

			default:
				fmt.Println("unknown type", msg.Type)
				break
			}
		}

		close(send)
		busMu.Lock()
		delete(bus, partner)
		busMu.Unlock()
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
