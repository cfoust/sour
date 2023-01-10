package maps

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps/worldio"

	"github.com/rs/zerolog/log"
)

func MapToGo(parent worldio.Cube) *Cube {
	children := make([]*Cube, 0)
	for i := 0; i < CUBE_FACTOR; i++ {
		cube := Cube{}
		member := worldio.CubeArray_getitem(parent, i)

		if member.GetChildren().Swigcptr() != 0 {
			cube.Children = MapToGo(member.GetChildren()).Children
		}

		if member.GetExt().Swigcptr() != 0 {
			ext := member.GetExt()
			for j := 0; j < 6; j++ {
				surface := worldio.SurfaceInfoArray_getitem(ext.GetSurfaces(), j)
				cube.SurfaceInfo[j].Lmid[0] = worldio.UcharArray_getitem(surface.GetLmid(), 0)
				cube.SurfaceInfo[j].Lmid[1] = worldio.UcharArray_getitem(surface.GetLmid(), 1)
				cube.SurfaceInfo[j].Verts = surface.GetVerts()
				cube.SurfaceInfo[j].NumVerts = surface.GetNumverts()
			}
		}

		// edges
		for j := 0; j < 12; j++ {
			value := worldio.UcharArray_getitem(member.GetEdges(), j)
			cube.Edges[j] = value
		}

		// texture
		for j := 0; j < 6; j++ {
			value := worldio.Uint16Array_getitem(member.GetTexture(), j)
			cube.Texture[j] = value
		}

		cube.Material = member.GetMaterial()
		cube.Merged = member.GetMerged()
		cube.Escaped = member.GetEscaped()
		children = append(children, &cube)
	}

	cube := Cube{
		Children: children,
	}

	return &cube
}

func LoadChildren(p *game.Buffer, size int32, mapVersion int32) (*Cube, error) {
	root := worldio.Loadchildren_buf(
		uintptr(unsafe.Pointer(&(*p)[0])),
		int64(len(*p)),
		int(size),
		int(mapVersion),
	)
	if root.Swigcptr() == 0 {
		return nil, fmt.Errorf("failed to load cubes")
	}
	cube := MapToGo(root)
	worldio.Freeocta(root)
	return cube, nil
}

func LoadVSlot(p *game.Buffer, slot *VSlot, changed int32) error {
	slot.Changed = changed
	if (changed & (1 << VSLOT_SHPARAM)) > 0 {
		numParams, _ := p.GetShort()

		for i := 0; i < int(numParams); i++ {
			param := SlotShaderParam{}
			name, _ := p.GetStringByte()

			// TODO getshaderparamname
			param.Name = name
			for k := 0; k < 4; k++ {
				value, _ := p.GetFloat()
				param.Val[k] = value
			}
			slot.Params = append(slot.Params, param)
		}
	}

	if (changed & (1 << VSLOT_SCALE)) > 0 {
		p.Get(&slot.Scale)
	}

	if (changed & (1 << VSLOT_ROTATION)) > 0 {
		p.Get(&slot.Rotation)
	}

	if (changed & (1 << VSLOT_OFFSET)) > 0 {
		p.Get(
			&slot.Offset.X,
			&slot.Offset.Y,
		)
	}

	if (changed & (1 << VSLOT_SCROLL)) > 0 {
		p.Get(
			&slot.Scroll.X,
			&slot.Scroll.Y,
		)
	}

	if (changed & (1 << VSLOT_LAYER)) > 0 {
		p.Get(&slot.Layer)
	}

	if (changed & (1 << VSLOT_ALPHA)) > 0 {
		p.Get(
			&slot.AlphaFront,
			&slot.AlphaBack,
		)
	}

	if (changed & (1 << VSLOT_COLOR)) > 0 {
		p.Get(
			&slot.ColorScale.X,
			&slot.ColorScale.Y,
			&slot.ColorScale.Z,
		)
	}

	return nil
}

