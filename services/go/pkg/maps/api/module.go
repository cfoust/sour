package api

import (
	C "github.com/cfoust/sour/pkg/game/constants"
	V "github.com/cfoust/sour/pkg/game/variables"
	"github.com/cfoust/sour/pkg/maps/api/entities"
)

type Map struct {
	WorldSize int32             `json:"worldSize"`
	GameType  string            `json:"gameType"`
	Entities  []entities.Entity `json:"entities"`
	Variables V.Variables       `json:"variables"`
}

type Typable interface {
	String() string
	FromString(string)
}

var _ Typable = (*C.EntityType)(nil)
var _ Typable = (*entities.ParticleType)(nil)
