package api

import (
	C "github.com/cfoust/sour/pkg/game/constants"
	V "github.com/cfoust/sour/pkg/game/variables"
	"github.com/cfoust/sour/pkg/maps/api/entities"
)

type Map struct {
	WorldSize int32
	GameType  string
	Entities  []entities.Entity
	Vars      V.Variables
}

type Typable interface {
	String() string
	FromString(string)
}

var _ Typable = (*C.EntityType)(nil)
var _ Typable = (*entities.ParticleType)(nil)
