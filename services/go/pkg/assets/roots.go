package assets

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

type Root interface {
	Exists(ctx context.Context, path string) bool
	ReadFile(ctx context.Context, path string) ([]byte, error)
	Reference(ctx context.Context, path string) (string, error)
}

// Sourdump can return (id, path) or (path, path) pairs

// An FSRoot is just an absolute path on the FS.
type FSRoot string

func (f FSRoot) getPath(file string) string {
	return filepath.Join(string(f), file)
}

func (f FSRoot) Exists(ctx context.Context, path string) bool {
	if _, err := os.Stat(f.getPath(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (f FSRoot) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return os.ReadFile(f.getPath(path))
}

func (f FSRoot) Reference(ctx context.Context, path string) (string, error) {
	if !f.Exists(ctx, path) {
		return "", fmt.Errorf("path %s not found in root", path)
	}

	return fmt.Sprintf("fs:%s", f.getPath(path)), nil
}

type RemoteRoot struct {
	cache  Store
	source string
	url    string
	// A path inside of the virtual FS to treat as the "root".
	base string

	// Quick check for existence
	assets map[string]struct{}

	// index -> asset id
	idLookup map[int]string

	// When building a bundle that references an asset in this root (and
	// the asset IDs match), do not include the asset in the bundle. This
	// is used for keeping desktop bundles small.
	skip bool

	maps []SlimMap

	bundles map[string]*[]Asset

	// FS path -> asset id
	FS map[string]int
}

func NewRemoteRoot(
	ctx context.Context,
	cache Store,
	url string,
	base string,
	shouldCache bool,
	skip bool,
) (*RemoteRoot, error) {
	urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))

	indexData, err := cache.Get(ctx, urlHash)
	if err != nil {
		if err != Missing {
			return nil, err
		}

		indexData, err = DownloadBytes(url)
		if err != nil {
			return nil, err
		}

		if shouldCache {
			err = cache.Set(ctx, urlHash, indexData)
			if err != nil {
				return nil, err
			}
		}
	}

	var index Index
	if err := cbor.Unmarshal(indexData, &index); err != nil {
		return nil, err
	}

	root := RemoteRoot{
		cache:  cache,
		url:    CleanSourcePath(url),
		source: url,
		base:   base,
		skip:   skip,
	}

	bundles := make(map[string]*[]Asset)
	assets := make(map[string]struct{})
	idLookup := make(map[int]string)
	fs := make(map[string]int)

	for i, asset := range index.Assets {
		assets[asset] = struct{}{}
		idLookup[i] = asset
	}

	for _, bundle := range index.Bundles {
		bundleAssets := make([]Asset, 0)

		for _, asset := range bundle.Assets {
			bundleAssets = append(bundleAssets, asset)
		}

		bundles[bundle.Id] = &bundleAssets
	}

	for _, ref := range index.Refs {
		path := ref.Path
		if base != "" {
			if !strings.HasPrefix(path, base) {
				continue
			}
			path = path[len(base):]
		}
		fs[path] = ref.Id
	}

	maps := make([]SlimMap, len(index.Maps))
	for _, map_ := range index.Maps {
		hasCFG := false

		cfgName := fmt.Sprintf(
			"%s.cfg",
			map_.Name,
		)

		for _, asset := range map_.Assets {
			if strings.HasSuffix(asset.Path, cfgName) {
				hasCFG = true
			}
		}

		maps = append(
			maps,
			SlimMap{
				Id:     map_.Id,
				Name:   map_.Name,
				Ogz:    map_.Ogz,
				Bundle: map_.Bundle,
				HasCFG: hasCFG,
			},
		)
	}
	root.maps = maps

	root.bundles = bundles
	root.assets = assets
	root.idLookup = idLookup
	root.FS = fs

	return &root, nil
}

func (f *RemoteRoot) Exists(ctx context.Context, path string) bool {
	_, ok := f.FS[path]
	return ok
}

func (f *RemoteRoot) GetID(index int) (string, error) {
	if id, ok := f.idLookup[index]; ok {
		return id, nil
	}

	return "", Missing
}

func (f *RemoteRoot) Reference(ctx context.Context, path string) (string, error) {
	index, ok := f.FS[path]
	if !ok {
		return "", Missing
	}

	id, ok := f.idLookup[index]
	if !ok {
		return "", Missing
	}

	return fmt.Sprintf("id:%s", id), nil
}

