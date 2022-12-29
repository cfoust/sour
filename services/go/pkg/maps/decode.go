package maps

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps/worldio"

	"github.com/rs/zerolog/log"
)

func GetString(p *game.Packet) (string, bool) {
	var length uint16
	err := p.GetRaw(&length)
	if err != nil {
		return "", false
	}
	value := make([]byte, length)
	err = p.GetRaw(&value)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func GetFloat(p *game.Packet) (float32, bool) {
	var value float32
	err := p.GetRaw(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func GetShort(p *game.Packet) (uint16, bool) {
	var value uint16
	err := p.GetRaw(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func GetInt(p *game.Packet) (int32, bool) {
	var value int32
	err := p.GetRaw(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func GetStringByte(p *game.Packet) (string, bool) {
	var length byte
	err := p.GetRaw(&length)
	if err != nil {
		return "", false
	}
	value := make([]byte, length+1)
	err = p.GetRaw(&value)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func convertOldMaterial(mat uint16) uint16 {
	return ((mat & 7) << MATF_VOLUME_SHIFT) | (((mat >> 3) & 3) << MATF_CLIP_SHIFT) | (((mat >> 5) & 7) << MATF_FLAG_SHIFT)
}

func LoadCube(p *game.Packet, cube *Cube, mapVersion int32) error {
	//log.Printf("pos=%d", unpack.Tell())

	var hasChildren = false
	octsav, _ := p.GetByte()

	//fmt.Printf("pos %d octsav %d\n", unpack.Tell(), octsav&0x7)

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		parent, err := LoadChildren(p, mapVersion)
		if err != nil {
			return err
		}
		cube.Children = parent.Children
		return nil
	case OCTSAV_LODCUB:
		hasChildren = true
		break
	case OCTSAV_EMPTY:
		cube.EmptyFaces()
		break
	case OCTSAV_SOLID:
		cube.SolidFaces()
		break
	case OCTSAV_NORMAL:
		p.GetRaw(&cube.Edges)
		break
	}

	if (octsav & 0x7) > 4 {
		log.Fatal().Msg("Map had invalid octsav")
		return errors.New("Map had invalid octsav")
	}

	for i := 0; i < 6; i++ {
		if mapVersion < 14 {
			texture, _ := p.GetByte()
			cube.Texture[i] = uint16(texture)
		} else {
			texture, _ := GetShort(p)
			cube.Texture[i] = texture
		}
		//log.Printf("Texture[%d]=%d", i, cube.Texture[i])
	}

	if mapVersion < 7 {
		p.Skip(3)
	} else if mapVersion <= 31 {
		mask, _ := p.GetByte()

		if (mask & 0x80) > 0 {
			mat, _ := p.GetByte()
			if mapVersion < 27 {
				matConv := []uint16{
					MAT_AIR,
					MAT_WATER,
					MAT_CLIP,
					MAT_GLASS | MAT_CLIP,
					MAT_NOCLIP,
					MAT_LAVA | MAT_DEATH,
					MAT_GAMECLIP,
					MAT_DEATH,
				}
				cube.Material = MAT_AIR
				if int(mat) < len(matConv) {
					cube.Material = matConv[mat]
				}
			} else {
				cube.Material = convertOldMaterial(uint16(mat))
			}
		}

		surfaces := make([]SurfaceCompat, 12)
		normals := make([]NormalsCompat, 6)
		merges := make([]MergeCompat, 6)

		var numSurfaces = 6
		var hasSurfs uint32
		var hasNorms uint32
		//var hasMerges uint32
		if (mask & 0x3F) > 0 {
			for i := 0; i < numSurfaces; i++ {
				if i >= 6 || mask&(1<<i) > 0 {
					surface := &surfaces[i]
					p.GetRaw(surface)
					if mapVersion < 10 {
						surface.Lmid++
					}
					if mapVersion < 18 {
						if surface.Lmid >= LMID_AMBIENT1 {
							surface.Lmid++
						}
						if surface.Lmid >= LMID_BRIGHT1 {
							surface.Lmid++
						}
					}
					if mapVersion < 19 {
						if surface.Lmid >= LMID_DARK {
							surface.Lmid += 2
						}
					}
					if i < 6 {
						if (mask & 0x40) > 0 {
							hasNorms |= 1 << i
							p.GetRaw(&normals[i])
						}
						if surface.Layer != 0 || surface.Lmid != LMID_AMBIENT {
							hasSurfs |= 1 << i
						}
						if (surfaces[i].Layer & 2) > 0 {
							numSurfaces++
						}
					}
				}
			}
		}

		if mapVersion <= 8 {
			// TODO edgespan2vectorcube
		}

		if mapVersion >= 20 && (octsav&0x80) > 0 {
			merged, _ := p.GetByte()
			cube.Merged = merged & 0x3F
			if (merged & 0x80) > 0 {
				mask, _ := p.GetByte()
				if mask > 0 {
					for i := 0; i < 6; i++ {
						if (mask & (1 << i)) > 0 {
							p.GetRaw(&merges[i])
						}
					}
				}
			}
		}
	} else {
		// TODO material
		if (octsav & 0x40) > 0 {
			if mapVersion <= 32 {
				p.GetByte()
			} else {
				GetShort(p)
			}
		}

		//fmt.Printf("a %d\n", unpack.Tell())

		// TODO merged
		if (octsav & 0x80) > 0 {
			p.GetByte()
		}

		if (octsav & 0x20) > 0 {
			surfMask, _ := p.GetByte()
			p.GetByte() // totalVerts

			surfaces := make([]SurfaceInfo, 6)
			var offset byte
			offset = 0
			for i := 0; i < 6; i++ {
				if surfMask&(1<<i) == 0 {
					continue
				}

				p.GetRaw(&surfaces[i])
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
						GetShort(p)
						GetShort(p)
						GetShort(p)
						GetShort(p)
						hasXYZ = false
					}

					//fmt.Printf("b-1 %d\n", unpack.Tell())
					if hasUV && (vertMask&0x02) > 0 {
						GetShort(p)
						GetShort(p)
						GetShort(p)
						GetShort(p)

						if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
							GetShort(p)
							GetShort(p)
							GetShort(p)
							GetShort(p)
						}

						hasUV = false
					}
					//fmt.Printf("c-2 %d\n", unpack.Tell())
				}

				//fmt.Printf("c %d\n", unpack.Tell())

				if hasNorm && (vertMask&0x08) > 0 {
					GetShort(p)
					hasNorm = false
				}

				if hasXYZ || hasUV || hasNorm {
					for k := 0; k < int(layerVerts); k++ {
						if hasXYZ {
							GetShort(p)
							GetShort(p)
						}

						if hasUV {
							GetShort(p)
							GetShort(p)
						}

						if hasNorm {
							GetShort(p)
						}
					}
				}

				if (surfaces[i].NumVerts & LAYER_DUP) > 0 {
					for k := 0; k < int(layerVerts); k++ {
						if hasUV {
							GetShort(p)
							GetShort(p)
						}
					}
				}
			}
		}
	}

	if hasChildren {
		parent, err := LoadChildren(p, mapVersion)
		if err != nil {
			return err
		}
		cube.Children = parent.Children
	}

	return nil
}

func LoadChildren(p *game.Packet, mapVersion int32) (*Cube, error) {
	cube := NewCubes(F_EMPTY, MAT_AIR)

	for i := 0; i < CUBE_FACTOR; i++ {
		err := LoadCube(p, cube.Children[i], mapVersion)
		if err != nil {
			return nil, err
		}
	}

	return cube, nil
}

func PrintCube(c *worldio.Cube) {
}

func LoadChildrenCube(p *game.Packet, mapVersion int32) (*Cube, error) {
	out := worldio.Loadchildren_buf(uintptr(unsafe.Pointer(&(*p)[0])), int64(len(*p)))
	log.Info().Msgf("%+v %d", out, *out.GetTexture())
	return nil, nil
}

func LoadVSlot(p *game.Packet, slot *VSlot, changed int32) error {
	slot.Changed = changed
	if (changed & (1 << VSLOT_SHPARAM)) > 0 {
		numParams, _ := GetShort(p)

		for i := 0; i < int(numParams); i++ {
			param := SlotShaderParam{}
			name, _ := GetStringByte(p)

			// TODO getshaderparamname
			param.Name = name
			for k := 0; k < 4; k++ {
				value, _ := GetFloat(p)
				param.Val[k] = value
			}
			slot.Params = append(slot.Params, param)
		}
	}

	if (changed & (1 << VSLOT_SCALE)) > 0 {
		p.GetRaw(&slot.Scale)
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		p.GetRaw(&slot.Rotation)
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		p.GetRaw(
			&slot.Offset.X,
			&slot.Offset.Y,
		)
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		p.GetRaw(
			&slot.Scroll.X,
			&slot.Scroll.Y,
		)
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		p.GetRaw(&slot.Layer)
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		p.GetRaw(
			&slot.AlphaFront,
			&slot.AlphaBack,
		)
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		p.GetRaw(
			&slot.ColorScale.X,
			&slot.ColorScale.Y,
			&slot.ColorScale.Z,
		)
	}

	return nil
}

func LoadVSlots(p *game.Packet, numVSlots int32) ([]*VSlot, error) {
	leftToRead := numVSlots

	vSlots := make([]*VSlot, 0)
	prev := make([]int32, numVSlots)

	addSlot := func() *VSlot {
		vslot := VSlot{}
		vslot.Index = int32(len(vSlots))
		vSlots = append(vSlots, &vslot)
		return &vslot
	}

	for leftToRead > 0 {
		changed, _ := GetInt(p)
		if changed < 0 {
			for i := 0; i < int(-1*changed); i++ {
				addSlot()
			}
			leftToRead += changed
		} else {
			prevValue, _ := GetInt(p)
			prev[len(vSlots)] = prevValue
			slot := addSlot()
			LoadVSlot(p, slot, changed)
			leftToRead--
		}
	}

	for i, slot := range vSlots {
		other := prev[i]
		if other >= 0 && int(other) < len(prev) {
			vSlots[other].Next = slot
		}
	}

	return vSlots, nil
}

func Decode(data []byte) (*GameMap, error) {
	p := game.Packet(data)

	gameMap := GameMap{}

	header := FileHeader{}
	err := p.GetRaw(&header)
	if err != nil {
		return nil, err
	}

	newFooter := NewFooter{}
	oldFooter := OldFooter{}
	if header.Version <= 28 {
		p.Skip(28) // 7 * 4, like in worldio.cpp
		err = p.GetRaw(&oldFooter)
		if err != nil {
			return nil, err
		}

		newFooter.BlendMap = int32(oldFooter.BlendMap)
		newFooter.NumVars = 0
		newFooter.NumVSlots = 0
	} else {
		q := p
		p.GetRaw(&newFooter)

		if header.Version <= 29 {
			newFooter.NumVSlots = 0
		}

		// v29 had one fewer field
		if header.Version == 29 {
			p = q[len(q)-len(p)-4:]
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

	//log.Debug().Msgf("header %+v", mapHeader)

	log.Printf("Version %d", header.Version)
	gameMap.Vars = make(map[string]Variable)

	for i := 0; i < int(newFooter.NumVars); i++ {
		_type, _ := p.GetByte()
		name, _ := GetString(&p)

		switch VariableType(_type) {
		case VariableTypeInt:
			value, _ := GetInt(&p)
			gameMap.Vars[name] = IntVariable(value)
			//log.Printf("%s=%d", name, value)
		case VariableTypeFloat:
			value, _ := GetFloat(&p)
			gameMap.Vars[name] = FloatVariable(value)
			//log.Printf("%s=%f", name, value)
		case VariableTypeString:
			value, _ := GetString(&p)
			gameMap.Vars[name] = StringVariable(value)
			//log.Printf("%s=%s", name, value)
		}
	}

	//log.Info().Msgf("%v", gameMap.Vars)

	gameType := "fps"
	if header.Version >= 16 {
		gameType, _ = GetStringByte(&p)
	}
	//log.Printf("GameType %s", gameType)

	mapHeader.GameType = gameType

	// We just skip extras
	var eif uint16 = 0
	if header.Version >= 16 {
		var extraBytes uint16
		err = p.GetRaw(
			&eif,
			&extraBytes,
		)
		if err != nil {
			return nil, err
		}
		p.Skip(int(extraBytes))
	}

	// Also skip the texture MRU
	if header.Version < 14 {
		p.Skip(256)
	} else {
		numMRUBytes, _ := GetShort(&p)
		p.Skip(int(numMRUBytes * 2))
	}

	entities := make([]Entity, header.NumEnts)

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}
		p.GetRaw(&entity)

		if gameType != "fps" {
			if eif > 0 {
				p.Skip(int(eif))
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

	vSlotData, err := LoadVSlots(&p, newFooter.NumVSlots)
	gameMap.VSlots = vSlotData

	cube, err := LoadChildrenCube(&p, header.Version)
	if err != nil {
		return nil, err
	}

	gameMap.WorldRoot = cube

	return &gameMap, nil
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
