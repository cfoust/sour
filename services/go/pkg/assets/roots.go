package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"crypto/sha256"

	"github.com/fxamacker/cbor/v2"
)

type Root interface {
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
	Reference(path string) (string, error)
}

// Sourdump can return (id, path) or (path, path) pairs

// An FSRoot is just an absolute path on the FS.
type FSRoot string

func (f FSRoot) getPath(file string) string {
	return filepath.Join(string(f), file)
}

func (f FSRoot) Exists(path string) bool {
	if _, err := os.Stat(f.getPath(path)); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (f FSRoot) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(f.getPath(path))
}

func (f FSRoot) Reference(path string) (string, error) {
	if !f.Exists(path) {
		return "", fmt.Errorf("path %s not found in root", path)
	}

	return f.getPath(path), nil
}

type RemoteRoot struct {
	cache Cache
	url   string
	// A path inside of the virtual FS to treat as the "root".
	base string

	// Quick check for existence
	assets map[string]struct{}

	// index -> asset id
	idLookup map[int]string

	// FS path -> asset id
	FS map[string]int
}

func NewRemoteRoot(cache Cache, url string, base string) (*RemoteRoot, error) {
	urlHash := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))

	indexData, err := cache.Get(urlHash)
	if err != nil {
		if err != Missing {
			return nil, err
		}

		indexData, err = DownloadBytes(url)
		if err != nil {
			return nil, err
		}

		err = cache.Set(urlHash, indexData)
		if err != nil {
		    return nil, err
		}
	}

	var index Index
	if err := cbor.Unmarshal(indexData, &index); err != nil {
		return nil, err
	}

	assets := make(map[string]struct{})
	idLookup := make(map[int]string)
	for i, asset := range index.Assets {
		assets[asset] = struct{}{}
		idLookup[i] = asset
	}

	fs := make(map[string]int)
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

	return &RemoteRoot{
		cache:    cache,
		url:      CleanSourcePath(url),
		base:     base,
		assets:   assets,
		idLookup: idLookup,
		FS:       fs,
	}, nil
}

func (f *RemoteRoot) Exists(path string) bool {
	_, ok := f.FS[path]
	return ok
}

func (f *RemoteRoot) Reference(path string) (string, error) {
	index, ok := f.FS[path]
	if !ok {
		return "", Missing
	}

	id, ok := f.idLookup[index]
	if !ok {
		return "", Missing
	}

	return id, nil
}

func (f *RemoteRoot) ReadAsset(id string) ([]byte, error) {
	if _, ok := f.assets[id]; !ok {
		return nil, Missing
	}

	cacheData, err := f.cache.Get(id)
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

	err = f.cache.Set(id, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (f *RemoteRoot) ReadFile(path string) ([]byte, error) {
	index, ok := f.FS[path]
	if !ok {
		return nil, Missing
	}

	id, ok := f.idLookup[index]
	if !ok {
		return nil, Missing
	}

	return f.ReadAsset(id)
}

var _ Root = (*FSRoot)(nil)
var _ Root = (*RemoteRoot)(nil)

func LoadRoots(cache Cache, targets []string) ([]Root, error) {
	roots := make([]Root, 0)
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			absolute, err := filepath.Abs(target)
			if err != nil {
				return nil, err
			}
			roots = append(roots, FSRoot(absolute))
			continue
		}

		// Specify a base dir with :/base/dir
		base := ""
		colons := strings.Count(target, ":")
		if colons == 2 {
			lastColon := strings.LastIndex(target, ":")
			base = target[lastColon+1:]
			target = target[:lastColon]
		}

		root, err := NewRemoteRoot(cache, target, base)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}

	return roots, nil
}
