package verse

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/cfoust/sour/pkg/maps"

	"github.com/go-redis/redis/v9"
)

const (
	MAP_KEY = "map-%s"
)

func HaveMap(ctx context.Context, client *redis.Client, id string) (bool, error) {
	value, err := client.Exists(ctx, fmt.Sprintf(MAP_KEY, id)).Result()
	if err != nil {
		return false, err
	}

	return value == 1, nil
}

func SaveMap(ctx context.Context, client *redis.Client, map_ *maps.GameMap) (string, error) {
	mapData, err := map_.EncodeOGZ()
	if err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(mapData))
	key := fmt.Sprintf(MAP_KEY, hash)

	// No point in setting this if it already is there
	if exists, _ := HaveMap(ctx, client, hash); exists {
		return hash, nil
	}

	return hash, client.Set(
		ctx,
		key,
		mapData,
		0,
	).Err()
}

func LoadMap(ctx context.Context, client *redis.Client, id string) (*maps.GameMap, error) {
	data, err := client.Get(ctx, fmt.Sprintf(MAP_KEY, id)).Bytes()
	if err != nil {
		return nil, err
	}

	map_, err := maps.FromGZ(data)
	if err := maps.LoadDefaultSlots(map_); err != nil {
		return nil, err
	}

	return map_, nil
}
