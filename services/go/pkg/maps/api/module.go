package api

import (
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
