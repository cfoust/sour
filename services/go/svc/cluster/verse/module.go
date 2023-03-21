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

func NewVerse(db *gorm.DB) *Verse {
	return &Verse{
		db: db,
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

func (m *Map) getPointer(ctx context.Context) (*state.MapPointer, error) {
	var pointer state.MapPointer
	err := m.db.WithContext(ctx).Where(state.MapPointer{
		Aliasable: state.Aliasable{UUID: m.id},
	}).First(&pointer).Error
	if err != nil {
		return nil, err
	}

	return &pointer, nil
}

func (m *Map) getMap(ctx context.Context) (*state.MapPointer, *state.Map, error) {
	pointer, err := m.getPointer(ctx)
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
	_, map_, err := m.getMap(ctx)
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

func (m *Map) SaveGameMap(ctx context.Context, creator *state.User, gameMap *maps.GameMap) error {
	pointer, err := m.getPointer(ctx)
	if err != nil {
		return err
	}

	mapData, err := gameMap.EncodeOGZ()
	if err != nil {
		return err
	}

	asset, err := m.store.Store(ctx, creator, "ogz", mapData)
	if err != nil {
		return err
	}

	newMap := state.Map{
		OgzID:     asset.ID,
		Creatable: state.NewCreatable(creator),
	}

	_, oldMap, err := m.getMap(ctx)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if err == nil {
		newMap.CfgID = oldMap.CfgID
	}

	db := m.db.WithContext(ctx)

	err = db.Create(&newMap).Error
	if err != nil {
		return err
	}

	pointer.MapID = newMap.ID
	err = db.Save(pointer).Error
	if err != nil {
		return err
	}

	return nil
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
	err = v.db.WithContext(ctx).Create(&pointer).Error
	if err != nil {
		return nil, err
	}

	return &Map{
		id: id,
		entity: entity{
			db:    v.db,
			store: v.store,
			verse: v,
		},
	}, nil
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
}

type Link struct {
	ID          uint8
	Destination string
}

type SpaceConfig struct {
	Alias       string
	Map         string
	Description string
	Links       []Link
}

func (s *UserSpace) GetConfig(ctx context.Context) (*SpaceConfig, error) {
	return &SpaceConfig{
		Alias:       s.Alias,
		Map:         s.Map.Hash,
		Description: s.Description,
		Links:       links,
	}, nil
}

func (v *Verse) NewSpaceID(ctx context.Context) (string, error) {
	for {
		number, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return "", err
		}

		hash := utils.HashString(fmt.Sprintf("%d", number))
		value, err := v.redis.Exists(ctx, fmt.Sprintf(SPACE_KEY, hash)).Result()
		if err != nil {
			return "", err
		}

		if value == 0 {
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

	space := UserSpace{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	err = space.init(ctx, SpaceConfig{
		Map:         map_.GetID(),
		Owner:       creator,
		Description: "",
		Alias:       "",
	})
	if err != nil {
		return nil, err
	}

	return &space, nil
}

func (v *Verse) HaveSpace(ctx context.Context, id string) (bool, error) {
	return v.have(ctx, fmt.Sprintf(SPACE_ID_KEY, id))
}

func (v *Verse) LoadSpace(ctx context.Context, id string) (*UserSpace, error) {
	space := UserSpace{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	exists, err := v.HaveSpace(ctx, id)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("space does not exist")
	}

	return &space, nil
}

// Find a space by a prefix
func (v *Verse) FindSpace(ctx context.Context, needle string) (*UserSpace, error) {
	return nil, nil
}