func LoadVSlots(p *game.Buffer, numVSlots int32) ([]*VSlot, error) {
	leftToRead := numVSlots

	vSlots := make([]*VSlot, 0)
	prev := make([]int32, numVSlots)

	addSlot := func() *VSlot {
		vslot := NewVSlot(nil, int32(len(vSlots)))
		vSlots = append(vSlots, vslot)
		return vslot
	}

	for leftToRead > 0 {
		changed, _ := p.GetInt()
		if changed < 0 {
			for i := 0; i < int(-1*changed); i++ {
				addSlot()
			}
			leftToRead += changed
		} else {
			prevValue, _ := p.GetInt()
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

func LoadPartial(p *game.Buffer, header Header) (worldio.MapState, error) {
	state := worldio.Partial_load_world(
		uintptr(unsafe.Pointer(&(*p)[0])),
		int64(len(*p)),
		int(header.NumVSlots),
		int(header.WorldSize),
		int(header.Version),
		int(header.LightMaps),
		int(header.NumPVs),
		int(header.BlendMap),
	)
	if state.Swigcptr() == 0 {
		return nil, fmt.Errorf("failed to load cubes")
	}
	return state, nil
}

func Decode(data []byte) (*GameMap, error) {
	p := game.Buffer(data)

	gameMap := GameMap{}

	header := FileHeader{}
	err := p.Get(&header)
	if err != nil {
		return nil, err
	}

	newFooter := NewFooter{}
	oldFooter := OldFooter{}
	if header.Version <= 28 {
		// Reset and read again
		p = game.Buffer(data)
		p.Skip(28) // 7 * 4, like in worldio.cpp
		err = p.Get(&oldFooter)
		if err != nil {
			return nil, err
		}

		newFooter.BlendMap = int32(oldFooter.BlendMap)
		newFooter.NumVars = 0
		newFooter.NumVSlots = 0
	} else {
		q := p
		p.Get(&newFooter)

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
	mapHeader.NumPVs = header.NumPVs
	mapHeader.BlendMap = newFooter.BlendMap
	mapHeader.NumVars = newFooter.NumVars
	mapHeader.NumVSlots = newFooter.NumVSlots

	gameMap.Header = mapHeader

	log.Debug().Msgf("Version %d", header.Version)
	gameMap.Vars = make(map[string]game.Variable)

	for i := 0; i < int(newFooter.NumVars); i++ {
		_type, _ := p.GetByte()
		name, _ := p.GetString()

		switch game.VariableType(_type) {
		case game.VariableTypeInt:
			value, _ := p.GetInt()
			gameMap.Vars[name] = game.IntVariable(value)
		case game.VariableTypeFloat:
			value, _ := p.GetFloat()
			gameMap.Vars[name] = game.FloatVariable(value)
		case game.VariableTypeString:
			value, _ := p.GetString()
			gameMap.Vars[name] = game.StringVariable(value)
		}
	}

	gameType := "fps"
	if header.Version >= 16 {
		gameType, _ = p.GetStringByte()
	}
	mapHeader.GameType = gameType

	// We just skip extras
	var eif uint16 = 0
	if header.Version >= 16 {
		var extraBytes uint16
		err = p.Get(
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
		numMRUBytes, _ := p.GetShort()
		p.Skip(int(numMRUBytes * 2))
	}

	entities := make([]Entity, header.NumEnts)

	// Load entities
	for i := 0; i < int(header.NumEnts); i++ {
		entity := Entity{}
		p.Get(&entity)

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

	state, err := LoadPartial(&p, gameMap.Header)
	log.Debug().Msgf("state %+v", state)
	os.Exit(0)

	//vSlotData, err := LoadVSlots(&p, newFooter.NumVSlots)
	//gameMap.VSlots = vSlotData

	//log.Debug().Msgf("Header %+v", header)

	//cube, err := LoadChildren(&p, header.WorldSize, header.Version)
	//if err != nil {
		//return nil, err
	//}

	//gameMap.WorldRoot = cube

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
