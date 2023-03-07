package game

import (
	"log"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
)

type CoopEdit struct {
	*teamlessMode
	noSpawnWait
	ffaSpawnState
	*handlesPickups

	s Server
}

func (*CoopEdit) ID() gamemode.ID { return gamemode.CoopEdit }

// assert interface implementations at compile time
var (
	_ Mode       = &CoopEdit{}
	_ PickupMode = &CoopEdit{}
)

func NewCoopEdit(s Server) *CoopEdit {
	return &CoopEdit{
		handlesPickups: handlingPickups(s),
		s:              s,
	}
}

func (m *CoopEdit) HandlePacket(p *Player, message P.Message) bool {
	switch message.Type() {
	case P.N_EDITMODE:
		msg := message.(P.EditMode)
		enabled := msg.Enabled

		if enabled && p.State == playerstate.Spectator {
			return true
		}

		if !enabled && p.State != playerstate.Editing {
			return true
		}

		if enabled {
			p.EditState = p.State
			p.State = playerstate.Editing

			// TODO
			//ci->events.setsize(0);
			//ci->state.rockets.reset();
			//ci->state.grenades.reset();
		} else {
			p.State = p.EditState
		}
	default:
		log.Println("received unrelated packet", message)
		return false
	}

	return false
}
