package main

import (
	"compress/gzip"
	"flag"
	"io"
	"os"
	"log"
	"time"

	I "github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"

	"github.com/rs/zerolog"
	Z "github.com/rs/zerolog/log"
)

type DemoHeader struct {
	Magic    [16]byte
	Version  int32
	Protocol int32
}

type SectionHeader struct {
	From    bool
	Millis  int32
	Channel int32
	Length  int32
}

func main() {
	Z.Logger = Z.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		Z.Fatal().Msg("You must provide only a single argument.")
	}

	file, err := os.Open(args[0])

	if err != nil {
		Z.Fatal().Err(err).Msg("could not open demo")
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		Z.Fatal().Err(err).Msg("could not unzip demo")
	}

	defer file.Close()
	defer gz.Close()

	buffer, err := io.ReadAll(gz)

	p := I.Buffer(buffer)

	section := SectionHeader{}
	for {
		if len(p) == 0 {
			return
		}
		p.Get(&section)

		bytes, _ := p.GetBytes(int(section.Length))
		messages, err := P.Decode(bytes, true)
		if err != nil {
			Z.Error().Err(err).Msg("failed to parse messages")
			continue
		}

		sender := "->"
		if !section.From {
			sender = "<-"
		}

		for _, message := range messages {
			log.Printf(
				"%s |%8d| %s %+v",
				sender,
				section.Millis,
				message.Type().String(),
				message,
			)
		}
	}
}
