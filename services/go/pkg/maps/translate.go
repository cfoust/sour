package maps

import (
	"github.com/cfoust/sour/pkg/maps/api"
	E "github.com/cfoust/sour/pkg/maps/api/entities"
)

func (m *GameMap) ToAPI() (*api.Map, error) {
	map_ := api.Map{}
	map_.WorldSize = m.Header.WorldSize
	map_.GameType = m.Header.GameType

	entities := make([]E.Entity, 0)
	for _, entity := range m.Entities {
		attributes := E.Attributes([]int16{
			entity.Attr1,
			entity.Attr2,
			entity.Attr3,
			entity.Attr4,
			entity.Attr5,
		})

		info, err := E.Decode(entity.Type, &attributes)
		if err != nil {
			return nil, err
		}

		entities = append(entities, E.Entity{
			Position: E.Vector{
				X: entity.Position.X,
				Y: entity.Position.Y,
				Z: entity.Position.Z,
			},
			Info: info,
		})
	}

	map_.Entities = entities
	map_.Variables = m.Vars

	return &map_, nil
}
