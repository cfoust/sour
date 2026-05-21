package maps

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"

	gIO "github.com/cfoust/sour/pkg/game/io"
	V "github.com/cfoust/sour/pkg/game/variables"

	"github.com/rs/zerolog/log"
)

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
			log.Debug().Msgf("Entity outside of world")
			log.Debug().Msgf("entity type %d", entity.Type)
			log.Debug().Msgf("entity pos x=%f,y=%f,z=%f", entity.Position.X, entity.Position.Y, entity.Position.Z)
		}

		if header.Version <= 14 && entity.Type == ET_MAPMODEL {
			entity.Position.Z += float32(entity.Attr3)
			entity.Attr3 = 0

			if entity.Attr4 > 0 {
				log.Debug().Msgf("warning: mapmodel ent (index %d) uses texture slot %d", i, entity.Attr4)
			}

			entity.Attr4 = 0
		}

		entities[i] = entity
	}

	gameMap.Entities = entities

	if skipCubes {
		return &gameMap, nil
	}

	// Load world data using pure Go loader
	remaining := []byte(p)
	state, err := LoadWorld(
		remaining,
		int(mapHeader.NumVSlots),
		int(mapHeader.WorldSize),
		mapHeader.Version,
		int(mapHeader.LightMaps),
		int(mapHeader.NumPVs),
		int(mapHeader.BlendMap),
	)
	if err != nil {
		return nil, err
	}

	gameMap.VSlots = state.VSlots
	gameMap.WorldRoot = state.Root
	gameMap.World = state

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
	if err != nil && err != gzip.ErrChecksum {
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
