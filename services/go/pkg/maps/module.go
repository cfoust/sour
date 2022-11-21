package maps

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
	Version    int32
	HeaderSize int32
	WorldSize  int32
	NumEnts    int32
	NumPVs     int32
	LightMaps  int32
	BlendMap   int32
	NumVars    int32
	NumVSlots  int32
	GameType   string
}

type FileHeader struct {
	Magic      [4]byte
	Version    int32
	HeaderSize int32
	WorldSize  int32
	NumEnts    int32
	NumPVs     int32
	LightMaps  int32
}

type NewFooter struct {
	BlendMap  int32
	NumVars   int32
	NumVSlots int32
}

// For versions <=28
type OldFooter struct {
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

const (
	LAYER_TOP    byte = (1 << 5)
	LAYER_BOTTOM      = (1 << 6)
	LAYER_DUP         = (1 << 7)
	LAYER_BLEND       = LAYER_TOP | LAYER_BOTTOM
	MAXFACEVERTS      = 15
)

func (surface *SurfaceInfo) TotalVerts() byte {
	if (surface.NumVerts & LAYER_DUP) > 0 {
		return (surface.NumVerts & MAXFACEVERTS) * 2
	}
	return surface.NumVerts & MAXFACEVERTS
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
	Header   Header
	Entities []Entity
	Vars     map[string]int32
	FVars    map[string]float32
	SVars    map[string]string
	Cubes    []Cube
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
	//log.Printf("pos=%d", unpack.Tell())

	var hasChildren = false
	octsav := unpack.Char()

	//log.Printf("octsav=%d", octsav&0x7)

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(unpack, mapVersion)
		if err != nil {
			return err
		}
		cube.Children = &children
		return nil
	case OCTSAV_LODCUB:
		hasChildren = true
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
	} else {
		// TODO material
		if (octsav & 0x40) > 0 {
			if mapVersion <= 32 {
				unpack.Char()
			} else {
				unpack.Short()
			}
		}

		// TODO merged
		if (octsav & 0x80) > 0 {
			unpack.Char()
		}

		if (octsav & 0x20) > 0 {
			surfMask := unpack.Char()
			unpack.Char() // totalVerts

			surfaces := make([]SurfaceInfo, 6)
			var offset byte
			offset = 0
			for i := 0; i < 6; i++ {
				if surfMask&(1<<i) == 0 {
					continue
				}

				unpack.Read(&surfaces[i])
				vertMask := surfaces[i].Verts
				numVerts := surfaces[i].TotalVerts()

				if numVerts == 0 {
					surfaces[i].Verts = 0
					continue
				}

				surfaces[i].Verts = offset
				offset += numVerts

				layerVerts := surfaces[i].NumVerts & MAXFACEVERTS
				hasXYZ := (vertMask & 0x04) != 0
				hasUV := (vertMask & 0x40) != 0
				hasNorm := (vertMask & 0x80) != 0

				if layerVerts == 4 {
					if hasXYZ && (vertMask&0x01) > 0 {
						unpack.Short()
						unpack.Short()
						unpack.Short()
						unpack.Short()
						hasXYZ = false
					}

					if hasUV && (vertMask&0x02) > 0 {
						unpack.Short()
						unpack.Short()
						unpack.Short()
						unpack.Short()

						if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
							unpack.Short()
							unpack.Short()
							unpack.Short()
							unpack.Short()
						}

						hasUV = false
					}
				}

				if hasNorm && (vertMask&0x08) > 0 {
					unpack.Short()
					hasNorm = false
				}

				if hasXYZ || hasUV || hasNorm {
					for k := 0; k < int(layerVerts); k++ {
						if hasXYZ {
							unpack.Short()
							unpack.Short()
						}

						if hasUV {
							unpack.Short()
							unpack.Short()
						}

						if hasNorm {
							unpack.Short()
						}
					}
				}

				if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
					for k := 0; k < int(layerVerts); k++ {
						if hasUV {
							unpack.Short()
							unpack.Short()
						}
					}
				}
			}
		}
	}

	if hasChildren {
		children, _ := LoadChildren(unpack, mapVersion)
		cube.Children = &children
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
			_ = unpack.StringByte()
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

func LoadMap(filename string) (*GameMap, error) {
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

	if err == gzip.ErrChecksum {
		log.Printf("Map file had invalid checksum")
	} else if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(buffer)

	unpack := NewUnpacker(reader)

	header := FileHeader{}
	err = unpack.Read(&header)
	if err != nil {
		return nil, err
	}

	newFooter := NewFooter{}
	oldFooter := OldFooter{}
	if header.Version <= 28 {
		reader.Seek(28, io.SeekStart) // 7 * 4, like in worldio.cpp
		err = unpack.Read(&oldFooter)
		if err != nil {
			return nil, err
		}

		newFooter.BlendMap = int32(oldFooter.BlendMap)
		newFooter.NumVars = 0
		newFooter.NumVSlots = 0
	} else {
		unpack.Read(&newFooter)

		if header.Version <= 29 {
			newFooter.NumVSlots = 0
		}

		// v29 had one fewer field
		if header.Version == 29 {
			reader.Seek(-4, io.SeekCurrent)
		}
	}

	mapHeader := Header{}
	gameMap.Header = mapHeader

	mapHeader.Version = header.Version
	mapHeader.HeaderSize = header.HeaderSize
	mapHeader.WorldSize = header.WorldSize
	mapHeader.NumEnts = header.NumEnts
	mapHeader.NumPVs = header.NumPVs
	mapHeader.LightMaps = header.LightMaps
	mapHeader.BlendMap = newFooter.BlendMap
	mapHeader.NumVars = newFooter.NumVars
	mapHeader.NumVSlots = newFooter.NumVSlots

	log.Printf("Version %d", header.Version)
	//log.Printf("HeaderSize %d", header.HeaderSize)
	//log.Printf("WorldSize %d", header.WorldSize)
	//log.Printf("NumEnts %d", header.NumEnts)
	//log.Printf("NumPVs %d", header.NumPVs)
	//log.Printf("LightMaps %d", header.LightMaps)
	//log.Printf("BlendMap %d", newFooter.BlendMap)
	//log.Printf("NumVars %d", newFooter.NumVars)
	//log.Printf("NumVSlots %d", newFooter.NumVSlots)

	gameMap.Vars = make(map[string]int32)
	gameMap.FVars = make(map[string]float32)
	gameMap.SVars = make(map[string]string)

	// These are apparently arbitrary Sauerbraten variables a map can set
	for i := 0; i < int(newFooter.NumVars); i++ {
		_type := unpack.Char()
		name := unpack.StringByte()

		switch _type {
		case ID_VAR:
			value := unpack.Int()
			gameMap.Vars[name] = value
		case ID_FVAR:
			value := unpack.Float()
			gameMap.FVars[name] = value
		case ID_SVAR:
			value := unpack.String()
			reader.Seek(-1, io.SeekCurrent)
			gameMap.SVars[name] = value
		}
	}

	gameType := "fps"
	if header.Version >= 16 {
		gameType = unpack.StringByte()
	}
	//log.Printf("GameType %s", gameType)

	mapHeader.GameType = gameType

	// We just skip extras
	var eif uint16 = 0
	if header.Version >= 16 {
		eif = unpack.Short()
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

	entities := make([]Entity, header.NumEnts)

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}

		unpack.Read(&entity)

		if gameType != "fps" {
			if eif > 0 {
				unpack.Skip(int64(eif))
			}
		}

		if !InsideWorld(header.WorldSize, entity.Position) {
			log.Printf("Entity outside of world")
			log.Printf("entity type %d", entity.Type)
			log.Printf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)
		}

		if header.Version <= 14 && entity.Type == ET_MAPMODEL {
			entity.Position.Z += float32(entity.Attr3)
			entity.Attr3 = 0

			if entity.Attr4 > 0 {
				log.Printf("warning: mapmodel ent (index %d) uses texture slot %d", i, entity.Attr4)
			}

			entity.Attr4 = 0
		}

		entities[i] = entity
	}

	gameMap.Entities = entities

	// vslots
	// TODO do we ever actually need v slots?
	if newFooter.NumVSlots > 0 {
		LoadVSlots(unpack, newFooter.NumVSlots)
	}

	cube, err := LoadChildren(unpack, header.Version)
	if err != nil {
		return nil, err
	}

	gameMap.Cubes = cube

	return &gameMap, nil
}
