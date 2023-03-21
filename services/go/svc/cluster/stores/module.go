package stores

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/state"

	"gorm.io/gorm"
)

type AssetStorage struct {
	defaultLocation string
	defaultStore    assets.Store
	stores          map[string]assets.Store
	db              *gorm.DB
}

func (s *AssetStorage) Get(ctx context.Context, asset *state.Asset) ([]byte, error) {
	store, ok := s.stores[asset.Location]
	if !ok {
		return nil, fmt.Errorf("store for asset not found: %s", asset.Location)
	}

	return store.Get(ctx, asset.Hash)
}

func (s *AssetStorage) Store(ctx context.Context, user *state.User, extension string, data []byte) (*state.Asset, error) {
	store := s.defaultStore
	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	err := store.Set(ctx, hash, data)
	if err != nil {
		return nil, err
	}

	asset := state.Asset{
		Creatable: state.NewCreatable(user),
		Hash:      hash,
		Extension: extension,
		Size:      uint(len(data)),
		Location:  s.defaultLocation,
	}

	err = s.db.WithContext(ctx).Create(&asset).Error
	if err != nil {
		return nil, err
	}

	return &asset, nil
}

func New(db *gorm.DB, storeConfigs []config.Store) (*AssetStorage, error) {
	stores := make(map[string]assets.Store)

	var defaultStore assets.Store = nil
	var defaultLocation string

	for _, storeConfig := range storeConfigs {
		var store assets.Store

		switch storeConfig.Config.Type() {
		case config.StoreTypeFS:
			fsConfig, ok := storeConfig.Config.(config.FSStoreConfig)
			if !ok {
				continue
			}

			err := os.MkdirAll(fsConfig.Path, 0755)
			if err != nil {
				return nil, err
			}

			store = assets.FSStore(fsConfig.Path)
			stores[storeConfig.Name] = store
		}

		if storeConfig.Default {
			defaultStore = store
			defaultLocation = storeConfig.Name
		}
	}

	if defaultStore == nil {
		return nil, fmt.Errorf("missing default asset store")
	}

	return &AssetStorage{
		stores:          stores,
		defaultStore:    defaultStore,
		defaultLocation: defaultLocation,
		db:              db,
	}, nil
}
