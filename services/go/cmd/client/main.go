package main

import (
	"context"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/enet"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	host, err := enet.NewConnectHost("localhost", 28785)
	if err != nil {
		log.Error().Err(err)
		return
	}

	ctx := context.Background()
	events := host.Service()
outer:
	for {
		select {
		case <-ctx.Done():
			break outer
		case event := <-events:
			log.Info().Msgf("event %v", event)
			if event.Type == enet.EventTypeReceive {
				log.Info().Msgf("packet %d", len(event.Packet.Data))

			}
		}
	}

	host.Shutdown()
}
