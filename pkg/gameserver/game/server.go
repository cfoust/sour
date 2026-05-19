package game

import (
	"github.com/cfoust/sour/pkg/game/protocol"
)

type Server interface {
	GameDuration() int64 // match length in milliseconds
	GameClock() int64    // current server time in milliseconds
	Broadcast(messages ...protocol.Message)
	Message(message string)
	Intermission()
	ForEachPlayer(func(*Player))
	UniqueName(*Player) string
	NumberOfPlayers() int
}
