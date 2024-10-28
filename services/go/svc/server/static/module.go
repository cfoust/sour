package static

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"
)

//go:embed site
var staticFiles embed.FS

// injectedFile pretends to be an fs.File, but just returns a byte buffer.
type injectedFile struct {
	name   string
	data   []byte
	reader *bytes.Reader
}

var _ fs.File = (*injectedFile)(nil)

type injectedFileInfo struct {
	name string
	size int64
}

var _ fs.FileInfo = (*injectedFileInfo)(nil)

func (i *injectedFileInfo) Name() string {
	return i.name
}

func (i *injectedFileInfo) Size() int64 {
	return i.size
}

func (i *injectedFileInfo) Mode() fs.FileMode {
	return 0444
}

func (i *injectedFileInfo) Sys() interface{} {
	return nil
}

func (i *injectedFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (i *injectedFileInfo) IsDir() bool {
	return false
}

func (i *injectedFile) Stat() (fs.FileInfo, error) {
	return &injectedFileInfo{
		name: i.name,
		size: int64(len(i.data)),
	}, nil
}

func (i *injectedFile) Read(p []byte) (n int, err error) {
	return i.reader.Read(p)
}

func (i *injectedFile) Close() error {
	return nil
}

// dynamicFS is an fs.FS that allows you to overwrite files with injected
// data.
type dynamicFS struct {
	original fs.FS
	injected map[string][]byte
}

var _ fs.FS = (*dynamicFS)(nil)

func (d *dynamicFS) Open(name string) (fs.File, error) {
	if data, ok := d.injected[name]; ok {
		return &injectedFile{
			name:   name,
			data:   data,
			reader: bytes.NewReader(data),
		}, nil
	}

	return d.original.Open(name)
}

func Site() (http.Handler, error) {
	content, _ := fs.Sub(fs.FS(staticFiles), "site")

	// Get index.js and inject config as JSON
	index, _ := fs.ReadFile(content, "index.js")

	prefix := fmt.Sprintf(
		"const INJECTED_SOUR_CONFIG = %s;\n",
		"{}",
	)

	index = append(
		[]byte(prefix),
		index...,
	)

	// Inject index.js
	injected := map[string][]byte{
		"index.js": index,
	}

	return http.FileServer(http.FS(&dynamicFS{
		original: content,
		injected: injected,
	})), nil
}
