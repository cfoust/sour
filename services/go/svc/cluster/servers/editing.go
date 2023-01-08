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

	log.Info().Msg("loadmap dump")
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		member := worldio.Getcubeindex(e.state, i)
		worldio.Dumpc(member)
	}
	log.Info().Msgf("addr %x", e.state.Swigcptr())

	return nil
}

func (e *EditingState) Consume(message game.Message) {
	log.Info().Msg("consume dump")
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		member := worldio.Getcubeindex(e.state, i)
		worldio.Dumpc(member)
	}
	log.Info().Msgf("addr %x", e.state.Swigcptr())
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
	log.Info().Msg("after consume dump")
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		member := worldio.Getcubeindex(e.state, i)
		worldio.Dumpc(member)
	}
	log.Info().Msgf("addr %x", e.state.Swigcptr())
}

func NewEditingState() *EditingState {
	gameMap := maps.NewMap()
	c := maps.MapToCXX(gameMap.WorldRoot)
	return &EditingState{
		state: c,
	}
}
