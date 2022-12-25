package redis

import (
	"context"

	"github.com/cfoust/sour/svc/cluster/config"

	"github.com/go-redis/redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(settings config.RedisSettings) *RedisService {
	return &RedisService{
		client: redis.NewClient(&redis.Options{
			Addr:     settings.Address,
			Password: settings.Password,
			DB:       settings.DB,
		}),
	}
}

func (r *RedisService) Set(ctx context.Context) error {
	return r.client.Set(ctx, "key", "value", 0).Err()
}
