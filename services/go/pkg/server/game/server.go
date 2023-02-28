package game

import (
	"time"

	"github.com/cfoust/sour/pkg/server/protocol/nmc"
)

type Server interface {
	GameDuration() time.Duration
	Broadcast(nmc.ID, ...interface{})
	Intermission()
	ForEachPlayer(func(*Player))
	UniqueName(*Player) string
	NumberOfPlayers() int
}
