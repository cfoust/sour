package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/maps"

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
	gameMap.Vars["maptitle"] = maps.StringVariable("getdemo 0")
	rawBytes, err := gameMap.Encode()
	if err != nil {
		log.Fatal().Err(err).Msg("could not envode map")
	}
	log.Info().Msgf("map %v", rawBytes)

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
