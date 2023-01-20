package assets

import (
	"fmt"
	"os"
	"path/filepath"
)

type Root interface {
	Exists(path string) bool
	ReadFile(path string) ([]byte, error)
	// Resolve a path inside of the root to one accessible on the filesystem.
	Resolve(uri string) (string, error)
}

// Sourdump can return (id, path) or (path, path) pairs

// An FSRoot is just an absolute path on the FS.
type FSRoot string

func (f *FSRoot) Exists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func (f *FSRoot) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *FSRoot) Resolve(uri string) (string, error) {
	if !f.Exists(uri) {
		return "", fmt.Errorf("file does not exist")
	}

	return filepath.Abs(uri)
}

type RemoteRoot struct {
	url string
	// A path inside of the virtual FS to treat as the "root".
	base string
}

func NewRemoteRoot(url string, base string) *RemoteRoot {
	return &RemoteRoot{
		url:  url,
		base: base,
	}
}

var _ Root = (*FSRoot)(nil)
var _ Root = (*RemoteRoot)(nil)
