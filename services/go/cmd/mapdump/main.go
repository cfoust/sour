package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"log"
	"os"
	"io"
)

type Header struct {
	Magic      [4]byte
	Version    int32
	HeaderSize int32
	WorldSize  int32
	NumEnts    int32
	NumPVs     int32
	LightMaps  int32
}

type NewHeader struct {
	BlendMap   int32
	NumVars    int32
	NumVSlots  int32
}

// For versions <=28
type OldHeader struct {
	LightPrecision int32
	LightError     int32
	LightLOD       int32
	Ambient        byte
	WaterColor     [3]byte
	BlendMap       byte
	LerpAngle      byte
	LerpSubDiv     byte
	LerpSubDivSize byte
	BumpError      byte
	SkyLight       [3]byte
	LavaColor      [3]byte
	WaterfallColor [3]byte
	Reserved       [10]byte
	MapTitle       [128]byte
}

const (
	ID_VAR  byte = 0
	ID_FVAR      = 1
	ID_SVAR      = 2
)

const MAX_MAP_SIZE = 8388608

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

	// Read the entire file into memory -- maps are small
	buffer := make([]byte, MAX_MAP_SIZE)
	bytesRead, err := gz.Read(buffer)

	if bytesRead == MAX_MAP_SIZE {
		log.Fatal("Map file too big")
		return
	}

	reader := bytes.NewReader(buffer)

	header := Header{}
	err = binary.Read(reader, binary.LittleEndian, &header)
	if err != nil {
		log.Fatal(err)
		log.Fatal("How did I end up here?")
		return
	}

	newHeader := NewHeader{}
	oldHeader := OldHeader{}
	if header.Version <= 28 {
		reader.Seek(224, io.SeekStart) // 7 * 32, like in worldio.cpp
		err = binary.Read(reader, binary.LittleEndian, &oldHeader)
		if err != nil {
			log.Fatal(err)
			return
		}

		newHeader.BlendMap = int32(oldHeader.BlendMap)
		newHeader.NumVars = 0
		newHeader.NumVSlots = 0
	} else {
		err = binary.Read(reader, binary.LittleEndian, &newHeader)

		// v29 had one fewer field
		if header.Version == 29 {
			reader.Seek(-4, io.SeekCurrent)
		}

		newHeader.NumVSlots = 0
	}

	log.Printf("Version %d", header.Version)
	log.Printf("HeaderSize %d", header.HeaderSize)
	log.Printf("WorldSize %d", header.WorldSize)
	log.Printf("NumEnts %d", header.NumEnts)
	log.Printf("NumPVs %d", header.NumPVs)
	log.Printf("LightMaps %d", header.LightMaps)
	log.Printf("BlendMap %d", newHeader.BlendMap)
	log.Printf("NumVars %d", newHeader.NumVars)
	log.Printf("NumVSlots %d", newHeader.NumVSlots)

	var (
		_type     byte
		nameBytes int8
	)

	// These are apparently arbitrary Sauerbraten variables a map can set
	for i := 0; i < int(newHeader.NumVars); i++ {
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
			var valueBytes uint16
			err = binary.Read(reader, binary.LittleEndian, &valueBytes)
			value := make([]byte, valueBytes+1)
			err = binary.Read(reader, binary.LittleEndian, &value)
			reader.Seek(-1, io.SeekCurrent)
			log.Printf("%s='%s'", name, value)
		}
	}

	gameType := "fps"
	if (header.Version >= 16) {
		var typeBytes uint8
		binary.Read(reader, binary.LittleEndian, &typeBytes)
		fileGameType := make([]byte, typeBytes+1)
		reader.Read(fileGameType)
		gameType = string(fileGameType)
	}

	log.Printf("type %s", gameType)
}