func (f *RemoteRoot) ReadAsset(ctx context.Context, id string) ([]byte, error) {
	if _, ok := f.assets[id]; !ok {
		return nil, Missing
	}

	cacheData, err := f.cache.Get(ctx, id)
	if err != nil && err != Missing {
		return nil, err
	}
	if err == nil {
		return cacheData, nil
	}

	url := fmt.Sprintf("%s%s", f.url, id)
	data, err := DownloadBytes(url)
	if err != nil {
		return nil, err
	}

	err = f.cache.Set(ctx, id, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (f *RemoteRoot) ReadFile(ctx context.Context, path string) ([]byte, error) {
	index, ok := f.FS[path]
	if !ok {
		return nil, Missing
	}

	id, ok := f.idLookup[index]
	if !ok {
		return nil, Missing
	}

	return f.ReadAsset(ctx, id)
}

var _ Root = (*FSRoot)(nil)
var _ Root = (*RemoteRoot)(nil)

func LoadRoots(ctx context.Context, cache Store, targets []string, onlyMaps bool) ([]Root, error) {
	roots := make([]Root, 0)
	haveSkip := false
	var skipRoot *RemoteRoot

	for _, target := range targets {
		// Specify a base dir with @/base/dir
		base := ""
		atIndex := strings.LastIndex(target, "@")
		if atIndex != -1 {
			base = target[atIndex+1:]
			target = target[:atIndex]
		}

		skip := false
		if strings.HasPrefix(target, "skip:") {
			if haveSkip {
				return nil, fmt.Errorf("you can only have one skip root")
			}
			skip = true
			haveSkip = true
			target = target[5:]
		}

		shouldCache := true
		if strings.HasPrefix(target, "!") {
			shouldCache = false
			target = target[1:]
		}

		if !strings.HasPrefix(target, "http") {
			absolute, err := filepath.Abs(target)
			if err != nil {
				return nil, err
			}
			roots = append(roots, FSRoot(absolute))
			continue
		}

		root, err := NewRemoteRoot(
			ctx,
			cache,
			target,
			base,
			shouldCache,
			skip,
		)
		if err != nil {
			return nil, err
		}

		if skip {
			skipRoot = root
		}
		roots = append(roots, root)
	}

	// We can save memory by removing all bundle assets found in the skip
	// root
	if haveSkip {
		for _, root := range roots {
			remote, ok := root.(*RemoteRoot)
			if !ok {
				continue
			}

			newBundles := make(map[string]*[]Asset)
			for id, assets := range remote.bundles {
				newAssets := make([]Asset, 0)
				for _, asset := range *assets {
					shouldSkip := false

					if refID, ok := skipRoot.FS[asset.Path]; ok {
						if assetId, ok := skipRoot.idLookup[refID]; ok {
							shouldSkip = asset.Id == assetId
						}
					}

					if shouldSkip {
						continue
					}

					newAssets = append(newAssets, asset)
				}

				newBundles[id] = &newAssets
			}

			remote.bundles = newBundles
		}
	}

	if onlyMaps {
		// First pass: note all of the assets used by maps
		mapAssets := make(map[string]struct{})
		for _, root := range roots {
			remote, ok := root.(*RemoteRoot)
			if !ok {
				continue
			}
			for _, _map := range remote.maps {
				mapAssets[_map.Ogz] = struct{}{}
			}
			for _, assets := range remote.bundles {
				for _, asset := range *assets {
					mapAssets[asset.Id] = struct{}{}
				}
			}
		}

		// Second pass: clear out assets not used by maps
		for _, root := range roots {
			remote, ok := root.(*RemoteRoot)
			if !ok {
				continue
			}

			// We need to preserve the whole FS to check existence
			if remote.skip {
				continue
			}

			newAssets := make(map[string]struct{})
			for asset := range remote.assets {
				if _, ok := mapAssets[asset]; ok {
					newAssets[asset] = struct{}{}
				}
			}
			remote.assets = newAssets
			remote.idLookup = make(map[int]string)
			remote.FS = make(map[string]int)
		}

		// Force a GC to free memory
		runtime.GC()
	}

	return roots, nil
}
