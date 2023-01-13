package verse

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/go-redis/redis/v9"
)

const (
	PREFIX       = "verse-"
	MAP_PREFIX   = PREFIX + "map-"
	MAP_META_KEY = MAP_PREFIX + "meta-%s"
	MAP_DATA_KEY = MAP_PREFIX + "data-%s"
	SPACE_KEY    = PREFIX + "space-%s"
	USER_KEY     = PREFIX + "user-%s"
)

func loadJSON(ctx context.Context, client *redis.Client, key string, v any) error {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

func saveJSON(ctx context.Context, client *redis.Client, key string, v any) error {
	str, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return client.Set(
		ctx,
		key,
		str,
		0,
	).Err()
}

type Verse struct {
	redis *redis.Client
}

func NewVerse(redis *redis.Client) *Verse {
	return &Verse{
		redis: redis,
	}
}

func (v *Verse) have(ctx context.Context, key string) (bool, error) {
	value, err := v.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return value == 1, nil
}

type entity struct {
	redis *redis.Client
	verse *Verse
}

type Map struct {
	entity
	id string
}

func (m *Map) GetID() string {
	return m.id
}

func (m *Map) dataKey() string {
	return fmt.Sprintf(MAP_DATA_KEY, m.id)
}

func (m *Map) metaKey() string {
	return fmt.Sprintf(MAP_META_KEY, m.id)
}

func (m *Map) LoadMapData(ctx context.Context) ([]byte, error) {
	data, err := m.redis.Get(ctx, m.dataKey()).Bytes()
	if err != nil {
		return nil, err
	}

	return data, nil
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

type mapMeta struct {
	Created time.Time
	Creator string
}

func (s *Map) load(ctx context.Context) (*mapMeta, error) {
	var jsonMap mapMeta
	err := loadJSON(ctx, s.redis, s.metaKey(), &jsonMap)
	if err != nil {
		return nil, err
	}

	return &jsonMap, nil
}

func (s *Map) save(ctx context.Context, data mapMeta) error {
	return saveJSON(ctx, s.redis, s.metaKey(), data)
}

func (s *Map) GetCreator(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Creator, nil
}

func (s *Map) Expire(ctx context.Context, when time.Duration) error {
	pipe := s.redis.Pipeline()
	pipe.Expire(ctx, s.dataKey(), when)
	pipe.Expire(ctx, s.metaKey(), when)
	_, err := pipe.Exec(ctx)
	return err
}

func (v *Verse) NewMap(ctx context.Context, creator string) (*Map, error) {
	map_, err := maps.NewMap()
	if err != nil {
		return nil, err
	}

	defer map_.Destroy()

	return v.SaveGameMap(ctx, creator, map_)
}

func (v *Verse) HaveMap(ctx context.Context, id string) (bool, error) {
	return v.have(ctx, fmt.Sprintf(MAP_DATA_KEY, id))
}

func (v *Verse) GetMap(ctx context.Context, id string) (*Map, error) {
	map_ := Map{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	return &map_, nil
}

func (v *Verse) SaveGameMap(ctx context.Context, creator string, gameMap *maps.GameMap) (*Map, error) {
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
			redis: v.redis,
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
		err = map_.Expire(ctx, time.Hour * 24)
		if err != nil {
		    return nil, err
		}
	}

	return &map_, nil
}

type UserSpace struct {
	entity
	id string
}

type spaceMeta struct {
	Owner       string
	Map         string
	Description string
}

func (s *UserSpace) GetID() string {
	return s.id
}

func (s *UserSpace) key() string {
	return fmt.Sprintf(SPACE_KEY, s.id)
}

func (s *UserSpace) load(ctx context.Context) (*spaceMeta, error) {
	var jsonSpace spaceMeta
	err := loadJSON(ctx, s.redis, s.key(), &jsonSpace)
	if err != nil {
		return nil, err
	}

	return &jsonSpace, nil
}

func (s *UserSpace) save(ctx context.Context, data spaceMeta) error {
	return saveJSON(ctx, s.redis, s.key(), data)
}

func (s *UserSpace) Expire(ctx context.Context, when time.Duration) error {
	return s.redis.Expire(ctx, s.key(), when).Err()
}

func (s *UserSpace) GetMeta(ctx context.Context) (*spaceMeta, error) {
	return s.load(ctx)
}

func (s *UserSpace) GetOwner(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Owner, nil
}

func (s *UserSpace) SetOwner(ctx context.Context, owner string) error {
	meta, err := s.load(ctx)
	if err != nil {
		return err
	}
	meta.Owner = owner
	return s.save(ctx, *meta)
}

func (s *UserSpace) GetDescription(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Description, nil
}

func (s *UserSpace) SetDescription(ctx context.Context, description string) error {
	meta, err := s.load(ctx)
	if err != nil {
		return err
	}
	meta.Description = description
	return s.save(ctx, *meta)
}

func (s *UserSpace) GetMapID(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Map, nil
}

func (s *UserSpace) GetMap(ctx context.Context) (*Map, error) {
	id, err := s.GetMapID(ctx)
	if err != nil {
		return nil, err
	}

	return s.verse.GetMap(ctx, id)
}

func (s *UserSpace) SetMapID(ctx context.Context, id string) error {
	meta, err := s.load(ctx)
	if err != nil {
		return err
	}
	meta.Map = id
	return s.save(ctx, *meta)
}

func (v *Verse) NewSpaceID(ctx context.Context) (string, error) {
	for {
		number, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		if err != nil {
			return "", err
		}

		bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
		hash := fmt.Sprintf("%x", bytes)[:5]
		value, err := v.redis.Exists(ctx, fmt.Sprintf(SPACE_KEY, hash)).Result()
		if err != nil {
			return "", err
		}

		if value == 0 {
			return hash, nil
		}
	}
}

func (v *Verse) NewSpace(ctx context.Context, creator string) (*UserSpace, error) {
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

	err = space.save(ctx, spaceMeta{
		Map:         map_.GetID(),
		Owner:       creator,
		Description: "",
	})
	if err != nil {
		return nil, err
	}

	return &space, nil
}

func (v *Verse) HaveSpace(ctx context.Context, id string) (bool, error) {
	return v.have(ctx, fmt.Sprintf(SPACE_KEY, id))
}

func (v *Verse) LoadSpace(ctx context.Context, id string) (*UserSpace, error) {
	space := UserSpace{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	_, err := space.load(ctx)
	if err != nil {
		return nil, err
	}

	return &space, nil
}

// Find a space by a prefix
func (v *Verse) FindSpace(ctx context.Context, prefix string) (*UserSpace, error) {
	// Check first if the space name is fully specified
	fullExists, err := v.HaveSpace(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if fullExists {
		return v.LoadSpace(ctx, prefix)
	}

	keys, err := v.redis.Keys(
		ctx,
		fmt.Sprintf(SPACE_KEY, prefix)+"*",
	).Result()

	if len(keys) == 0 {
		return nil, fmt.Errorf("no keys matching prefix")
	}

	if len(keys) > 1 {
		return nil, fmt.Errorf("unambiguous reference")
	}

	return v.LoadSpace(ctx, keys[0])
}

type User struct {
	entity
	id string
}

type userMeta struct {
	// Space ID
	Home string
}

func (u *User) key() string {
	return fmt.Sprintf(USER_KEY, u.id)
}

func (u *User) GetID() string {
	return u.id
}

func (u *User) load(ctx context.Context) (*userMeta, error) {
	var jsonUser userMeta
	err := loadJSON(ctx, u.redis, u.key(), &jsonUser)
	if err != nil {
		return nil, err
	}

	return &jsonUser, nil
}

func (u *User) save(ctx context.Context, data userMeta) error {
	return saveJSON(ctx, u.redis, u.key(), data)
}

func (u *User) GetHomeID(ctx context.Context) (string, error) {
	meta, err := u.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Home, nil
}

func (u *User) GetHomeSpace(ctx context.Context) (*UserSpace, error) {
	id, err := u.GetHomeID(ctx)
	if err != nil {
		return nil, err
	}

	return u.verse.LoadSpace(ctx, id)
}

func (v *Verse) NewUser(ctx context.Context, id string) (*User, error) {
	user := User{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	space, err := v.NewSpace(ctx, id)
	if err != nil {
		return nil, err
	}

	err = user.save(ctx, userMeta{
		Home: space.GetID(),
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (v *Verse) HaveUser(ctx context.Context, id string) (bool, error) {
	return v.have(ctx, fmt.Sprintf(USER_KEY, id))
}

func (v *Verse) GetUser(ctx context.Context, id string) (*User, error) {
	exists, err := v.HaveUser(ctx, id)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, redis.Nil
	}

	user := User{
		id: id,
		entity: entity{
			redis: v.redis,
			verse: v,
		},
	}

	_, err = user.load(ctx)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (v *Verse) GetOrCreateUser(ctx context.Context, id string) (*User, error) {
	exists, err := v.HaveUser(ctx, id)
	if err != nil {
		return nil, err
	}

	if !exists {
		return v.NewUser(ctx, id)
	}

	return v.GetUser(ctx, id)
}
