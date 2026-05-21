package packager

import (
	"context"
	"fmt"

	"github.com/cfoust/sour/pkg/assets"
)

// MemReader implements assets.PackageReader backed by a MemStore.
type MemReader struct {
	store     *MemStore
	indexData []byte
}

var _ assets.PackageReader = (*MemReader)(nil)

func NewMemReader(mp *MemPackager) (*MemReader, error) {
	indexData, err := mp.DumpIndex()
	if err != nil {
		return nil, fmt.Errorf("generating index: %w", err)
	}

	return &MemReader{
		store:     mp.Store,
		indexData: indexData,
	}, nil
}

func (r *MemReader) Index(ctx context.Context) ([]byte, error) {
	return r.indexData, nil
}

func (r *MemReader) Read(ctx context.Context, id string) ([]byte, error) {
	data, err := r.store.Get(id)
	if err == nil {
		return data, nil
	}

	// Bundles are stored with .sour suffix
	data, err = r.store.Get(id + ".sour")
	if err == nil {
		return data, nil
	}

	return nil, assets.Missing
}

// AsPackagedRoot creates a PackagedRoot backed by in-memory data.
func AsPackagedRoot(ctx context.Context, mp *MemPackager) (*assets.PackagedRoot, error) {
	reader, err := NewMemReader(mp)
	if err != nil {
		return nil, err
	}

	return assets.NewPackagedRoot(ctx, reader, "", false)
}
