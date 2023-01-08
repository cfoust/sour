package servers

import (
	"sync"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"

	//"github.com/rs/zerolog/log"
)

type EditingState struct {
	state maps.CCube
	mutex sync.Mutex
}

func (e *EditingState) LoadBytes(data []byte) error {
	map_, err := maps.FromGZ(data)
	if err != nil {
		return err
	}

	c := maps.MapToCXX(map_.WorldRoot)
	e.state = c

	return nil
}

func (e *EditingState) LoadMap(path string) error {
	map_, err := maps.FromFile(path)
	if err != nil {
		return err
	}

	c := maps.MapToCXX(map_.WorldRoot)
	e.state = c

	return nil
}

func (e *EditingState) Consume(message game.Message) {
}

func NewEditingState() *EditingState {
	gameMap := maps.NewMap()
	c := maps.MapToCXX(gameMap.WorldRoot)
	return &EditingState{
		state: c,
	}
}
