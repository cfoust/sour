package maps

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"

	//"github.com/cfoust/sour/pkg/game"
	"github.com/rs/zerolog/log"
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
	value := make([]byte, bytes)
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

	//fmt.Printf("pos %d octsav %d\n", unpack.Tell(), octsav&0x7)

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
		log.Fatal().Msg("Map had invalid octsav")
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

		//fmt.Printf("a %d\n", unpack.Tell())

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
				//fmt.Printf("%d %d %d %d\n", surfaces[i].Lmid[0], surfaces[i].Lmid[1], surfaces[i].Verts, surfaces[i].NumVerts)
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

				//fmt.Printf("%d %t %t %t\n", vertMask, hasXYZ, hasUV, hasNorm)
				//fmt.Printf("b %d\n", unpack.Tell())

				if layerVerts == 4 {
					if hasXYZ && (vertMask&0x01) > 0 {
						unpack.Short()
						unpack.Short()
						unpack.Short()
						unpack.Short()
						hasXYZ = false
					}

					//fmt.Printf("b-1 %d\n", unpack.Tell())
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
					//fmt.Printf("c-2 %d\n", unpack.Tell())
				}

				//fmt.Printf("c %d\n", unpack.Tell())

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

func LoadVSlot(unpack *Unpacker, slot *VSlot, changed int32) error {
	slot.Changed = changed
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
		slot.Layer = unpack.Int()
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

func LoadVSlots(unpack *Unpacker, numVSlots int32) ([]*VSlot, error) {
	leftToRead := numVSlots

	vslots := make([]*VSlot, 0)
	prev := make([]int32, numVSlots)

	addSlot := func() *VSlot {
		vslot := VSlot{}
		vslot.Index = int32(len(vslots))
		vslots = append(vslots, &vslot)
		return &vslot
	}

	for leftToRead > 0 {
		changed := unpack.Int()
		if changed < 0 {
			for i := 0; i < int(-1*changed); i++ {
				addSlot()
			}
			leftToRead += changed
		} else {
			prev[len(vslots)] = unpack.Int()
			slot := addSlot()
			LoadVSlot(unpack, slot, changed)
			leftToRead--
		}
	}

	//loopv(vslots) if(vslots.inrange(prev[i])) vslots[prev[i]]->next = vslots[i];

	return vslots, nil
}

func Decode(data []byte) (*GameMap, error) {
	gameMap := GameMap{}
	reader := bytes.NewReader(data)
	unpack := NewUnpacker(reader)

	header := FileHeader{}
	err := unpack.Read(&header)
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

	mapHeader.Version = header.Version
	mapHeader.HeaderSize = header.HeaderSize
	mapHeader.WorldSize = header.WorldSize
	mapHeader.LightMaps = header.LightMaps
	mapHeader.BlendMap = newFooter.BlendMap
	mapHeader.NumVars = newFooter.NumVars
	mapHeader.NumVSlots = newFooter.NumVSlots

	gameMap.Header = mapHeader

	log.Printf("Version %d", header.Version)
	gameMap.Vars = make(map[string]Variable)

	// These are apparently arbitrary Sauerbraten variables a map can set
	for i := 0; i < int(newFooter.NumVars); i++ {
		_type := unpack.Char()
		name := unpack.String()

		switch VariableType(_type) {
		case VariableTypeInt:
			value := unpack.Int()
			gameMap.Vars[name] = IntVariable(value)
			//log.Printf("%s=%d", name, value)
		case VariableTypeFloat:
			value := unpack.Float()
			gameMap.Vars[name] = FloatVariable(value)
			//log.Printf("%s=%f", name, value)
		case VariableTypeString:
			value := unpack.String()
			gameMap.Vars[name] = StringVariable(value)
			//log.Printf("%s=%s", name, value)
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

	vSlotData, err := LoadVSlots(unpack, newFooter.NumVSlots)
	gameMap.VSlots = vSlotData

	cube, err := LoadChildren(unpack, header.Version)
	if err != nil {
		return nil, err
	}

	gameMap.Cubes = cube

	return &gameMap, nil
	return nil, nil
}

func FromGZ(data []byte) (*GameMap, error) {
	buffer := bytes.NewReader(data)
	gz, err := gzip.NewReader(buffer)
	defer gz.Close()
	if err != nil {
		return nil, err
	}

	rawBytes, err := io.ReadAll(gz)
	if err == gzip.ErrChecksum {
		log.Warn().Msg("Map file had invalid checksum")
	} else if err != nil {
		return nil, err
	}

	return Decode(rawBytes)
}

func FromFile(filename string) (*GameMap, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	buffer, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return FromGZ(buffer)
}
