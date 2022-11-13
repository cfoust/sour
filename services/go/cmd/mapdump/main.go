package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io"
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
}

type NewHeader struct {
	BlendMap  int32
	NumVars   int32
	NumVSlots int32
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

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Entity struct {
	Position Vector
	Attr1    int16
	Attr2    int16
	Attr3    int16
	Attr4    int16
	Attr5    int16
	Type     byte
	Reserved byte
}

type SurfaceInfo struct {
	Lmid     [2]byte
	Verts    byte
	NumVerts byte
}

const CUBE_FACTOR = 8

type Cube struct {
	Children    *[]Cube
	SurfaceInfo [6]SurfaceInfo
	Edges       [12]byte
	Texture     [6]uint16
	Material    uint16
	Merged      byte
	Escaped     byte
}

type GameMap struct {
	Header    Header
	NewHeader NewHeader
	GameType  string
}

const (
	ID_VAR  byte = iota
	ID_FVAR      = iota
	ID_SVAR      = iota
)

const (
	ET_EMPTY        byte = iota
	ET_LIGHT             = iota
	ET_MAPMODEL          = iota
	ET_PLAYERSTART       = iota
	ET_ENVMAP            = iota
	ET_PARTICLES         = iota
	ET_SOUND             = iota
	ET_SPOTLIGHT         = iota
	ET_GAMESPECIFIC      = iota
)

const (
	OCTSAV_CHILDREN byte = iota
	OCTSAV_EMPTY         = iota
	OCTSAV_SOLID         = iota
	OCTSAV_NORMAL        = iota
	OCTSAV_LODCUB        = iota
)

const MAX_MAP_SIZE = 8388608

func LoadCube(reader *bytes.Reader, cube *Cube, version int32) error {
	//var hasChildren = false
	var octSav byte
	binary.Read(reader, binary.LittleEndian, &octSav)

	switch octSav {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(reader, version)
		if err != nil {
			return err
		}
		cube.Children = &children
	case OCTSAV_LODCUB:
		//hasChildren = true
		break
	case OCTSAV_EMPTY:
		// TODO emptyfaces
		break
	case OCTSAV_SOLID:
		// TODO solidfaces
		break
	case OCTSAV_NORMAL:
		// TODO handle edges
		reader.Seek(12, io.SeekCurrent)
		break
	}

	for i := 0; i < 6; i++ {
		if version < 14 {
			var texture byte
			binary.Read(reader, binary.LittleEndian, &texture)
			cube.Texture[i] = uint16(texture)
		} else {
			var texture uint16
			binary.Read(reader, binary.LittleEndian, &texture)
			cube.Texture[i] = texture
		}
		log.Printf("texture[%d]=%d", i, cube.Texture[i])
	}

	return nil
}

func LoadChildren(reader *bytes.Reader, version int32) ([]Cube, error) {
	children := make([]Cube, CUBE_FACTOR)

	for i := 0; i < CUBE_FACTOR; i++ {
		err := LoadCube(reader, &children[i], version)
		if err != nil {
			return nil, err
		}
	}

	return children, nil
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

	gameMap := GameMap{}

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
		reader.Seek(28, io.SeekStart) // 7 * 4, like in worldio.cpp
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

	gameMap.Header = header
	gameMap.NewHeader = newHeader

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
	if header.Version >= 16 {
		var typeBytes uint8
		binary.Read(reader, binary.LittleEndian, &typeBytes)
		fileGameType := make([]byte, typeBytes+1)
		reader.Read(fileGameType)
		gameType = string(fileGameType)
	}

	gameMap.GameType = gameType

	// We just skip extras
	var eif uint16 = 0
	if header.Version >= 16 {
		binary.Read(reader, binary.LittleEndian, &eif)
		var extraSize uint16
		binary.Read(reader, binary.LittleEndian, &extraSize)

		// TODO do we need extras?
		reader.Seek(int64(extraSize), io.SeekCurrent)
	}

	// Also skip the texture MRU
	if header.Version < 14 {
		reader.Seek(256, io.SeekCurrent)
	} else {
		var numMRUBytes uint16
		binary.Read(reader, binary.LittleEndian, &numMRUBytes)
		log.Printf("numMRUBytes %d", numMRUBytes)
		reader.Seek(int64(numMRUBytes*2), io.SeekCurrent)
	}

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}
		binary.Read(reader, binary.LittleEndian, &entity)

		//log.Printf("entity type %d", entity.Type)
		//log.Printf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)

		if header.Version <= 14 && entity.Type == ET_MAPMODEL {
			entity.Position.Z += float32(entity.Attr3)
			entity.Attr3 = 0
			entity.Attr4 = 0
		}
	}

	// vslots
	// TODO do we ever actually need v slots?
	if newHeader.NumVSlots > 0 {
		log.Fatal("Maps with vslots are not supported")
		return
	}

	_, err = LoadChildren(reader, header.Version)
}
