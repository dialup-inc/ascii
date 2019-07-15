package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func main() {
	var (
		port = flag.Int("port", 8080, "http port")
	)
	flag.Parse()

	srv := NewServer()

	log.Info().Int("port", *port).Msg("listening")
	if err := http.ListenAndServe(fmt.Sprint(":", *port), srv); err != nil {
		log.Fatal().Err(err).Msg("http server failed")
	}
}
