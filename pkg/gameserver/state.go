package gameserver

import (
	"time"

	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/protocol/mastermode"
)

type State struct {
	Clock      game.Clock
	MasterMode mastermode.ID
	GameMode   game.Mode
	Map        string
	UpSince    time.Time
	NumClients func() int // number of clients connected
}
