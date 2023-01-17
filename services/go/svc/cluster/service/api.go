package service

import (
	"context"
	"net/http"
	"regexp"

	"github.com/go-redis/redis/v9"
)

var (
	DEMO_PATH_REGEX = regexp.MustCompile(`^/api/demo/([\w-]+)$`)
)

func (c *Cluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	matches := DEMO_PATH_REGEX.FindStringSubmatch(r.URL.Path)
	if len(matches) == 2 {
		id := matches[1]

		demo, err := c.GetDemo(context.Background(), id)
		if err == redis.Nil {
			w.WriteHeader(404)
			return
		}

		header := w.Header()
		header.Add("Content-Type", "application/octet-stream")
		w.Write(demo)
		return
	}

	w.WriteHeader(400)
}
