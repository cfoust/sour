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

const (
	ID_VAR  byte = 0
	ID_FVAR      = 1
	ID_SVAR      = 2
)

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

	reader := bufio.NewReader(gz)

	header := Header{}
	err = binary.Read(reader, binary.LittleEndian, &header)
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

	var (
		_type     byte
		nameBytes int8
	)
	for i := 0; i < int(header.NumVars); i++ {
		err = binary.Read(reader, binary.LittleEndian, &_type)
		err = binary.Read(reader, binary.LittleEndian, &nameBytes)

		name := make([]byte, nameBytes+1)
		_, err = reader.Read(name)

		switch _type {
		case ID_VAR:
			var value int32
			err = binary.Read(reader, binary.LittleEndian, &value)
			log.Printf("%s=%d", name, value)
		case ID_FVAR:
			var value float32
			err = binary.Read(reader, binary.LittleEndian, &value)
			log.Printf("%s=%f", name, value)
		case ID_SVAR:
			var valueBytes int8
			err = binary.Read(reader, binary.LittleEndian, &valueBytes)
			value := make([]byte, valueBytes+1)
			err = binary.Read(reader, binary.LittleEndian, &value)
			log.Printf("%s=%s", name, value)
		}
	}
}
