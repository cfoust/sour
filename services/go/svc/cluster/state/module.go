package state

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/svc/cluster/config"

	"github.com/go-redis/redis/v9"
)

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

const (
	AUTH_PREFIX       = "auth-"
	KEY_CODE_TO_TOKEN = AUTH_PREFIX + "code-%s"
	TOKEN_PREFIX = AUTH_PREFIX + "token-"
	KEY_TOKEN_TO_REFRESH = TOKEN_PREFIX + "refresh-%s"
	KEY_TOKEN_TO_REFRESH_EXPIRED = TOKEN_PREFIX + "refresh-expired-%s"
)

const Nil = redis.Nil

func (r *StateService) GetTokenForCode(ctx context.Context, code string) (string, error) {
	result, err := r.client.Get(ctx, fmt.Sprintf(KEY_CODE_TO_TOKEN, code)).Result()
	return result, err
}

func (r *StateService) SaveTokenForCode(ctx context.Context, code string, token string, expiresIn int) error {
	return r.client.Set(
		ctx,
		fmt.Sprintf(KEY_CODE_TO_TOKEN, code),
		token,
		time.Duration(expiresIn) * time.Second,
	).Err()
}

func (r *StateService) SaveToken(ctx context.Context, token string, expiresIn int, refreshToken string) error {
	pipe := r.client.Pipeline()

	pipe.Set(
		ctx,
		fmt.Sprintf(KEY_TOKEN_TO_REFRESH, token),
		refreshToken,
		time.Duration(expiresIn) * time.Second,
	)

	pipe.Set(
		ctx,
		fmt.Sprintf(KEY_TOKEN_TO_REFRESH_EXPIRED, token),
		// Value does not matter
		"1",
		// Flag this a bit sooner so we can refresh it
		time.Duration(expiresIn / 2) * time.Second,
	)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *StateService) GetRefreshForToken(ctx context.Context, token string) (string, error) {
	return r.client.Get(ctx, fmt.Sprintf(KEY_TOKEN_TO_REFRESH, token)).Result()
}

func (r *StateService) TokenNeedsRefresh(ctx context.Context, token string) (bool, error) {
	_, err := r.client.Get(ctx, fmt.Sprintf(KEY_TOKEN_TO_REFRESH_EXPIRED, token)).Result()

	if err != nil && err != Nil {
		return false, err
	}

	if err == Nil {
		return true, nil
	}

	return false, nil
}
