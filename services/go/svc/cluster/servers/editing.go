package servers

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/maps/worldio"

	"github.com/rs/zerolog/log"
)

type Edit struct {
	Time    time.Time
	Message game.Message
}

func NewEdit(message game.Message) *Edit {
	return &Edit{
		Time:    time.Now(),
		Message: message,
	}
}

// Sort of like maps.GameMap, but composed of the C++ types.
type MapState struct {
	World worldio.Cube
	Size  int32
	Vars  game.Variables
}

func MapStateFromGameMap(map_ *maps.GameMap) *MapState {
	return &MapState{
		World: maps.MapToCXX(map_.WorldRoot),
		Vars:  map_.Vars,
		Size:  map_.Header.WorldSize,
	}
}

func MapStateFromBytes(data []byte) (*MapState, error) {
	map_, err := maps.FromGZ(data)
	if err != nil {
		return nil, err
	}

	return MapStateFromGameMap(map_), nil
}

func MapStateFromFile(path string) (*MapState, error) {
	map_, err := maps.FromFile(path)
	if err != nil {
		return nil, err
	}

	return MapStateFromGameMap(map_), nil
}

func (m *MapState) Apply(edits []*Edit) error {
	buffer := make([]byte, 0)
	for _, edit := range edits {
		if edit.Message.Type() != game.N_EDITVAR {
			buffer = append(buffer, edit.Message.Data()...)
			continue
		}

		varEdit := edit.Message.Contents().(*game.EditVar)
		err := m.Vars.Set(varEdit.Key, varEdit.Value)
		if err != nil {
			log.Warn().Err(err).Msgf("setting map variable failed %s=%+v", varEdit.Key, varEdit.Value)
		}
	}

	if len(buffer) == 0 {
		return nil
	}

	result := worldio.Apply_messages(
		m.World,
		int(m.Size),
		uintptr(unsafe.Pointer(&(buffer)[0])),
		int64(len(buffer)),
	)
	if result.Swigcptr() == 0 {
		return fmt.Errorf("applying changes failed")
	}

	m.World = result

	map_ := maps.NewMap()
	map_.WorldRoot = maps.MapToGo(m.World)
	map_.Vars = m.Vars
	err := map_.ToFile("../test.ogz")
	if err != nil {
		log.Warn().Err(err).Msgf("failed to save map")
	}
	return nil
}

type EditingState struct {
	Edits    []*Edit
	MapState *MapState
	mutex    sync.Mutex
}

// Apply all of the edits to the map.
func (e *EditingState) Checkpoint() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	err := e.MapState.Apply(e.Edits)
	e.Edits = make([]*Edit, 0)
	return err
}

func (e *EditingState) Process(message game.Message) {
	e.mutex.Lock()
	e.Edits = append(e.Edits, NewEdit(message))
	e.mutex.Unlock()
}

func (e *EditingState) LoadMap(path string) error {
	state, err := MapStateFromFile(path)
	if err != nil {
		return err
	}

	e.mutex.Lock()
	e.Edits = make([]*Edit, 0)
	e.MapState = state
	e.mutex.Unlock()
	return nil
}

func NewEditingState() *EditingState {
	return &EditingState{
		Edits:    make([]*Edit, 0),
		MapState: MapStateFromGameMap(maps.NewMap()),
	}
}

func (e *EditingState) PollEdits(ctx context.Context) {
	tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			e.Checkpoint()
			continue
		}
	}
}
