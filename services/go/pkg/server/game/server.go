package game

import (
	"time"

	"github.com/cfoust/sour/pkg/game/protocol"
)

type Server interface {
	GameDuration() time.Duration
	Broadcast(messages ...protocol.Message)
	Message(message string)
	Intermission()
	ForEachPlayer(func(*Player))
	UniqueName(*Player) string
	NumberOfPlayers() int
}
