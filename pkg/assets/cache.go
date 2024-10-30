package assets

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v9"
)

type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, data []byte) error
}

type FSStore string

var Missing = fmt.Errorf("asset missing")

func (f FSStore) getPath(key string) string {
	return filepath.Join(string(f), key)
}

func (f FSStore) Get(ctx context.Context, key string) ([]byte, error) {
	target := f.getPath(key)

	if !FileExists(target) {
		return nil, Missing
	}

	return os.ReadFile(target)
}

func (f FSStore) Set(ctx context.Context, key string, data []byte) error {
	target := f.getPath(key)
	return WriteBytes(data, target)
}

const (
	ASSET_KEY    = "assets-%s"
	ASSET_EXPIRY = time.Duration(1 * time.Hour)
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

func (r *RedisStore) Get(ctx context.Context, id string) ([]byte, error) {
	key := fmt.Sprintf(ASSET_KEY, id)
	data, err := r.client.Get(ctx, key).Bytes()

	if err == redis.Nil {
		return nil, Missing
	}

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (r *RedisStore) Set(ctx context.Context, id string, data []byte) error {
	key := fmt.Sprintf(ASSET_KEY, id)
	return r.client.Set(ctx, key, data, ASSET_EXPIRY).Err()
}

type RedisCache struct {
	*RedisStore
	ttl time.Duration
}

func NewRedisCache(client *redis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{
		RedisStore: NewRedisStore(client),
		ttl:        ttl,
	}
}

func (r *RedisCache) Set(ctx context.Context, id string, data []byte) error {
	err := r.RedisStore.Set(ctx, id, data)
	if err != nil {
		return err
	}

	return r.client.Expire(ctx, id, r.ttl).Err()
}

var _ Store = (*FSStore)(nil)
var _ Store = (*RedisStore)(nil)
var _ Store = (*RedisCache)(nil)
