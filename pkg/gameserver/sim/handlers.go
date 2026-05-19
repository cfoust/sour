package sim

import (
	"sync/atomic"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

// HandleMessages processes a batch of messages from the server, faithfully
// mirroring the C++ client's parsemessages function (client.cpp:1530).
func (c *Client) HandleMessages(messages []P.Message) {
	for _, msg := range messages {
		c.HandleMessage(msg)
	}
}

// HandleMessage processes a single server message. The cases here correspond
// to the switch(type) in parsemessages (client.cpp:1542).
func (c *Client) HandleMessage(msg P.Message) {
	switch m := msg.(type) {

	// client.cpp:1546 - N_SERVINFO: welcome message from the server.
	// On receiving this, the C++ client sends N_CONNECT (sendintro,
	// client.cpp:1323-1349).
	case P.ServerInfo:
		c.Mu.Lock()
		c.CN = m.Client
		c.Mu.Unlock()
		// Send N_CONNECT (mirrors sendintro)
		c.Send(1, P.Connect{
			Name:            c.Name,
			Model:           0,
			Password:        "",
			AuthDescription: "",
			AuthName:        "",
		})

	// client.cpp:1564 - N_WELCOME: marks that we're fully connected.
	case P.Welcome:
		atomic.StoreInt32(&c.connected, 1)

	// client.cpp:1636 - N_MAPCHANGE: server changed the map.
	case P.MapChange:
		// The C++ client calls changemapserv which resets game state.
		// We reset our alive state so we can re-enter the spawn flow.
		c.Mu.Lock()
		c.State = playerstate.Dead
		c.Mu.Unlock()
		c.aliveMu.Lock()
		c.alive = make(chan struct{})
		c.aliveMu.Unlock()

	// client.cpp:1753 - N_SPAWNSTATE: server tells us how to spawn.
	// The C++ client responds by setting state to CS_ALIVE and sending
	// N_SPAWN to confirm (client.cpp:1775).
	case P.SpawnState:
		c.Mu.Lock()
		if m.Client != c.CN {
			c.Mu.Unlock()
			return
		}
		c.LifeSequence = m.EntityState.LifeSequence
		c.Health = m.EntityState.Health
		c.MaxHealth = m.EntityState.MaxHealth
		c.Armour = m.EntityState.Armour
		c.GunSelect = m.EntityState.Gunselect
		lifeSeq := c.LifeSequence
		gunSel := c.GunSelect
		c.State = playerstate.Alive
		c.Mu.Unlock()

		// Confirm spawn (client.cpp:1775)
		c.Send(1, P.SpawnRequest{
			LifeSequence: lifeSeq,
			GunSelect:    gunSel,
		})

		c.aliveMu.Lock()
		// Signal that we're alive
		select {
		case <-c.alive:
			// Already closed (stale), make a new one then close it
			c.alive = make(chan struct{})
		default:
		}
		close(c.alive)
		c.aliveMu.Unlock()

		// Signal joined on first spawn
		select {
		case <-c.joined:
		default:
			close(c.joined)
		}

	// client.cpp:1832 - N_DIED: a player died.
	case P.Died:
		c.Mu.Lock()
		if m.Client == c.CN {
			c.State = playerstate.Dead
			c.Deaths++
		}
		if m.Killer == c.CN {
			c.Frags = m.KillerFrags
		}
		c.Mu.Unlock()

		if m.Client == c.CN {
			c.aliveMu.Lock()
			c.alive = make(chan struct{})
			c.aliveMu.Unlock()
		}

	// client.cpp:1646 - N_FORCEDEATH.
	case P.ForceDeath:
		if m.Client == c.CN {
			c.Mu.Lock()
			c.State = playerstate.Dead
			c.Mu.Unlock()
			c.aliveMu.Lock()
			c.alive = make(chan struct{})
			c.aliveMu.Unlock()
		}

	// client.cpp:2053 - N_PONG: server responding to our ping.
	// player1->ping = (player1->ping*5+totalmillis-getint(p))/6
	// addmsg(N_CLIENTPING, "i", player1->ping);
	case P.Pong:
		rtt := c.Totalmillis() - m.Cmillis
		c.Mu.Lock()
		c.Ping = (c.Ping*5 + rtt) / 6
		ping := c.Ping
		c.Mu.Unlock()
		c.Send(1, P.ClientPing{Ping: ping})

	// client.cpp:2138 - N_SPECTATOR.
	case P.Spectator:
		if m.Client == c.CN {
			c.Mu.Lock()
			if m.Spectating {
				c.State = playerstate.Spectator
			} else {
				c.State = playerstate.Dead
			}
			c.Mu.Unlock()
		}

	// client.cpp:1805 - N_DAMAGE.
	case P.Damage:
		if m.Client == c.CN {
			c.Mu.Lock()
			c.Armour = m.Armour
			c.Health = m.Health
			c.Mu.Unlock()
		}

	// Everything below is either visual-only or irrelevant for sim.
	case P.TimeUp, P.Resume, P.InitClient, P.ClientDisconnected,
		P.TeamInfo, P.SetTeam, P.ShotFX, P.ExplodeFX, P.HitPush,
		P.ItemSpawn, P.ItemAck, P.ItemList, P.Sound, P.Taunt,
		P.Text, P.SayTeam, P.ServerMessage, P.CurrentMaster,
		P.MasterMode, P.PauseGame, P.GameSpeed, P.ClientPing,
		P.GunSelect, P.ServerInitFlags, P.DropFlag, P.ScoreFlag,
		P.ReturnFlag, P.ServerTakeFlag, P.ResetFlag, P.InvisFlag,
		P.Announce, P.ClientPacket, P.SpawnResponse, P.SwitchName,
		P.SwitchModel, P.EditMode, P.NewMap, P.MapCRC:
		// No-op.
	}
}
