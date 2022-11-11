package main

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"log"
	"os"
)

type Header struct {
	Magic      [4]byte
	Version    int32
	HeaderSize int32
	WorldSize  int32
	NumEnts    int32
	NumPVs     int32
	LightMaps  int32
	BlendMap   int32
	NumVars    int32
	NumVSlots  int32
}

func main() {
	args := os.Args[1:]

	if len(args) != 1 {
		log.Fatal("Please provide at least one argument.")
		return
	}

	filename := args[0]

	file, err := os.Open(filename)

	if err != nil {
		log.Fatal(err)
	}

	gz, err := gzip.NewReader(file)

	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	defer gz.Close()

	scanner := bufio.NewReader(gz)

	header := Header{}
	err = binary.Read(scanner, binary.LittleEndian, &header)
	if err != nil {
		log.Fatal(err)
		log.Fatal("How did I end up here?")
		return
	}

	log.Printf("Version %d", header.Version)
	log.Printf("HeaderSize %d", header.HeaderSize)
	log.Printf("WorldSize %d", header.WorldSize)
	log.Printf("NumEnts %d", header.NumEnts)
	log.Printf("NumPVs %d", header.NumPVs)
	log.Printf("LightMaps %d", header.LightMaps)
	log.Printf("BlendMap %d", header.BlendMap)
	log.Printf("NumVars %d", header.NumVars)
	log.Printf("NumVSlots %d", header.NumVSlots)
}
