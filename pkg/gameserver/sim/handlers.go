package sim

import (
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

// HandleMessages processes a batch of messages from the server.
func (c *Client) HandleMessages(messages []P.Message) {
	for _, msg := range messages {
		c.HandleMessage(msg)
	}
}

// HandleMessage processes a single server message, mirroring the C++
// client's parsemessages (client.cpp:1542).
func (c *Client) HandleMessage(msg P.Message) {
	switch m := msg.(type) {
	case P.ServerInfo:
		c.CN = m.Client
		c.Send(1, P.Connect{
			Name: c.Name,
		})

	case P.Welcome:
		// Connected

	case P.MapChange:
		c.State = playerstate.Dead
		c.Alive = false

	case P.SpawnState:
		if m.Client != c.CN {
			return
		}
		c.LifeSequence = m.EntityState.LifeSequence
		c.Health = m.EntityState.Health
		c.MaxHealth = m.EntityState.MaxHealth
		c.Armour = m.EntityState.Armour
		c.GunSelect = m.EntityState.Gunselect

		c.Send(1, P.SpawnRequest{
			LifeSequence: c.LifeSequence,
			GunSelect:    c.GunSelect,
		})

		c.State = playerstate.Alive
		c.Alive = true

	case P.Died:
		if m.Client == c.CN {
			c.State = playerstate.Dead
			c.Alive = false
			c.Deaths++
		}
		if m.Killer == c.CN {
			c.Frags = m.KillerFrags
		}

	case P.ForceDeath:
		if m.Client == c.CN {
			c.State = playerstate.Dead
			c.Alive = false
		}

	case P.Pong:
		rtt := c.millis - m.Cmillis
		c.Ping = (c.Ping*5 + rtt) / 6
		c.Send(1, P.ClientPing{Ping: c.Ping})

	case P.Spectator:
		if m.Client == c.CN {
			if m.Spectating {
				c.State = playerstate.Spectator
			} else {
				c.State = playerstate.Dead
			}
		}

	case P.Damage:
		if m.Client == c.CN {
			c.Armour = m.Armour
			c.Health = m.Health
		}
	}
}
