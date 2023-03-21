package verse

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"time"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/utils"
	"github.com/cfoust/sour/svc/cluster/state"
	"github.com/cfoust/sour/svc/cluster/stores"

	"gorm.io/gorm"
)

type Verse struct {
	store *stores.AssetStorage
	db    *gorm.DB
}

func NewVerse(db *gorm.DB, store *stores.AssetStorage) *Verse {
	return &Verse{
		db:    db,
		store: store,
	}
}

type entity struct {
	db    *gorm.DB
	store *stores.AssetStorage
	verse *Verse
}

type Map struct {
	entity
	id string
}

func (m *Map) GetID() string {
	return m.id
}

func (m *Map) GetPointer(ctx context.Context) (*state.MapPointer, error) {
	var pointer state.MapPointer
	err := m.db.WithContext(ctx).Where(state.MapPointer{
		Aliasable: state.Aliasable{UUID: m.id},
	}).Joins("Creator").First(&pointer).Error
	if err != nil {
		return nil, err
	}

	return &pointer, nil
}

func (m *Map) GetMap(ctx context.Context) (*state.MapPointer, *state.Map, error) {
	pointer, err := m.GetPointer(ctx)
	if err != nil {
		return nil, nil, err
	}

	var map_ state.Map
	query := state.Map{}
	query.ID = pointer.MapID
	err = m.db.WithContext(ctx).
		Where(query).
		Joins("Ogz").
		First(&map_).
		Error
	if err != nil {
		return nil, nil, err
	}

	return pointer, &map_, nil
}

func (m *Map) LoadMapData(ctx context.Context) ([]byte, error) {
	_, map_, err := m.GetMap(ctx)
	if err != nil {
		return nil, err
	}

	return m.store.Get(ctx, map_.Ogz)
}

func (m *Map) LoadGameMap(ctx context.Context) (*maps.GameMap, error) {
	data, err := m.LoadMapData(ctx)
	if err != nil {
		return nil, err
	}

	map_, err := maps.FromGZ(data)
	if err := maps.LoadDefaultSlots(map_); err != nil {
		return nil, err
	}

	return map_, nil
}

func (m *Map) SaveGameMap(ctx context.Context, creator *state.User, gameMap *maps.GameMap) (*state.Map, error) {
	pointer, err := m.GetPointer(ctx)
	if err != nil {
		return nil, err
	}

	mapData, err := gameMap.EncodeOGZ()
	if err != nil {
		return nil, err
	}

	asset, err := m.store.Store(ctx, creator, "ogz", mapData)
	if err != nil {
		return nil, err
	}

	newMap := state.Map{
		OgzID:     asset.ID,
		Creatable: state.NewCreatable(creator),
		UUID:      utils.Hash(mapData),
	}

	_, oldMap, err := m.GetMap(ctx)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err == nil {
		newMap.CfgID = oldMap.CfgID
	}

	db := m.db.WithContext(ctx)

	err = db.Create(&newMap).Error
	if err != nil {
		return nil, err
	}

	pointer.MapID = newMap.ID
	err = db.Save(pointer).Error
	if err != nil {
		return nil, err
	}

	return &newMap, nil
}

