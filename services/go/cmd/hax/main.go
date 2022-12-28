package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/game"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	flag.Parse()
	args := flag.Args()

	out, err := os.Create(args[0])
	if err != nil {
		log.Fatal().Err(err).Msg("could not create map")
	}
	defer out.Close()

	gameMap := maps.NewMap()
	gameMap.Vars["cloudlayer"] = maps.StringVariable("")
	gameMap.Vars["skyboxcolour"] = maps.IntVariable(0)
	gameMap.Vars["maptitle"] = maps.StringVariable("can_teleport_1 = [ echo test ]")
	gameMap.Header.WorldSize = 64

	gameMap.Entities = append(gameMap.Entities, maps.Entity{
		Type: game.EntityTypeTeleport,
		Attr3: 1,
	})

	log.Info().Msgf("%v", gameMap.Entities)

	mapBytes, err := gameMap.EncodeOGZ()
	if err != nil {
		log.Fatal().Err(err).Msg("could not encode map")
	}
	buffer := bytes.NewReader(mapBytes)

	_, err = io.Copy(out, buffer)
	if err != nil {
		return
	}
}
