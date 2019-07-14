package main

import (
	"context"
	"flag"
	"log"

	roulette "github.com/dialupdotcom/ascii_roulette"
)

func main() {
	var (
		signalerURL = flag.String("signaler-url", "asciirtc-signaler.pion.ly:8080", "host and port of the signaler")
		room        = flag.String("room", "main", "Name of room to join ")
	)
	flag.Parse()

	ctx := context.Background()

	app, err := roulette.New(*signalerURL, *room)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
