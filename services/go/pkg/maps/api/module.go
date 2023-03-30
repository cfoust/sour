package api

import (
	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/maps/api/entities"
)

type Map struct {
	WorldSize int32
	GameType  string
	Entities  []entities.Entity
}

type Typable interface {
	String() string
	FromString(string)
}

var _ Typable = (*C.EntityType)(nil)
var _ Typable = (*entities.ParticleType)(nil)
