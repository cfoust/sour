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
	AUTH_PREFIX  = "auth-"
	TOKEN_PREFIX = AUTH_PREFIX + "token-"

	KEY_CODE_TO_ID               = AUTH_PREFIX + "code-%s"
	KEY_ID_TO_TOKEN              = AUTH_PREFIX + "token-%s"
	KEY_TOKEN_TO_REFRESH         = TOKEN_PREFIX + "refresh-%s"
	KEY_TOKEN_TO_REFRESH_EXPIRED = TOKEN_PREFIX + "refresh-expired-%s"
	KEY_ID_TO_PUBLIC             = AUTH_PREFIX + "public-%s"
	KEY_ID_TO_PRIVATE            = AUTH_PREFIX + "private-%s"
)

const Nil = redis.Nil

func (r *StateService) GetIdForCode(ctx context.Context, code string) (string, error) {
	return r.client.Get(ctx, fmt.Sprintf(KEY_CODE_TO_ID, code)).Result()
}

func (r *StateService) SetIdForCode(ctx context.Context, code string, id string, expiresIn int) error {
	return r.client.Set(
		ctx,
		fmt.Sprintf(KEY_CODE_TO_ID, code),
		id,
		time.Duration(expiresIn)*time.Second,
	).Err()
}

func (r *StateService) GetTokenForId(ctx context.Context, id string) (string, error) {
	return r.client.Get(ctx, fmt.Sprintf(KEY_ID_TO_TOKEN, id)).Result()
}

func (r *StateService) SetTokenForId(ctx context.Context, id string, token string, expiresIn int) error {
	return r.client.Set(
		ctx,
		fmt.Sprintf(KEY_ID_TO_TOKEN, id),
		token,
		time.Duration(expiresIn)*time.Second,
	).Err()
}

func (r *StateService) GetTokenForCode(ctx context.Context, code string) (string, error) {
	id, err := r.GetIdForCode(ctx, code)
	if err != nil {
		return "", err
	}
	return r.GetTokenForId(ctx, id)
}

func (r *StateService) GetAuthKeyForId(ctx context.Context, id string) (public string, private string, err error) {
	// TODO pipeline
	public, err = r.client.Get(ctx, fmt.Sprintf(KEY_ID_TO_PUBLIC, id)).Result()
	if err != nil {
		return "", "", err
	}
	private, err = r.client.Get(ctx, fmt.Sprintf(KEY_ID_TO_PRIVATE, id)).Result()
	if err != nil {
		return "", "", err
	}

	return public, private, nil
}

func (r *StateService) SaveAuthKeyForUser(ctx context.Context, id string, public string, private string) error {
	// TODO pipeline
	err := r.client.Set(
		ctx,
		fmt.Sprintf(KEY_ID_TO_PUBLIC, id),
		public,
		0,
	).Err()
	if err != nil {
		return err
	}
	err = r.client.Set(
		ctx,
		fmt.Sprintf(KEY_ID_TO_PRIVATE, id),
		private,
		0,
	).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *StateService) SaveToken(ctx context.Context, token string, expiresIn int, refreshToken string) error {
	pipe := r.client.Pipeline()

	pipe.Set(
		ctx,
		fmt.Sprintf(KEY_TOKEN_TO_REFRESH, token),
		refreshToken,
		time.Duration(expiresIn)*time.Second,
	)

	pipe.Set(
		ctx,
		fmt.Sprintf(KEY_TOKEN_TO_REFRESH_EXPIRED, token),
		// Value does not matter
		"1",
		// Flag this a bit sooner so we can refresh it
		time.Duration(expiresIn/2)*time.Second,
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
