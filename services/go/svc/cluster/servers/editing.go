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
	"github.com/cfoust/sour/svc/cluster/verse"

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

type EditingState struct {
	Edits   []*Edit
	GameMap *maps.GameMap
	Map     *verse.Map
	Space   *verse.Space
	mutex   sync.Mutex
	verse   *verse.Verse
}

const (
	MAP_EXPIRE = time.Hour * 24
)

// Apply all of the edits to the map.
func (e *EditingState) Checkpoint(ctx context.Context) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(e.Edits) == 0 {
		return nil
	}

	err := e.Apply(e.Edits)
	e.Edits = make([]*Edit, 0)

	creator, err := e.Map.GetCreator(ctx)
	if err != nil {
	    return err
	}

	newMap, err := e.verse.SaveGameMap(ctx, creator, e.GameMap)
	if err != nil {
	    return err
	}

	// Expire the old map after a day
	e.Map.Expire(ctx, MAP_EXPIRE)

	e.Map = newMap

	if e.Space != nil {
		err = e.Space.SetMapID(ctx, newMap.GetID())
		if err != nil {
		    return err
		}
		log.Info().Msgf("saved map %s for space %s", newMap.GetID(), e.Space.GetID())
	} else {
		log.Info().Msgf("saved map %s", newMap.GetID())
	}

	return err
}

func (e *EditingState) Process(message game.Message) {
	e.mutex.Lock()
	e.Edits = append(e.Edits, NewEdit(message))
	e.mutex.Unlock()
}

func (e *EditingState) LoadMap(map_ *maps.GameMap) error {
	e.mutex.Lock()
	e.Edits = make([]*Edit, 0)
	e.GameMap = map_
	e.mutex.Unlock()

	return nil
}

func NewEntity(entities *[]maps.Entity, index int, entity maps.Entity) *maps.Entity {
	for len(*entities) < index {
		*entities = append(*entities, maps.Entity{})
	}

	*entities = append(*entities, maps.Entity{})
	if index > 0 && index < len(*entities) {
		(*entities)[index] = entity
	} else {
		*entities = append(*entities, entity)
	}

	return &((*entities)[index])
}

func EditEntity(entities *[]maps.Entity, edit *game.EditEnt) {
	i := edit.Index

	if i < 0 || i >= game.MAXENTS {
		return
	}

	if len(*entities) <= i {
		entity := NewEntity(entities, i, maps.Entity{
			Position: maps.Vector{
				X: edit.X,
				Y: edit.Y,
				Z: edit.Z,
			},
			Attr1: int16(edit.Attr1),
			Attr2: int16(edit.Attr2),
			Attr3: int16(edit.Attr3),
			Attr4: int16(edit.Attr4),
			Attr5: int16(edit.Attr5),
			Type:  game.EntityType(edit.Type),
		})
		if entity == nil {
			return
		}
	} else {
		entity := &(*entities)[i]
		entity.Type = game.EntityType(edit.Type)
		entity.Position = maps.Vector{
			X: edit.X,
			Y: edit.Y,
			Z: edit.Z,
		}
		entity.Attr1 = int16(edit.Attr1)
		entity.Attr2 = int16(edit.Attr2)
		entity.Attr3 = int16(edit.Attr3)
		entity.Attr4 = int16(edit.Attr4)
		entity.Attr5 = int16(edit.Attr5)
	}
}

func (e *EditingState) Apply(edits []*Edit) error {
	buffer := make([]byte, 0)
	for _, edit := range edits {
		if edit.Message.Type() == game.N_EDITVAR {
			varEdit := edit.Message.Contents().(*game.EditVar)
			err := e.GameMap.Vars.Set(varEdit.Key, varEdit.Value)
			if err != nil {
				log.Warn().Err(err).Msgf("setting map variable failed %s=%+v", varEdit.Key, varEdit.Value)
			}
			continue
		}

		if edit.Message.Type() == game.N_EDITENT {
			entEdit := edit.Message.Contents().(*game.EditEnt)
			EditEntity(&e.GameMap.Entities, entEdit)
			continue
		}

		buffer = append(buffer, edit.Message.Data()...)
	}

	if len(buffer) == 0 {
		return nil
	}

	result := worldio.Apply_messages(
		e.GameMap.C,
		int(e.GameMap.Header.WorldSize),
		uintptr(unsafe.Pointer(&(buffer)[0])),
		int64(len(buffer)),
	)
	if !result {
		return fmt.Errorf("applying changes failed")
	}

	err := e.GameMap.ToFile("../test.ogz")
	if err != nil {
		log.Warn().Err(err).Msgf("failed to save map")
	}
	return nil
}

func NewEditingState(verse *verse.Verse, space *verse.Space, map_ *verse.Map) *EditingState {
	return &EditingState{
		Edits: make([]*Edit, 0),
		verse: verse,
		Map: map_,
		Space: space,
	}
}

func (e *EditingState) PollEdits(ctx context.Context) {
	tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			e.Checkpoint(ctx)
			continue
		}
	}
}
