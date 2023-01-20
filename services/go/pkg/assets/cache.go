package assets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v9"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte) error
}

type FSCache string

var Missing = fmt.Errorf("not in cache")

func (f FSCache) getPath(key string) string {
	return filepath.Join(string(f), key)
}

func (f FSCache) Get(key string) ([]byte, error) {
	target := f.getPath(key)

	if !FileExists(target) {
		return nil, Missing
	}

	return os.ReadFile(target)
}

func (f FSCache) Set(key string, data []byte) error {
	target := f.getPath(key)
	return WriteBytes(data, target)
}

const (
	ASSET_KEY    = "assets-%s"
	ASSET_EXPIRY = time.Duration(1 * time.Hour)
)

type RedisCache struct {
	client *redis.Client
}

func (r *RedisCache) Get(id string) ([]byte, error) {
	key := fmt.Sprintf(ASSET_KEY, id)
	data, err := r.client.Get(context.Background(), key).Bytes()

	if err == redis.Nil {
		return nil, Missing
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *RedisCache) Set(id string, data []byte) error {
	return r.client.Set(context.Background(), id, data, ASSET_EXPIRY).Err()
}

var _ Cache = (*FSCache)(nil)
var _ Cache = (*RedisCache)(nil)
