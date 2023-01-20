package assets

import (
	"fmt"
	"os"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

type Root interface {
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
}

// Sourdump can return (id, path) or (path, path) pairs

// An FSRoot is just an absolute path on the FS.
type FSRoot string

func (f FSRoot) Exists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (f FSRoot) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
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

	// fs path -> asset id
	fs map[string]int
}

func NewRemoteRoot(cache Cache, url string, base string) (*RemoteRoot, error) {
	indexData, err := DownloadBytes(url)
	if err != nil {
		return nil, err
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
		fs[ref.Path] = ref.Id
	}

	return &RemoteRoot{
		cache:    cache,
		url:      url,
		base:     CleanSourcePath(base),
		assets:   assets,
		idLookup: idLookup,
		fs:       fs,
	}, nil
}

func (f *RemoteRoot) Exists(path string) bool {
	_, ok := f.fs[path]
	return ok
}

func (f *RemoteRoot) ReadFile(path string) ([]byte, error) {
	index, ok := f.fs[path]
	if !ok {
		return nil, Missing
	}

	id, ok := f.idLookup[index]
	if !ok {
		return nil, Missing
	}

	cacheData, err := f.cache.Get(id)
	if err != nil && err != Missing {
	    return nil, err
	}
	if err == nil {
		return cacheData, nil
	}

	url := fmt.Sprintf("%s%s", f.base, id)
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

var _ Root = (*FSRoot)(nil)
var _ Root = (*RemoteRoot)(nil)

func LoadRoots(cache Cache, targets []string) ([]Root, error) {
	roots := make([]Root, 0)
	for _, target := range targets {
		if !strings.HasPrefix(target, "http") {
			roots = append(roots, FSRoot(target))
			continue
		}

		root, err := NewRemoteRoot(cache, target, "")
		if err != nil {
		    return nil, err
		}
		roots = append(roots, root)
	}

	return roots, nil
}
