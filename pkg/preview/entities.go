package preview

import (
	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/maps"
)

var entityTypeMap = map[C.EntityType]PreviewEntityType{
	C.EntityTypePlayerStart:  EntityPlayerStart,
	C.EntityTypeFlag:         EntityFlag,
	C.EntityTypeBase:         EntityBase,
	C.EntityTypeTeleport:     EntityTeleport,
	C.EntityTypeJumpPad:      EntityJumpPad,
	C.EntityTypeHealth:       EntityHealth,
	C.EntityTypeBoost:        EntityHealth,
	C.EntityTypeGreenArmour:  EntityArmour,
	C.EntityTypeYellowArmour: EntityArmour,
	C.EntityTypeShells:       EntityAmmo,
	C.EntityTypeBullets:      EntityAmmo,
	C.EntityTypeRockets:      EntityAmmo,
	C.EntityTypeRounds:       EntityAmmo,
	C.EntityTypeGrenades:     EntityAmmo,
	C.EntityTypeCartridges:   EntityAmmo,
	C.EntityTypeQuad:         EntityQuad,
}

// ExtractEntities filters map entities to gameplay-relevant types and
// quantizes their positions to the preview grid.
func ExtractEntities(entities []maps.Entity, worldSize int32, gridSize uint16) []PreviewEntity {
	var result []PreviewEntity
	for _, e := range entities {
		pType, ok := entityTypeMap[e.Type]
		if !ok {
			continue
		}

		// Quantize position from world space to grid space.
		gx := uint16(clampInt(int(e.Position.X*float32(gridSize)/float32(worldSize)), 0, int(gridSize)-1))
		gy := uint16(clampInt(int(e.Position.Y*float32(gridSize)/float32(worldSize)), 0, int(gridSize)-1))
		gz := uint16(clampInt(int(e.Position.Z*float32(gridSize)/float32(worldSize)), 0, int(gridSize)-1))

		result = append(result, PreviewEntity{
			X:    gx,
			Y:    gy,
			Z:    gz,
			Type: pType,
		})
	}
	return result
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
