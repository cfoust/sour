package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed site
var staticFiles embed.FS

func Site() (http.Handler, error) {
	content, _ := fs.Sub(fs.FS(staticFiles), "site")
	return http.FileServer(http.FS(content)), nil
}
