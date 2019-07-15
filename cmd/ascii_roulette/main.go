package main

import (
	"context"
	"flag"
	"log"

	"github.com/dialup-inc/ascii"
)

func main() {
	var (
		signalerURL = flag.String("signaler-url", "wss://roulette.dialup.com/ws", "host and port of the signaler")
	)
	flag.Parse()

	ctx := context.Background()

	app, err := ascii.New(*signalerURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
