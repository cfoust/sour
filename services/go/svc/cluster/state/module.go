package state

import (
	"github.com/cfoust/sour/svc/cluster/config"

	"github.com/go-redis/redis/v9"
)

const Nil = redis.Nil

type StateService struct {
	client *redis.Client
}

func NewStateService(settings config.RedisSettings) *StateService {
	return &StateService{
		client: redis.NewClient(&redis.Options{
			Addr:     settings.Address,
			Password: settings.Password,
			DB:       settings.DB,
		}),
	}
}
