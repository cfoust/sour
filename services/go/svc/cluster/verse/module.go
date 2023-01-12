package verse

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/go-redis/redis/v9"
)

const (
	MAP_KEY = "map-%s"
)

type Verse struct {
	redis *redis.Client
}

func (v *Verse) HaveMap(ctx context.Context, id string) (bool, error) {
	value, err := v.redis.Exists(ctx, fmt.Sprintf(MAP_KEY, id)).Result()
	if err != nil {
		return false, err
	}

	return value == 1, nil
}

func (v *Verse) SaveMap(ctx context.Context, map_ *maps.GameMap) (string, error) {
	mapData, err := map_.EncodeOGZ()
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(mapData))
	key := fmt.Sprintf(MAP_KEY, hash)

	// No point in setting this if it already is there
	if exists, _ := v.HaveMap(ctx, hash); exists {
		return hash, nil
	}

	return hash, v.redis.Set(
		ctx,
		key,
		mapData,
		0,
	).Err()
}

func (v *Verse) NewMap(ctx context.Context) (string, error) {
	map_, err := maps.NewMap()
	if err != nil {
		return "", err
	}

	return v.SaveMap(ctx, map_)
}

func (v *Verse) LoadMap(ctx context.Context, id string) (*maps.GameMap, error) {
	data, err := v.redis.Get(ctx, fmt.Sprintf(MAP_KEY, id)).Bytes()
	if err != nil {
		return nil, err
	}

	map_, err := maps.FromGZ(data)
	if err := maps.LoadDefaultSlots(map_); err != nil {
		return nil, err
	}

	return map_, nil
}

type Space struct {
	id    string
	redis *redis.Client
	verse *Verse
}

type SpaceMeta struct {
	Owner string
	Map   string
}

func (s *Space) GetID() string {
	return s.id
}

func (s *Space) load(ctx context.Context) (*SpaceMeta, error) {
	data, err := s.redis.Get(ctx, fmt.Sprintf(MAP_KEY, s.id)).Bytes()
	if err != nil {
		return nil, err
	}

	var jsonSpace SpaceMeta
	err = json.Unmarshal(data, &jsonSpace)
	if err != nil {
		return nil, err
	}

	return &jsonSpace, nil
}

func (s *Space) save(ctx context.Context, data SpaceMeta) error {
	str, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return s.redis.Set(
		ctx,
		fmt.Sprintf(SPACE_KEY, s.id),
		str,
		0,
	).Err()
}

func (s *Space) GetOwner(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Owner, nil
}

func (s *Space) SetOwner(ctx context.Context, owner string) error {
	meta, err := s.load(ctx)
	if err != nil {
		return err
	}
	meta.Owner = owner
	return s.save(ctx, *meta)
}

func (s *Space) GetMap(ctx context.Context) (string, error) {
	meta, err := s.load(ctx)
	if err != nil {
		return "", err
	}

	return meta.Map, nil
}

func (s *Space) SetMap(ctx context.Context, id string) error {
	meta, err := s.load(ctx)
	if err != nil {
		return err
	}
	meta.Map = id
	return s.save(ctx, *meta)
}

const (
	SPACE_KEY = "space-%s"
)

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

func (v *Verse) NewSpace(ctx context.Context) (*Space, error) {
	id, err := v.NewSpaceID(ctx)
	if err != nil {
		return nil, err
	}

	mapId, err := v.NewMap(ctx)
	if err != nil {
		return nil, err
	}

	space := Space{
		id:    id,
		redis: v.redis,
		verse: v,
	}

	err = space.save(ctx, SpaceMeta{
		Map:   mapId,
		Owner: "",
	})
	if err != nil {
		return nil, err
	}

	return &space, nil
}

func (v *Verse) HaveSpace(ctx context.Context, id string) (bool, error) {
	value, err := v.redis.Exists(ctx, fmt.Sprintf(SPACE_KEY, id)).Result()
	if err != nil {
		return false, err
	}

	return value == 1, nil
}

func (v *Verse) LoadSpace(ctx context.Context, id string) (*Space, error) {
	space := Space{
		id:    id,
		redis: v.redis,
		verse: v,
	}

	_, err := space.load(ctx)
	if err != nil {
		return nil, err
	}

	return &space, nil
}
