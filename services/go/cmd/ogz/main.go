package main

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/maps/api"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Dump(filename string) error {
	gameMap, err := maps.FromFile(filename)
	if err != nil {
		return err
	}

	apiMap, err := gameMap.ToAPI()
	if err != nil {
		return err
	}

	data, err := json.Marshal(apiMap)
	if err != nil {
		return err
	}

	os.Stdout.Write(data)

	var decoded api.Map
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}

	//log.Info().Msgf("%+v", decoded)

	//for _, entity := range decoded.Entities {
	//log.Info().Msgf("%+v", entity.Info)
	//}

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
