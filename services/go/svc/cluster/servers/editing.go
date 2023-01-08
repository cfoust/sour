package servers

import (
	"sync"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/maps/worldio"

	"github.com/rs/zerolog/log"
)

type EditingState struct {
	state worldio.Cube
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
	p := message.Data()
	result := worldio.Apply_messages(
		e.state,
		1024,
		uintptr(unsafe.Pointer(&(p)[0])),
		int64(len(p)),
	)
	if result.Swigcptr() == 0 {
		log.Error().Msg("applying changes failed")
		return
	}

	e.state = result

	map_ := maps.NewMap()
	map_.WorldRoot = maps.MapToGo(e.state)
	map_.ToFile("../test.ogz")
}

func NewEditingState() *EditingState {
	gameMap := maps.NewMap()
	c := maps.MapToCXX(gameMap.WorldRoot)
	return &EditingState{
		state: c,
	}
}
