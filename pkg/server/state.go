package server

import (
	"time"

	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/protocol/mastermode"
)

type State struct {
	Clock      game.Clock
	MasterMode mastermode.ID
	GameMode   game.Mode
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}
