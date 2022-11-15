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

const (
	VSLOT_SHPARAM  byte = iota
	VSLOT_SCALE         = iota
	VSLOT_ROTATION      = iota
	VSLOT_OFFSET        = iota
	VSLOT_SCROLL        = iota
	VSLOT_LAYER         = iota
	VSLOT_ALPHA         = iota
	VSLOT_COLOR         = iota
	VSLOT_NUM           = iota
)

type Unpacker struct {
	Reader *bytes.Reader
}

func NewUnpacker(reader *bytes.Reader) *Unpacker {
	unpacker := Unpacker{}
	unpacker.Reader = reader
	return &unpacker
}

func (unpack *Unpacker) Read(data any) error {
	return binary.Read(unpack.Reader, binary.LittleEndian, data)
}

func (unpack *Unpacker) Float() float32 {
	var value float32
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Int() int32 {
	var value int32
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Char() byte {
	var value byte
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) Short() uint16 {
	var value uint16
	unpack.Read(&value)
	return value
}

func (unpack *Unpacker) String() string {
	bytes := unpack.Short()
	log.Printf("bytes=%d", bytes)
	value := make([]byte, bytes+1)
	unpack.Read(value)
	return string(value)
}

func (unpack *Unpacker) StringByte() string {
	var bytes byte
	unpack.Read(&bytes)
	value := make([]byte, bytes+1)
	unpack.Read(value)
	return string(value)
}

func (unpack *Unpacker) Skip(bytes int64) {
	unpack.Reader.Seek(bytes, io.SeekCurrent)
}

func (unpack *Unpacker) Tell() int64 {
	pos, _ := unpack.Reader.Seek(0, io.SeekCurrent)
	return pos
}

func LoadCube(unpack *Unpacker, cube *Cube, mapVersion int32) error {
	log.Printf("pos=%d", unpack.Tell())

	//var hasChildren = false
	octsav := unpack.Char()

	log.Printf("octsav=%d", octsav&0x7)

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(unpack, mapVersion)
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
		unpack.Read(&cube.Edges)
		break
	}

	if (octsav & 0x7) > 4 {
		log.Fatal("Map had invalid octsav")
		return errors.New("Map had invalid octsav")
	}

	for i := 0; i < 6; i++ {
		if mapVersion < 14 {
			texture := unpack.Char()
			cube.Texture[i] = uint16(texture)
		} else {
			texture := unpack.Short()
			cube.Texture[i] = texture
		}
		//log.Printf("Texture[%d]=%d", i, cube.Texture[i])
	}

	if mapVersion < 7 {
		unpack.Skip(3)
	} else if mapVersion <= 31 {
		mask := unpack.Char()

		if (mask & 0x80) > 0 {
			unpack.Skip(1)
		}

		surfaces := make([]SurfaceCompat, 12)
		normals := make([]NormalsCompat, 6)
		merges := make([]MergeCompat, 6)

		var numSurfaces = 6
		if (mask & 0x3F) > 0 {
			for i := 0; i < numSurfaces; i++ {
				if i >= 6 || mask&(1<<i) > 0 {
					unpack.Read(&surfaces[i])
					if i < 6 {
						if (mask & 0x40) > 0 {
							unpack.Read(&normals[i])
						}
						if (surfaces[i].Layer & 2) > 0 {
							numSurfaces++
						}
					}
				}
			}
		}

		if mapVersion >= 20 && (octsav&0x80) > 0 {
			merged := unpack.Char()
			cube.Merged = merged & 0x3F
			if (merged & 0x80) > 0 {
				mask := unpack.Char()
				if mask > 0 {
					for i := 0; i < 6; i++ {
						if (mask & (1 << i)) > 0 {
							unpack.Read(&merges[i])
						}
					}
				}
			}
		}
	}

	return nil
}

func LoadChildren(unpack *Unpacker, mapVersion int32) ([]Cube, error) {
	children := make([]Cube, CUBE_FACTOR)

	for i := 0; i < CUBE_FACTOR; i++ {
		err := LoadCube(unpack, &children[i], mapVersion)
		if err != nil {
			return nil, err
		}
	}

	return children, nil
}

func LoadVSlot(unpack *Unpacker, changed int32) error {
	if (changed & (1 << VSLOT_SHPARAM)) > 0 {
		numParams := unpack.Short()

		for i := 0; i < int(numParams); i++ {
			_ = unpack.String()
			// TODO vslots
			for k := 0; k < 4; k++ {
				unpack.Float()
			}
		}
	}

	if (changed & (1 << VSLOT_SCALE)) > 0 {
		unpack.Float()
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		unpack.Int()
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		unpack.Int()
		unpack.Int()
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		unpack.Float()
		unpack.Float()
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		unpack.Int()
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		unpack.Float()
		unpack.Float()
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		for k := 0; k < 3; k++ {
			unpack.Float()
		}
	}

	return nil
}

func LoadVSlots(unpack *Unpacker, numVSlots int32) error {
	leftToRead := numVSlots

	for leftToRead > 0 {
		changed := unpack.Int()
		if changed < 0 {
			leftToRead += changed
		} else {
			unpack.Int()
			LoadVSlot(unpack, changed)
			leftToRead--
		}
	}

	return nil
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

	unpack := NewUnpacker(reader)

	header := Header{}
	err = unpack.Read(&header)
	if err != nil {
		log.Fatal(err)
		log.Fatal("How did I end up here?")
		return
	}

	newHeader := NewHeader{}
	oldHeader := OldHeader{}
	if header.Version <= 28 {
		reader.Seek(28, io.SeekStart) // 7 * 4, like in worldio.cpp
		err = unpack.Read(&oldHeader)
		if err != nil {
			log.Fatal(err)
			return
		}

		newHeader.BlendMap = int32(oldHeader.BlendMap)
		newHeader.NumVars = 0
		newHeader.NumVSlots = 0
	} else {
		unpack.Read(&newHeader)

		// v29 had one fewer field
		if header.Version == 29 {
			reader.Seek(-4, io.SeekCurrent)
		}
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

	// These are apparently arbitrary Sauerbraten variables a map can set
	for i := 0; i < int(newHeader.NumVars); i++ {
		_type := unpack.Char()
		name := unpack.StringByte()

		switch _type {
		case ID_VAR:
			value := unpack.Int()
			log.Printf("%s=%d", name, value)
		case ID_FVAR:
			value := unpack.Float()
			log.Printf("%s=%f", name, value)
		case ID_SVAR:
			value := unpack.String()
			reader.Seek(-1, io.SeekCurrent)
			log.Printf("%s='%s'", name, value)
		}
	}

	gameType := "fps"
	if header.Version >= 16 {
		gameType = unpack.StringByte()
	}
	log.Printf("GameType %s", gameType)

	gameMap.GameType = gameType

	//// We just skip extras
	if header.Version >= 16 {
		unpack.Skip(2) // eif
		extraBytes := unpack.Short()
		unpack.Skip(int64(extraBytes))
	}

	// Also skip the texture MRU
	if header.Version < 14 {
		unpack.Skip(256)
	} else {
		numMRUBytes := unpack.Short()
		unpack.Skip(int64(numMRUBytes * 2))
	}

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}
		unpack.Read(&entity)

		if !InsideWorld(header.WorldSize, entity.Position) {
			log.Printf("Entity outside of world")
			log.Printf("entity type %d", entity.Type)
			log.Printf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)
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
		LoadVSlots(unpack, newHeader.NumVSlots)
	}

	_, err = LoadChildren(unpack, header.Version)
}
