package assets

import (
	"path/filepath"
	"fmt"
	"os"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte) error
}

type FSCache string

var Missing = fmt.Errorf("not in cache")

func (f *FSCache) getPath(key string) string {
	return filepath.Join(string(*f), key)
}

func (f *FSCache) Get(key string) ([]byte, error) {
	target := f.getPath(key)

	if !FileExists(target) {
		return nil, Missing
	}

	return os.ReadFile(target)
}

func (f *FSCache) Set(key string, data []byte) error {
	target := f.getPath(key)
	return WriteBytes(data, target)
}

var _ Cache = (*FSCache)(nil)
