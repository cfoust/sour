package verse

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"time"

	"github.com/cfoust/sour/pkg/maps"
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

func (m *Map) getMap(ctx context.Context) (*state.Map, error) {
	pointer, err := m.getPointer(ctx)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return &map_, nil
}

func (m *Map) LoadMapData(ctx context.Context) ([]byte, error) {
	map_, err := m.getMap(ctx)
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

func (v *Verse) NewMap(ctx context.Context, creator *state.User) (*Map, error) {
	map_, err := maps.NewMap()
	if err != nil {
		return nil, err
	}

	defer map_.Destroy()

	return v.SaveGameMap(ctx, creator, map_)
}

func (v *Verse) HaveMap(ctx context.Context, id string) (bool, error) {
	var pointer state.MapPointer
	query := state.MapPointer{}
	query.UUID = id
	err := v.db.WithContext(ctx).Where(query).First(&pointer).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (v *Verse) GetMap(ctx context.Context, id string) (*Map, error) {
	map_ := Map{
		id: id,
		entity: entity{
			db:    v.db,
			store: v.store,
			verse: v,
		},
	}

	return &map_, nil
}

func (v *Verse) SaveGameMap(ctx context.Context, creator *state.User, gameMap *maps.GameMap) (*Map, error) {
	mapData, err := gameMap.EncodeOGZ()
	if err != nil {
		return nil, err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(mapData))

	// No point in setting this if it already is there
	if exists, _ := v.HaveMap(ctx, hash); exists {
		return v.GetMap(ctx, hash)
	}

	map_ := Map{
		id: hash,
		entity: entity{
			db:    v.db,
			store: v.store,
			verse: v,
		},
	}

	err = v.redis.Set(ctx, map_.dataKey(), mapData, 0).Err()
	if err != nil {
		return nil, err
	}

	err = map_.save(ctx, mapMeta{
		Creator: creator,
		Created: time.Now(),
	})
	if err != nil {
		return nil, err
	}

	if creator == "" {
		err = map_.Expire(ctx, time.Hour*24)
		if err != nil {
			return nil, err
		}
	}

	return &map_, nil
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

		bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
		hash := fmt.Sprintf("%x", bytes)

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
