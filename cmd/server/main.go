package main

import (
	"log"
	"net/http"

	"github.com/dialupdotcom/ascii_roulette/signal"
)

func main() {
	srv := signal.NewServer()
	http.Handle("/ws", srv)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
