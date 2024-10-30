package assets

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
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
	roots   []*PackagedRoot
	cache   Store
}

func NewAssetFetcher(ctx context.Context, cache Store, roots []string, onlyMaps bool) (*AssetFetcher, error) {
	loaded, err := LoadRoots(ctx, cache, roots, onlyMaps)
	if err != nil {
		return nil, err
	}

	remotes := make([]*PackagedRoot, 0)
	for _, root := range loaded {
		if remote, ok := root.(*PackagedRoot); ok {
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
		data, err := root.ReadAsset(ctx, id)
		if err == Missing {
			continue
		}
		return data, err
	}

	return nil, Missing
}

func (m *AssetFetcher) getBundle(ctx context.Context, id string) ([]byte, error) {
	key := fmt.Sprintf(ASSET_KEY, id)

	cacheData, err := m.cache.Get(ctx, key)
	if err != nil && err != Missing {
		return nil, err
	}
	if err == nil {
		return cacheData, nil
	}

	var assets *[]Asset
	for _, root := range m.roots {
		if bundleAssets, ok := root.bundles[id]; ok {
			assets = bundleAssets
			break
		}
	}

	if assets == nil {
		return nil, Missing
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)

	for _, asset := range *assets {
		data, err := m.fetchAsset(ctx, asset.Id)
		if err != nil {
			return nil, err
		}

		file, err := writer.Create(asset.Path)
		if err != nil {
			return nil, err
		}

		_, err = file.Write(data)
		if err != nil {
			return nil, err
		}
	}

	writer.Close()

	data := buffer.Bytes()

	err = m.cache.Set(ctx, key, data)
	if err != nil {
		return nil, err
	}

	return data, nil
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

func (m *AssetFetcher) fetchBundle(ctx context.Context, id string) ([]byte, error) {
	out := make(chan FetchResult)
	m.bundles <- Job{
		Id:     id,
		Result: out,
	}

	result := <-out
	return result.Data, result.Err
}

type FoundMap struct {
	Map   *SlimMap
	Root  *PackagedRoot
	fetch *AssetFetcher
}

func (f *FoundMap) GetOGZ(ctx context.Context) ([]byte, error) {
	return f.fetch.fetchAsset(ctx, f.Map.Ogz)
}

func (f *FoundMap) GetBundle(ctx context.Context) ([]byte, error) {
	return f.fetch.fetchBundle(ctx, f.Map.Bundle)
}

func (m *AssetFetcher) GetMaps(skipRoot string) []SlimMap {
	maps := make([]SlimMap, 0)

	skippedMaps := make(map[string]struct{})
	for _, root := range m.roots {
		if root.source != skipRoot {
			continue
		}
		for _, gameMap := range root.maps {
			skippedMaps[gameMap.Name] = struct{}{}
		}
	}

	for _, root := range m.roots {
		for _, gameMap := range root.maps {
			if _, ok := skippedMaps[gameMap.Name]; ok {
				continue
			}
			maps = append(maps, gameMap)
		}
	}

	return maps
}

func (m *AssetFetcher) FindMap(needle string) *FoundMap {
	otherTarget := needle + ".ogz"
	for _, root := range m.roots {
		for _, gameMap := range root.maps {
			if gameMap.Name != needle && gameMap.Name != otherTarget && !strings.HasPrefix(gameMap.Id, needle) {
				continue
			}

			return &FoundMap{
				Map:   &gameMap,
				Root:  root,
				fetch: m,
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

func (m *AssetFetcher) FetchMapBundle(ctx context.Context, needle string) ([]byte, error) {
	map_ := m.FindMap(needle)
	if map_ == nil {
		return nil, Missing
	}

	return m.fetchBundle(ctx, map_.Map.Bundle)
}
