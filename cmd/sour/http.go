package main

import (
	"net/http"
	"strings"
)

// SkipIndex is an http.Handler that disables the browser cache for .source
// files.
func SkipIndex(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set cache-control to none if url contains ".source"
		if strings.Contains(r.URL.Path, ".source") {
			w.Header().Set("Cache-Control", "no-store")
		}

		h.ServeHTTP(w, r)
	})
}
