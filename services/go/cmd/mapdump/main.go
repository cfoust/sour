package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
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

type SurfaceCompat struct {
	TexCoords [8]byte
	Width     byte
	Height    byte
	X         uint16
	Y         uint16
	Lmid      byte
	Layer     byte
}

type BVec struct {
	X byte
	Y byte
	Z byte
}

type NormalsCompat struct {
	Normals [4]BVec
}

type MergeCompat struct {
	U1 uint16
	U2 uint16
	V1 uint16
	V2 uint16
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

func LoadCube(reader *bytes.Reader, cube *Cube, mapVersion int32) error {
	pos, _ := reader.Seek(0, io.SeekCurrent)
	log.Printf("pos=%d", pos)

	//var hasChildren = false
	var octsav byte
	binary.Read(reader, binary.LittleEndian, &octsav)

	log.Printf("octsav=%d", octsav&0x7)

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(reader, mapVersion)
		if err != nil {
			return err
		}
		cube.Children = &children
		return nil
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
		binary.Read(reader, binary.LittleEndian, &cube.Edges)
		break
	}

	if (octsav & 0x7) > 4 {
		log.Fatal("Map had invalid octsav")
		return errors.New("Map had invalid octsav")
	}

	for i := 0; i < 6; i++ {
		if mapVersion < 14 {
			var texture byte
			binary.Read(reader, binary.LittleEndian, &texture)
			cube.Texture[i] = uint16(texture)
		} else {
			var texture uint16
			binary.Read(reader, binary.LittleEndian, &texture)
			cube.Texture[i] = texture
		}
		//log.Printf("Texture[%d]=%d", i, cube.Texture[i])
	}

	if mapVersion < 7 {
		reader.Seek(3, io.SeekCurrent)
	} else if mapVersion <= 31 {
		var mask byte
		binary.Read(reader, binary.LittleEndian, &mask)

		if (mask & 0x80) > 0 {
			// TODO convert materials?
			reader.Seek(1, io.SeekCurrent)
		}

		surfaces := make([]SurfaceCompat, 12)
		normals := make([]NormalsCompat, 6)
		merges := make([]MergeCompat, 6)

		var numSurfaces = 6
		if (mask & 0x3F) > 0 {
			for i := 0; i < numSurfaces; i++ {
				if i >= 6 || mask&(1<<i) > 0 {
					binary.Read(reader, binary.LittleEndian, &surfaces[i])
					if i < 6 {
						if (mask & 0x40) > 0 {
							binary.Read(reader, binary.LittleEndian, &normals[i])
						}
						if (surfaces[i].Layer & 2) > 0 {
							numSurfaces++
						}
					}
				}
			}
		}

		if mapVersion >= 20 && (octsav&0x80) > 0 {
			var merged byte
			binary.Read(reader, binary.LittleEndian, &merged)
			cube.Merged = merged & 0x3F
			if (merged & 0x80) > 0 {
				var mask byte
				binary.Read(reader, binary.LittleEndian, &mask)
				if mask > 0 {
					for i := 0; i < 6; i++ {
						if (mask & (1 << i)) > 0 {
							binary.Read(reader, binary.LittleEndian, &merges[i])
						}
					}
				}
			}
		}
	}

	return nil
}

func LoadChildren(reader *bytes.Reader, mapVersion int32) ([]Cube, error) {
	children := make([]Cube, CUBE_FACTOR)

	for i := 0; i < CUBE_FACTOR; i++ {
		err := LoadCube(reader, &children[i], mapVersion)
		if err != nil {
			return nil, err
		}
	}

	return children, nil
}

func InsideWorld(size int32, vector Vector) bool {
	return vector.X >= 0 && vector.X < float32(size) && vector.Y >= 0 && vector.Y < float32(size) && vector.Z >= 0 && vector.Z < float32(size)
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
	buffer, err := io.ReadAll(gz)

	if err != nil {
		log.Fatal(err)
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
		binary.Read(reader, binary.LittleEndian, &newHeader)

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

		log.Printf("entity type %d", entity.Type)
		log.Printf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)

		if !InsideWorld(header.WorldSize, entity.Position) {
			log.Fatal("Entity outside of world")
		}

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