func (v *Verse) NewMap(ctx context.Context, creator *state.User) (*Map, error) {
	gameMap, err := maps.NewMap()
	if err != nil {
		return nil, err
	}

	defer gameMap.Destroy()

	mapData, err := gameMap.EncodeOGZ()
	if err != nil {
		return nil, err
	}

	id := utils.HashString(fmt.Sprintf("%s%s", time.Now(), utils.Hash(mapData)))
	pointer := state.MapPointer{
		Creatable: state.NewCreatable(creator),
	}
	pointer.UUID = id
	pointer.Alias = id
	err = v.db.WithContext(ctx).Create(&pointer).Error
	if err != nil {
		return nil, err
	}

	map_, err := v.GetMap(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = map_.SaveGameMap(ctx, creator, gameMap)
	if err != nil {
		return nil, err
	}

	return map_, nil
}

func (v *Verse) GetMap(ctx context.Context, id string) (*Map, error) {
	var pointer state.MapPointer
	query := state.MapPointer{}
	query.UUID = id
	err := v.db.WithContext(ctx).Where(query).First(&pointer).Error

	if err != nil {
		return nil, err
	}

	map_ := Map{
		id: pointer.UUID,
		entity: entity{
			db:    v.db,
			store: v.store,
			verse: v,
		},
	}

	return &map_, nil
}

func (v *Verse) HaveMap(ctx context.Context, id string) (bool, error) {
	_, err := v.GetMap(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

var (
	SPACE_ALIAS_REGEX = regexp.MustCompile(`^[a-z0-9-_.:]+$`)
)

func IsValidAlias(alias string) bool {
	return SPACE_ALIAS_REGEX.MatchString(alias)
}

type UserSpace struct {
	entity
	*state.Space
	id string
}

func (u *UserSpace) GetID() string {
	return u.id
}

type Link struct {
	Teleport    uint8
	Teledest    uint8
	Destination string
}

type SpaceConfig struct {
	Alias       string
	Map         string
	Description string
	Links       []Link
}

func (s *UserSpace) GetSpace(ctx context.Context) (*state.Space, error) {
	var space state.Space
	query := state.Space{}
	query.UUID = s.id
	err := s.db.WithContext(ctx).
		Where(query).
		Preload("MapPointer").
		Preload("Links").
		First(&space).Error
	if err != nil {
		return nil, err
	}

	return &space, nil
}

func (s *UserSpace) SetAlias(ctx context.Context, alias string) error {
	space, err := s.GetSpace(ctx)
	if err != nil {
		return err
	}

	space.Alias = alias
	return s.db.WithContext(ctx).Save(&space).Error
}

func (s *UserSpace) SetDescription(ctx context.Context, description string) error {
	space, err := s.GetSpace(ctx)
	if err != nil {
		return err
	}

	space.Description = description
	return s.db.WithContext(ctx).Save(&space).Error
}

func (s *UserSpace) GetMap(ctx context.Context) (*Map, error) {
	space, err := s.GetSpace(ctx)
	if err != nil {
		return nil, err
	}

	return s.verse.GetMap(ctx, space.MapPointer.UUID)
}

func (s *UserSpace) GetConfig(ctx context.Context) (*SpaceConfig, error) {
	space, err := s.GetSpace(ctx)
	if err != nil {
		return nil, err
	}

	links := make([]Link, 0)
	for _, link := range space.Links {
		var destination state.Space
		query := state.Space{}
		query.ID = link.DestinationID
		err := s.db.WithContext(ctx).Where(query).First(&destination).Error
		if err != nil {
			return nil, err
		}
		links = append(links, Link{
			Destination: destination.UUID,
			Teleport:    uint8(link.Teleport),
			Teledest:    uint8(link.Teledest),
		})
	}

	return &SpaceConfig{
		Alias:       space.Alias,
		Map:         space.MapPointer.UUID,
		Description: space.Description,
		Links:       links,
	}, nil
}

func (v *Verse) GetSpace(ctx context.Context, id string) (*UserSpace, error) {
	var space state.Space
	query := state.Space{}
	query.UUID = id
	err := v.db.WithContext(ctx).Where(query).First(&space).Error
	if err != nil {
		return nil, err
	}

	return &UserSpace{
		id: id,
		entity: entity{
			db:    v.db,
			store: v.store,
			verse: v,
		},
	}, nil
}

func (v *Verse) HaveSpace(ctx context.Context, id string) (bool, error) {
	_, err := v.GetSpace(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (v *Verse) NewSpaceID(ctx context.Context) (string, error) {
	for {
		number, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return "", err
		}

		hash := utils.HashString(fmt.Sprintf("%d", number))
		exists, err := v.HaveSpace(ctx, hash)
		if err != nil {
			return "", err
		}

		if !exists {
			return hash, nil
		}
	}
}

func (v *Verse) NewSpace(ctx context.Context, creator *state.User) (*UserSpace, error) {
	id, err := v.NewSpaceID(ctx)
	if err != nil {
		return nil, err
	}

	map_, err := v.NewMap(ctx, creator)
	if err != nil {
		return nil, err
	}

	pointer, err := map_.GetPointer(ctx)
	if err != nil {
		return nil, err
	}

	space := state.Space{
		Creatable: state.NewCreatable(creator),
		Aliasable: state.Aliasable{
			UUID:  id,
			Alias: id,
		},
		Description:  "",
		OwnerID:      creator.ID,
		MapPointerID: pointer.ID,
	}

	err = v.db.WithContext(ctx).Create(&space).Error
	if err != nil {
		return nil, err
	}

	return v.GetSpace(ctx, id)
}

// Find a map by a prefix
func (v *Verse) FindMap(ctx context.Context, needle string) (*Map, error) {
	var map_ state.MapPointer
	err := v.db.WithContext(ctx).
		Where("uuid LIKE ?", needle+"%").
		First(&map_).Error
	if err != nil {
		return nil, err
	}

	return v.GetMap(ctx, map_.UUID)
}

// Find a space by a prefix
func (v *Verse) FindSpace(ctx context.Context, needle string) (*UserSpace, error) {
	var space state.Space
	err := v.db.WithContext(ctx).
		Where("uuid LIKE ?", needle+"%").
		First(&space).Error
	if err != nil {
		return nil, err
	}

	return v.GetSpace(ctx, space.UUID)
}
