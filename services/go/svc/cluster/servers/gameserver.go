package servers

import (
	"context"
	_ "embed"
	"time"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type GameServer struct {
	server.Server

	Id string
	// Another way for the client to refer to this server
	Alias string

	Entities []maps.Entity

	// Whether this map was in our assets (ie can we send it to the client)
	IsBuiltMap bool

	Hidden bool

	// The last time a client connected
	LastEvent time.Time

	Mutex deadlock.RWMutex
}

func (server *GameServer) GetEntities() []maps.Entity {
	server.Mutex.RLock()
	teleports := server.Entities
	server.Mutex.RUnlock()
	return teleports
}

// Whether this string is a reference to this server (either an alias or an id).
func (server *GameServer) IsReference(reference string) bool {
	return server.Id == reference || server.Alias == reference
}

func (server *GameServer) Reference() string {
	if server.Alias != "" {
		return server.Alias
	}
	return server.Id
}

func (server *GameServer) GetFormattedReference() string {
	reference := server.Reference()
	if server.Hidden {
		reference = "???"
	}
	return reference
}

func (server *GameServer) Logger() zerolog.Logger {
	return log.With().Str("server", server.Reference()).Logger()
}

func (server *GameServer) Shutdown() {
	server.Cancel()
}

func (server *GameServer) Start(ctx context.Context) {
	go server.Poll(ctx)
}
