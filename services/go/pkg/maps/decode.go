package maps

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"unsafe"

	gIO "github.com/cfoust/sour/pkg/game/io"
	V "github.com/cfoust/sour/pkg/game/variables"
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

func VSlotsToGo(state worldio.MapState) []*VSlot {
	vslots := make([]*VSlot, 0)

	refs := make(map[uintptr]*VSlot)

	for i := 0; i < worldio.Getnumvslots(state); i++ {
		vslot := VSlot{}
		slot := worldio.Getvslotindex(state, i)
		vslot.Index = int32(slot.GetIndex())
		vslot.Changed = int32(slot.GetChanged())
		vslot.Layer = int32(slot.GetLayer())
		vslot.Linked = slot.GetLinked()
		vslot.Scale = float32(slot.GetScale())
		vslot.Rotation = int32(slot.GetRotation())
		vslot.AlphaFront = float32(slot.GetAlphafront())
		vslot.AlphaBack = float32(slot.GetAlphaback())

		// TODO Params, Offset, Scroll, ColorScale, GlowColor

		refs[slot.Swigcptr()] = &vslot

		vslots = append(vslots, &vslot)
	}

	// Second pass, link up next pointers
	for i := 0; i < worldio.Getnumvslots(state); i++ {
		vslot := vslots[i]
		slot := worldio.Getvslotindex(state, i)

		ptr := slot.GetNext().Swigcptr()
		if ptr == 0 {
			continue
		}

		next, ok := refs[ptr]
		if !ok || next == nil {
			continue
		}

		vslot.Next = next
	}

	return vslots
}

func LoadPartial(p *gIO.Buffer, header Header) (worldio.MapState, error) {
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

func decode(data []byte, skipCubes bool) (*GameMap, error) {
	p := gIO.Buffer(data)

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
		p = gIO.Buffer(data)
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

	//log.Debug().Msgf("Version %d", header.Version)
	gameMap.Vars = make(map[string]V.Variable)

	for i := 0; i < int(newFooter.NumVars); i++ {
		_type, _ := p.GetByte()
		name, _ := p.GetString()

		switch V.VariableType(_type) {
		case V.VariableTypeInt:
			value, _ := p.GetInt()
			gameMap.Vars[name] = V.IntVariable(value)
		case V.VariableTypeFloat:
			value, _ := p.GetFloat()
			gameMap.Vars[name] = V.FloatVariable(value)
		case V.VariableTypeString:
			value, _ := p.GetString()
			gameMap.Vars[name] = V.StringVariable(value)
		}
	}

	gameType := "fps"
	if header.Version >= 16 {
		gameType, _ = p.GetStringByte()
	}
	mapHeader.GameType = gameType

	gameMap.Header = mapHeader

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

	if skipCubes {
		return &gameMap, nil
	}

	state, err := LoadPartial(&p, gameMap.Header)
	if err != nil {
		return nil, err
	}

	gameMap.VSlots = VSlotsToGo(state)
	// TODO wow, guess we don't need this anymore
	//gameMap.WorldRoot = MapToGo(state.GetRoot())
	gameMap.C = state

	return &gameMap, nil
}

func Decode(data []byte) (*GameMap, error) {
	return decode(data, false)
}

func DecodeBasics(data []byte) (*GameMap, error) {
	return decode(data, true)
}

func fromGZ(data []byte, skipCubes bool) (*GameMap, error) {
	buffer := bytes.NewReader(data)
	gz, err := gzip.NewReader(buffer)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	rawBytes, err := io.ReadAll(gz)
	if err == gzip.ErrChecksum {
		log.Warn().Msg("Map file had invalid checksum")
	} else if err != nil {
		return nil, err
	}

	return decode(rawBytes, skipCubes)
}

func FromGZ(data []byte) (*GameMap, error) {
	return fromGZ(data, false)
}

func BasicsFromGZ(data []byte) (*GameMap, error) {
	return fromGZ(data, true)
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
