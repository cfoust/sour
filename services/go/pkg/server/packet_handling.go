package server

import (
	"fmt"
	"log"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
	"github.com/cfoust/sour/pkg/server/protocol/mastermode"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/role"
	"github.com/cfoust/sour/pkg/server/protocol/weapon"
)

func mapVec(v P.Vec) *geom.Vector {
	return geom.NewVector(
		v.X,
		v.Y,
		v.Z,
	)
}

func mapHits(hits []P.Hit) []hit {
	result := make([]hit, 0)
	for _, hit_ := range hits {
		result = append(
			result,
			hit{
				uint32(hit_.Target),
				int32(hit_.LifeSequence),
				hit_.Distance,
				int32(hit_.Rays),
				mapVec(hit_.Direction),
			},
		)
	}

	return result
}

// checks if the client is allowed to send a certain type of message to us.
func isValidMessage(c *Client, code P.MessageCode) bool {
	if code == P.N_PING {
		return true
	}

	if !c.Joined {
		if c.AuthRequiredBecause > disconnectreason.None {
			return code == P.N_AUTHANS
		}
		return code == P.N_CONNECT
	}

	if P.IsServerOnly(code) {
		return false
	}

	return true
}

// parses a packet and decides what to do based on the network message code at the front of the packet
func (s *Server) HandlePacket(client *Client, channelID uint8, message P.Message) {
	// this implementation does not support channel 2 (for coop edit purposes) yet.
	if client == nil || 0 > channelID || channelID > 1 {
		return
	}

	if !client.Joined && channelID == 0 {
		return
	}

	packetType := message.Type()

	if !isValidMessage(client, packetType) {
		log.Println("invalid network message code", packetType, "from CN", client.CN)
		s.Disconnect(client, disconnectreason.MessageError)
		return
	}

	if P.IsEditMessage(message.Type()) {
		if s.GameMode.ID() != gamemode.CoopEdit {
			return
		}

		s.Clients.Broadcast(message)

		s.Edits.Publish(MapEdit{
			Client:  client.SessionID,
			Message: message,
		})
		return
	}

	switch packetType {

	// channel 0 traffic
	case P.N_POS:
		msg := message.(P.Pos)

		// client sending his position and movement in the world
		if client.State == playerstate.Alive {
			msg.State.LifeSequence = client.LifeSequence
			client.Positions.Publish(msg)
			client.Position = mapVec(msg.State.O)
		}
		return

	case P.N_JUMPPAD:
		msg := message.(P.JumpPad)
		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	case P.N_TELEPORT:
		msg := message.(P.Teleport)

		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	case P.N_ADDBOT, P.N_DELBOT:
		client.Message("bots currently not supported")

	// channel 1 traffic
	case P.N_CONNECT:
		msg := message.(P.Connect)
		s.TryJoin(client, msg.Name, int32(msg.Model), msg.AuthDescription, msg.AuthName)

	case P.N_SETMASTER:
		msg := message.(P.SetMaster)
		cn := uint32(msg.Client)

		switch msg.Master {
		case 0:
			s.setRole(client, cn, role.None)
			for domain := range client.Authentications {
				delete(client.Authentications, domain)
			}
		default:
			s.setRole(client, cn, role.Master)
		}

	case P.N_KICK:
		msg := message.(P.Kick)

		cn := uint32(msg.Victim)

		victim := s.Clients.GetClientByCN(cn)
		if victim == nil {
			return
		}

		s.Kick(client, victim, msg.Reason)

	case P.N_MASTERMODE:
		msg := message.(P.MasterMode)
		mm := mastermode.ID(msg.MasterMode)
		s.SetMasterMode(client, mm)

	case P.N_SPECTATOR:
		msg := message.(P.Spectator)

		spectator := s.Clients.GetClientByCN(uint32(msg.Client))
		if spectator == nil {
			return
		}
		toggle := msg.Spectating

		if client.Role == role.None {
			// unprivileged clients can never change spec state of others
			if spectator != client {
				client.Message(cubecode.Fail("you can't do that"))
				return
			}
			// unprivileged clients can not unspec themselves in mm>=2
			if client.State == playerstate.Spectator && s.MasterMode >= mastermode.Locked {
				client.Message(cubecode.Fail("you can't do that"))
				return
			}
		}
		if (spectator.State == playerstate.Spectator) == !toggle {
			// nothing to do
			return
		}

		if toggle {
			if client.State == playerstate.Alive {
				s.GameMode.HandleFrag(&spectator.Player, &spectator.Player)
			}
			s.GameMode.Leave(&spectator.Player)
			s.Clock.Leave(&spectator.Player)
			spectator.State = playerstate.Spectator
		} else {
			spectator.State = playerstate.Dead
			if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
				teamedMode.Join(&spectator.Player)
			}
			// todo: checkmap
		}
		s.Clients.Broadcast(P.Spectator{int32(spectator.CN), toggle})

	case P.N_MAPVOTE:
		msg := message.(P.MapVote)

		mapname := msg.Map
		if mapname == "" {
			mapname = s.Map
		}

		modeID := gamemode.ID(msg.Mode)

		if !gamemode.Valid(modeID) {
			client.Message(cubecode.Fail(fmt.Sprintf("%s is not implemented on this server", modeID)))
			log.Println("invalid gamemode", modeID, "requested")
			return
		}

		if s.MasterMode < mastermode.Veto {
			client.Message(cubecode.Fail("this server does not support map voting"))
			return
		}

		if client.Role < role.Master {
			client.Message(cubecode.Fail("you can't do that"))
			return
		}

		s.StartGame(s.StartMode(modeID), mapname)
		s.Message(fmt.Sprintf("%s forced %s on %s", s.Clients.UniqueName(client), modeID, mapname))
		log.Println(client, "forced", modeID, "on", mapname)

	case P.N_PING:
		msg := message.(P.Ping)

		// client pinging server → send pong
		client.Send(P.Pong{msg.Cmillis})

	case P.N_CLIENTPING:
		msg := message.(P.ClientPing)

		// client sending the amount of lag he measured to the server → broadcast to other clients
		client.Ping = int32(msg.Ping)
		client.Packets.Publish(P.ClientPing{int32(client.Ping)})

	case P.N_TEXT:
		client.Packets.Publish(message.(P.Text))

	case P.N_SAYTEAM:
		// client sending team chat message → pass on to team immediately
		msg := message.(P.SayTeam).Text
		s.Clients.SendToTeam(client, P.SayTeam{msg})

	case P.N_SWITCHMODEL:
		msg := message.(P.SwitchModel)
		client.Model = msg.Model
		s.Broadcast(msg)

	case P.N_SWITCHNAME:
		msg := message.(P.SwitchName)

		newName := cubecode.Filter(msg.Name, false)

		if len(newName) == 0 || len(newName) > 24 {
			return
		}

		client.Name = newName
		client.Packets.Publish(msg)

	case P.N_SWITCHTEAM:
		msg := message.(P.SwitchTeam)

		teamName := msg.Team

		if client.Team.Name == teamName {
			return
		}

		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}

		teamMode.ChangeTeam(&client.Player, teamName, false)

	case P.N_SETTEAM:
		msg := message.(P.SetTeam)

		victim := s.Clients.GetClientByCN(uint32(msg.Client))
		teamName := msg.Team

		if victim == nil || victim.Team.Name == teamName || client.Role == role.None {
			return
		}

		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}

		teamMode.ChangeTeam(&victim.Player, teamName, true)

	case P.N_MAPCRC:
		// TODO

	case P.N_TRYSPAWN:
		if !client.Joined || client.State != playerstate.Dead || !client.LastSpawnAttempt.IsZero() || !s.GameMode.CanSpawn(&client.Player) {
			return
		}
		s.Spawn(client)
		client.Send(P.SpawnState{int32(client.CN), client.ToWire()})

	case P.N_SPAWN:
		msg := message.(P.SpawnRequest)
		s.ConfirmSpawn(client, int32(msg.LifeSequence), int32(msg.GunSelect))

	case P.N_GUNSELECT:
		msg := message.(P.GunSelect)
		requested := weapon.ID(msg.GunSelect)
		selected, ok := client.SelectWeapon(requested)
		if !ok {
			break
		}
		client.Packets.Publish(P.GunSelect{int32(selected.ID)})

	case P.N_TAUNT:
		client.Packets.Publish(message)

	case P.N_SHOOT:
		msg := message.(P.Shoot)

		wpn := weapon.ByID(weapon.ID(msg.Gun))
		if time.Now().Before(client.GunReloadEnd) || client.Ammo[wpn.ID] <= 0 {
			return
		}

		from := mapVec(msg.From)
		to := mapVec(msg.To)

		if dist := geom.Distance(from, to); dist > wpn.Range+1.0 {
			log.Println("shot distance out of weapon's range: distane =", dist, "range =", wpn.Range+1)
			return
		}

		s.HandleShoot(
			client,
			wpn,
			int32(msg.Id),
			from,
			to,
			mapHits(msg.Hits),
		)

	case P.N_EXPLODE:
		msg := message.(P.Explode)
		wpn := weapon.ByID(weapon.ID(msg.Gun))
		s.HandleExplode(client, int32(msg.Cmillis), wpn, int32(msg.Id), mapHits(msg.Hits))

	case P.N_SUICIDE:
		s.GameMode.HandleFrag(&client.Player, &client.Player)

	case P.N_SOUND:
		msg := message.(P.Sound)
		client.Packets.Publish(msg)

	case P.N_PAUSEGAME:
		msg := message.(P.PauseGame)
		if s.MasterMode < mastermode.Locked {
			if client.Role == role.None {
				return
			}
		}
		if msg.Paused {
			s.Clock.Pause(&client.Player)
		} else {
			s.Clock.Resume(&client.Player)
		}

	default:
		handled := false
		if mode, ok := s.GameMode.(game.HandlesPackets); ok {
			handled = mode.HandlePacket(&client.Player, message)
		}
		if !handled {
			log.Println("unhandled message", message.Type().String(), "received on channel", channelID)
			return
		}
	}
}
