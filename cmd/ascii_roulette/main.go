package main

import (
	"context"
	"flag"
	"log"

	roulette "github.com/dialupdotcom/ascii_roulette"
)

func main() {
	var (
		signalerURL = flag.String("signaler-url", "wss://roulette.dialup.com/ws", "host and port of the signaler")
	)
	flag.Parse()

	ctx := context.Background()

	app, err := roulette.New(*signalerURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
