package assets

import (
	"context"
	"strings"
)

type FetchResult struct {
	Data []byte
	Err  error
}

type Job struct {
	Id     string
	Result chan FetchResult
}

type AssetFetcher struct {
	assets  chan Job
	bundles chan Job
	roots   []*RemoteRoot
	cache   Cache
}

func NewAssetFetcher(cache Cache, roots []string, onlyMaps bool) (*AssetFetcher, error) {
	loaded, err := LoadRoots(cache, roots, onlyMaps)
	if err != nil {
		return nil, err
	}

	remotes := make([]*RemoteRoot, 0)
	for _, root := range loaded {
		if remote, ok := root.(*RemoteRoot); ok {
			remotes = append(remotes, remote)
		}
	}

	return &AssetFetcher{
		roots:   remotes,
		assets:  make(chan Job),
		bundles: make(chan Job),
		cache:   cache,
	}, nil
}

func (m *AssetFetcher) getAsset(ctx context.Context, id string) ([]byte, error) {
	for _, root := range m.roots {
		data, err := root.ReadAsset(id)
		if err == Missing {
			continue
		}
		return data, err
	}

	return nil, Missing
}

func (m *AssetFetcher) getBundle(ctx context.Context, id string) ([]byte, error) {
	return nil, Missing
}

func (m *AssetFetcher) pollAssetJobs(ctx context.Context) {
	for {
		select {
		case job := <-m.assets:
			data, err := m.getAsset(ctx, job.Id)
			job.Result <- FetchResult{
				Data: data,
				Err:  err,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *AssetFetcher) pollBundleJobs(ctx context.Context) {
	for {
		select {
		case job := <-m.bundles:
			data, err := m.getBundle(ctx, job.Id)
			job.Result <- FetchResult{
				Data: data,
				Err:  err,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *AssetFetcher) PollDownloads(ctx context.Context) {
	// We want these to be separate goroutines since an asset fetch should
	// not block a bundle fetch
	go m.pollAssetJobs(ctx)
	go m.pollBundleJobs(ctx)
}

func (m *AssetFetcher) fetchAsset(ctx context.Context, id string) ([]byte, error) {
	out := make(chan FetchResult)
	m.assets <- Job{
		Id:     id,
		Result: out,
	}

	result := <-out
	return result.Data, result.Err
}

type FoundMap struct {
	Map  *SlimMap
	Root *RemoteRoot
}

func (m *AssetFetcher) FindMap(needle string) *FoundMap {
	otherTarget := needle + ".ogz"
	for _, root := range m.roots {
		for _, gameMap := range root.maps {
			if gameMap.Name != needle && gameMap.Name != otherTarget && !strings.HasPrefix(gameMap.Id, needle) {
				continue
			}

			return &FoundMap{
				Map:  &gameMap,
				Root: root,
			}
		}
	}

	return nil
}

func (m *AssetFetcher) FetchMapBytes(ctx context.Context, needle string) ([]byte, error) {
	map_ := m.FindMap(needle)
	if map_ == nil {
		return nil, Missing
	}

	return m.fetchAsset(ctx, map_.Map.Ogz)
}
