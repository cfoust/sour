package service

import (
	"sync"

	"github.com/cfoust/sour/pkg/config"
)

type ELO struct {
	Rating uint
	Wins   uint
	Draws  uint
	Losses uint
}

func NewELO() *ELO {
	return &ELO{
		Rating: 1200,
	}
}

type ELOState struct {
	Ratings map[string]*ELO
	Mutex   sync.Mutex
}

func NewELOState(duels []config.DuelType) *ELOState {
	state := ELOState{
		Ratings: make(map[string]*ELO),
	}

	for _, type_ := range duels {
		state.Ratings[type_.Name] = NewELO()
	}

	return &state
}
