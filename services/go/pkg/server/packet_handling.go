package server

import (
	"fmt"
	"log"
	"strings"
	"time"

	p "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
	"github.com/cfoust/sour/pkg/server/protocol/mastermode"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/role"
	"github.com/cfoust/sour/pkg/server/protocol/weapon"
)

func mapVec(v p.Vec) *geom.Vector {
	return geom.NewVector(
		v.X,
		v.Y,
		v.Z,
	)
}

func mapHits(hits []p.Hit) []hit {
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
func isValidMessage(c *Client, networkMessageCode nmc.ID) bool {
	if networkMessageCode == nmc.Ping {
		return true
	}

	if !c.Joined {
		if c.AuthRequiredBecause > disconnectreason.None {
			return networkMessageCode == nmc.AuthAnswer
		}
		return networkMessageCode == nmc.TryJoin
	} else if networkMessageCode == nmc.TryJoin {
		return false
	}

	for _, soNMC := range nmc.ServerOnlyNMCs {
		if soNMC == networkMessageCode {
			return false
		}
	}

	return true
}

// parses a packet and decides what to do based on the network message code at the front of the packet
func (s *GameServer) HandlePacket(client *Client, channelID uint8, message p.Message) {
	// this implementation does not support channel 2 (for coop edit purposes) yet.
	if client == nil || 0 > channelID || channelID > 1 {
		return
	}

	if !client.Joined && channelID == 0 {
		return
	}

	packetType := message.Type()

	if !isValidMessage(client, nmc.ID(packetType)) {
		log.Println("invalid network message code", packetType, "from CN", client.CN)
		s.Disconnect(client, disconnectreason.MessageError)
		return
	}

	switch packetType {

	// channel 0 traffic

	case p.N_POS:
		msg := message.(*p.Pos)

		// client sending his position and movement in the world
		if client.State == playerstate.Alive {
			client.Positions.Publish(msg)
			o := msg.State.O
			client.Position = geom.NewVector(o.X, o.Y, o.Z)
		}
		return

	case p.N_JUMPPAD:
		msg := message.(*p.JumpPad)
		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	case p.N_TELEPORT:
		msg := message.(*p.Teleport)

		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	// channel 1 traffic

	case p.N_CONNECT:
		msg := message.(*p.Connect)
		s.TryJoin(client, msg.Name, int32(msg.Model), msg.AuthDescription, msg.AuthName)

	case p.N_SETMASTER:
		msg := message.(*p.SetMaster)
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

	case p.N_KICK:
		msg := message.(*p.Kick)

		cn := uint32(msg.Victim)

		victim := s.Clients.GetClientByCN(cn)
		if victim == nil {
			return
		}

		s.Kick(client, victim, msg.Reason)

	case p.N_MASTERMODE:
		msg := message.(*p.MasterMode)
		mm := mastermode.ID(msg.MasterMode)
		s.SetMasterMode(client, mm)

	case p.N_SPECTATOR:
		msg := message.(*p.Spectator)

		spectator := s.Clients.GetClientByCN(uint32(msg.Client))
		if spectator == nil {
			return
		}
		toggle := msg.Spectating

		if client.Role == role.None {
			// unprivileged clients can never change spec state of others
			if spectator != client {
				client.SendServerMessage(cubecode.Fail("you can't do that"))
				return
			}
			// unprivileged clients can not unspec themselves in mm>=2
			if client.State == playerstate.Spectator && s.MasterMode >= mastermode.Locked {
				client.SendServerMessage(cubecode.Fail("you can't do that"))
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
		s.Clients.Broadcast(p.Spectator{int(spectator.CN), toggle})

	case p.N_MAPVOTE:
		msg := message.(*p.MapVote)

		mapname := msg.Map
		if mapname == "" {
			mapname = s.Map
		}

		modeID := gamemode.ID(msg.Mode)

		if !gamemode.Valid(modeID) {
			client.SendServerMessage(cubecode.Fail(fmt.Sprintf("%s is not implemented on this server", modeID)))
			log.Println("invalid gamemode", modeID, "requested")
			return
		}

		if s.MasterMode < mastermode.Veto {
			client.SendServerMessage(cubecode.Fail("this server does not support map voting"))
			return
		}

		if client.Role < role.Master {
			client.SendServerMessage(cubecode.Fail("you can't do that"))
			return
		}

		s.StartGame(s.StartMode(modeID), mapname)
		s.SendServerMessage(fmt.Sprintf("%s forced %s on %s", s.Clients.UniqueName(client), modeID, mapname))
		log.Println(client, "forced", modeID, "on", mapname)

	case p.N_PING:
		msg := message.(*p.Ping)

		// client pinging server → send pong
		client.Send(p.Pong{msg.Cmillis})

	case p.N_CLIENTPING:
		msg := message.(*p.ClientPing)

		// client sending the amount of lag he measured to the server → broadcast to other clients
		client.Ping = int32(msg.Ping)
		client.Packets.Publish(p.ClientPing{int(client.Ping)})

	case p.N_TEXT:
		msg := message.(*p.Text).Text

		// client sending chat message → broadcast to other clients
		if strings.HasPrefix(msg, "#") {
			s.Commands.Handle(client, msg[1:])
		} else {
			client.Packets.Publish(p.Text{msg})
		}

	case p.N_SAYTEAM:
		// client sending team chat message → pass on to team immediately
		msg := message.(*p.SayTeam).Text
		s.Clients.SendToTeam(client, p.SayTeam{msg})

	case p.N_SWITCHNAME:
		msg := message.(*p.SwitchName)

		newName := cubecode.Filter(msg.Name, false)

		if len(newName) == 0 || len(newName) > 24 {
			return
		}

		client.Name = newName
		client.Packets.Publish(msg)

	case p.N_SWITCHTEAM:
		msg := message.(*p.SwitchTeam)

		teamName := msg.Team

		if client.Team.Name == teamName {
			return
		}

		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}

		teamMode.ChangeTeam(&client.Player, teamName, false)

	case p.N_SETTEAM:
		msg := message.(*p.SetTeam)

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

	case p.N_MAPCRC:
		//msg := message.(*p.MapCRC)
		// client sends crc hash of his map file
		// TODO
		//clientMapName := p.GetString()
		//clientMapCRC := p.GetInt32()
		log.Println("todo: MAPCRC")

	case p.N_TRYSPAWN:
		if !client.Joined || client.State != playerstate.Dead || !client.LastSpawnAttempt.IsZero() || !s.GameMode.CanSpawn(&client.Player) {
			return
		}
		s.Spawn(client)
		client.Send(p.SpawnState{int(client.CN), client.ToWire()})

	case p.N_SPAWN:
		msg := message.(*p.SpawnRequest)
		s.ConfirmSpawn(client, int32(msg.LifeSequence), int32(msg.GunSelect))

	case p.N_GUNSELECT:
		msg := message.(*p.GunSelect)
		requested := weapon.ID(msg.GunSelect)
		selected, ok := client.SelectWeapon(requested)
		if !ok {
			break
		}
		client.Packets.Publish(p.GunSelect{int(selected.ID)})

	case p.N_SHOOT:
		msg := message.(*p.Shoot)

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

	case p.N_EXPLODE:
		msg := message.(*p.Explode)
		wpn := weapon.ByID(weapon.ID(msg.Gun))
		s.HandleExplode(client, int32(msg.Cmillis), wpn, int32(msg.Id), mapHits(msg.Hits))

	case p.N_SUICIDE:
		s.GameMode.HandleFrag(&client.Player, &client.Player)

	case p.N_SOUND:
		msg := message.(*p.Sound)
		client.Packets.Publish(msg)

	case p.N_PAUSEGAME:
		msg := message.(*p.PauseGame)
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

	case p.N_SERVCMD:
		msg := message.(*p.ServCMD)
		s.Commands.Handle(client, msg.Command)

	default:
		handled := false
		if mode, ok := s.GameMode.(game.HandlesPackets); ok {
			handled = mode.HandlePacket(&client.Player, message)
		}
		if !handled {
			log.Println("unhandled message %s", message.Type().String(), "received on channel", channelID)
			return
		}
	}
}
