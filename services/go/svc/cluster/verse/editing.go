package verse

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
	Sender  int32
	Message game.Message
}

func NewEdit(sender int32, message game.Message) *Edit {
	return &Edit{
		Time:    time.Now(),
		Sender:  sender,
		Message: message,
	}
}

type EditingState struct {
	Clipboards map[int32]worldio.Editinfo
	Edits      []*Edit
	GameMap    *maps.GameMap
	Map        *Map
	Space      *UserSpace
	OpenEdit   bool

	mutex sync.Mutex
	verse *Verse
}

const (
	MAP_EXPIRE = time.Hour * 24
)

func (e *EditingState) IsOpenEdit() bool {
	e.mutex.Lock()
	val := e.OpenEdit
	e.mutex.Unlock()
	return val
}

func (e *EditingState) SetOpenEdit(val bool) {
	e.mutex.Lock()
	e.OpenEdit = val
	e.mutex.Unlock()
}

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

	if e.Space != nil {
		creator, err = e.Space.GetOwner(ctx)
		if err != nil {
			return err
		}
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

func (e *EditingState) Process(sender int32, message game.Message) {
	e.mutex.Lock()
	e.Edits = append(e.Edits, NewEdit(sender, message))
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

		if edit.Message.Type() == game.N_NEWMAP {
			e.GameMap.Entities = make([]maps.Entity, 0)
		}

		if edit.Message.Type() == game.N_COPY {
			data := edit.Message.Data()
			worldio.M.Lock()
			info := worldio.Store_copy(
				e.GameMap.C,
				uintptr(unsafe.Pointer(&(data)[0])),
				int64(len(data)),
			)
			worldio.M.Unlock()
			if info.Swigcptr() == 0 {
				log.Warn().Msg("failed to store copy")
				continue
			}

			e.Clipboards[edit.Sender] = info
			continue
		}

		if edit.Message.Type() == game.N_PASTE {
			data := edit.Message.Data()

			info, ok := e.Clipboards[edit.Sender]
			if !ok {
				log.Warn().Msgf("client %d had nothing in clipboard")
				continue
			}

			worldio.M.Lock()
			worldio.Apply_paste(
				e.GameMap.C,
				info,
				uintptr(unsafe.Pointer(&(data)[0])),
				int64(len(data)),
			)
			worldio.M.Unlock()
			continue
		}

		buffer = append(buffer, edit.Message.Data()...)
	}

	if len(buffer) == 0 {
		return nil
	}

	worldio.M.Lock()
	result := worldio.Apply_messages(
		e.GameMap.C,
		int(e.GameMap.Header.WorldSize),
		uintptr(unsafe.Pointer(&(buffer)[0])),
		int64(len(buffer)),
	)
	worldio.M.Unlock()
	if !result {
		return fmt.Errorf("applying changes failed")
	}

	return nil
}

func NewEditingState(verse *Verse, space *UserSpace, map_ *Map) *EditingState {
	return &EditingState{
		OpenEdit:   false,
		Edits:      make([]*Edit, 0),
		Clipboards: make(map[int32]worldio.Editinfo),
		verse:      verse,
		Map:        map_,
		Space:      space,
	}
}

func (e *EditingState) Destroy() {
	e.GameMap.Destroy()

	for _, info := range e.Clipboards {
		worldio.Free_edit(info)
	}
}

func (e *EditingState) SavePeriodically(ctx context.Context) {
	tick := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			err := e.Checkpoint(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("failed to checkpoint map")
			}
			continue
		}
	}
}
