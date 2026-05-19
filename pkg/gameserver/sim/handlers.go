package sim

import (
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
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
		c.Map = m.Name
		c.GameMode = gamemode.ID(m.Mode)

	case P.TimeUp:
		c.TimeLeft = m.Remaining

	case P.SetTeam:
		if m.Client == c.CN {
			c.Team = m.Team
		}

	case P.InitClient:
		if c.KnownPlayers == nil {
			c.KnownPlayers = make(map[int32]string)
		}
		c.KnownPlayers[m.Client] = m.Name

	case P.ClientDisconnected:
		if c.KnownPlayers != nil {
			delete(c.KnownPlayers, m.Client)
		}
		if c.ResumeFrags != nil {
			delete(c.ResumeFrags, m.Client)
		}

	case P.Resume:
		if c.ResumeFrags == nil {
			c.ResumeFrags = make(map[int32]int32)
		}
		for _, cs := range m.Clients {
			c.ResumeFrags[cs.Id] = cs.Frags
		}

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

	// CTF messages
	case P.ServerInitFlags:
		c.TeamScore[0] = m.Scores[0].Score
		c.TeamScore[1] = m.Scores[1].Score
		c.Flags = make([]FlagInfo, len(m.Flags))
		for i, f := range m.Flags {
			c.Flags[i] = FlagInfo{
				Version: f.Version,
				Owner:   f.Owner,
				Dropped: f.Dropped,
			}
		}

	case P.ServerTakeFlag:
		if int(m.Flag) < len(c.Flags) {
			c.Flags[m.Flag].Version = m.Version
			c.Flags[m.Flag].Owner = m.Client
			c.Flags[m.Flag].Dropped = false
		}

	case P.DropFlag:
		if int(m.Flag) < len(c.Flags) {
			c.Flags[m.Flag].Version = m.Version
			c.Flags[m.Flag].Owner = -1
			c.Flags[m.Flag].Dropped = true
		}

	case P.ReturnFlag:
		if int(m.Flag) < len(c.Flags) {
			c.Flags[m.Flag].Version = m.Version
			c.Flags[m.Flag].Owner = -1
			c.Flags[m.Flag].Dropped = false
		}

	case P.ResetFlag:
		if int(m.Flag) < len(c.Flags) {
			c.Flags[m.Flag].Version = m.Version
			c.Flags[m.Flag].Owner = -1
			c.Flags[m.Flag].Dropped = false
		}
		// Update team score
		if m.Team >= 1 && m.Team <= 2 {
			c.TeamScore[m.Team-1] = m.Score
		}

	case P.ScoreFlag:
		if m.Team >= 1 && m.Team <= 2 {
			c.TeamScore[m.Team-1] = m.Score
		}
		// The goal flag gets returned
		if int(m.Goalflag) < len(c.Flags) {
			c.Flags[m.Goalflag].Version = m.Goalversion
		}
		// The relay flag (enemy flag being carried) gets returned
		if int(m.Relayflag) < len(c.Flags) {
			c.Flags[m.Relayflag].Version = m.Relayversion
			c.Flags[m.Relayflag].Owner = -1
			c.Flags[m.Relayflag].Dropped = false
		}
	}
}
