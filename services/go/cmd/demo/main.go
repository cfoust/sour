package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"os"

	"github.com/cfoust/sour/pkg/game"
)

type DemoHeader struct {
	Magic    [16]byte
	Version  int32
	Protocol int32
}

type SectionHeader struct {
	Millis int32
	Channel int32
	Length int32
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		log.Fatal("You must provide only a single argument.")
	}

	file, err := os.Open(args[0])

	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	defer gz.Close()

	buffer, err := io.ReadAll(gz)
	reader := bytes.NewReader(buffer)

	header := DemoHeader{}
	err = binary.Read(reader, binary.LittleEndian, &header)

	log.Print(header.Version)
	log.Print(header.Protocol)

	section := SectionHeader{}
	for {
		err = binary.Read(reader, binary.LittleEndian, &section)
		log.Printf("%d %d %d", section.Millis, section.Channel, section.Length)

		bytes := make([]byte, section.Length)
		reader.Read(bytes)
		log.Print(game.Read(bytes))
		break
	}
}
