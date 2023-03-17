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

func Dump(filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	gameMap, err := maps.NewMap()
	if err != nil {
		return err
	}

	mapBytes, err := gameMap.EncodeOGZ()
	if err != nil {
		return err
	}
	buffer := bytes.NewReader(mapBytes)

	_, err = io.Copy(out, buffer)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)

	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		log.Fatal().Msg("You must provide at least one argument.")
	}

	switch args[0] {
	case "dump":
		dumpCmd.Parse(args[1:])
		args := dumpCmd.Args()
		if len(args) != 1 {
			log.Fatal().Msg("You must provide only a single argument.")
		}
		err := Dump(args[0])
		if err != nil {
			log.Fatal().Err(err).Msg("could not dump map")
		}
	}
}
