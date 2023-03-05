package server

import (
	"fmt"
	"log"
	"strings"

	"github.com/cfoust/sour/pkg/game/protocol"
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
func (s *GameServer) HandlePacket(client *Client, channelID uint8, message protocol.Message) {
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

	case protocol.N_POS:
		msg := message.(*protocol.Pos)

		// client sending his position and movement in the world
		if client.State == playerstate.Alive {
			client.Positions.Publish(msg)
			o := msg.State.O
			client.Position = geom.NewVector(o.X, o.Y, o.Z)
		}
		return

	case protocol.N_JUMPPAD:
		msg := message.(*protocol.JumpPad)
		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	case protocol.N_TELEPORT:
		msg := message.(*protocol.Teleport)

		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg)
		}

	// channel 1 traffic

	case protocol.N_CONNECT:
		msg := message.(*protocol.Connect)
		s.TryJoin(client, msg.Name, int32(msg.Model), msg.AuthDescription, msg.AuthName)

	case protocol.N_SETMASTER:
		msg := message.(*protocol.SetMaster)
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

	case protocol.N_KICK:
		msg := message.(*protocol.Kick)

		cn := uint32(msg.Victim)

		victim := s.Clients.GetClientByCN(cn)
		if victim == nil {
			return
		}

		s.Kick(client, victim, msg.Reason)

	case protocol.N_MASTERMODE:
		msg := message.(*protocol.MasterMode)
		mm := mastermode.ID(msg.MasterMode)
		s.SetMasterMode(client, mm)

	case protocol.N_SPECTATOR:
		msg := message.(*protocol.Spectator)

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
		s.Clients.Broadcast(protocol.Spectator{int(spectator.CN), toggle})

	case protocol.N_MAPVOTE:
		msg := message.(*protocol.MapVote)

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

	case protocol.N_PING:
		msg := message.(*protocol.Ping)

		// client pinging server → send pong
		client.Send(protocol.Pong{msg.Cmillis})

	case protocol.N_CLIENTPING:
		msg := message.(*protocol.ClientPing)

		// client sending the amount of lag he measured to the server → broadcast to other clients
		client.Ping = int32(msg.Ping)
		client.Packets.Publish(protocol.ClientPing{int(client.Ping)})

	case protocol.N_TEXT:
		msg := message.(*protocol.Text).Text

		// client sending chat message → broadcast to other clients
		if strings.HasPrefix(msg, "#") {
			s.Commands.Handle(client, msg[1:])
		} else {
			client.Packets.Publish(protocol.Text{msg})
		}

	case protocol.N_SAYTEAM:
		// client sending team chat message → pass on to team immediately
		msg := message.(*protocol.SayTeam).Text
		s.Clients.SendToTeam(client, protocol.SayTeam{msg})

	case protocol.N_SWITCHNAME:
		msg := message.(*protocol.SwitchName)

		newName := cubecode.Filter(msg.Name, false)

		if len(newName) == 0 || len(newName) > 24 {
			return
		}

		client.Name = newName
		client.Packets.Publish(msg)

	case protocol.N_SWITCHTEAM:
		msg := message.(*protocol.SwitchTeam)

		teamName := msg.Team

		if client.Team.Name == teamName {
			return
		}

		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}

		teamMode.ChangeTeam(&client.Player, teamName, false)

	case protocol.N_SETTEAM:
		msg := message.(*protocol.SetTeam)

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

	case protocol.N_MAPCRC:
		//msg := message.(*protocol.MapCRC)
		// client sends crc hash of his map file
		// TODO
		//clientMapName := p.GetString()
		//clientMapCRC := p.GetInt32()
		log.Println("todo: MAPCRC")

	case protocol.N_TRYSPAWN:
		if !client.Joined || client.State != playerstate.Dead || !client.LastSpawnAttempt.IsZero() || !s.GameMode.CanSpawn(&client.Player) {
			return
		}
		s.Spawn(client)
		client.Send(protocol.SpawnState{int(client.CN), client.ToWire()})

	case protocol.N_SPAWN:
		msg := message.(*protocol.SpawnRequest)
		s.ConfirmSpawn(client, int32(msg.LifeSequence), int32(msg.GunSelect))

	case nmc.ChangeWeapon:
		// player changing weapon
		_requested, ok := p.GetInt()
		if !ok {
			log.Println("could not read weapon ID from weapon change packet:", p)
			return
		}
		requested := weapon.ID(_requested)
		selected, ok := client.SelectWeapon(requested)
		if !ok {
			break
		}
		client.Packets.Publish(nmc.ChangeWeapon, selected.ID)

	case nmc.Shoot:
		wpn, id, from, to, hits, ok := parseShoot(client, &p)
		if !ok {
			return
		}
		s.HandleShoot(client, wpn, id, from, to, hits)

	case nmc.Explode:
		millis, wpn, id, hits, ok := parseExplode(client, &p)
		if !ok {
			return
		}
		s.HandleExplode(client, millis, wpn, id, hits)

	case nmc.Suicide:
		s.GameMode.HandleFrag(&client.Player, &client.Player)

	case nmc.Sound:
		sound, ok := p.GetInt()
		if !ok {
			log.Println("could not read sound ID from sound packet:", p)
			return
		}
		client.Packets.Publish(nmc.Sound, sound)

	case nmc.PauseGame:
		pause, ok := p.GetInt()
		if !ok {
			log.Println("could not read pause toggle from pause packet:", p)
			return
		}
		if s.MasterMode < mastermode.Locked {
			if client.Role == role.None {
				return
			}
		}
		if pause == 1 {
			s.Clock.Pause(&client.Player)
		} else {
			s.Clock.Resume(&client.Player)
		}

	case nmc.ServerCommand:
		cmd, ok := p.GetString()
		if !ok {
			log.Println("could not read command from server command packet:", p)
			return
		}
		s.Commands.Handle(client, cmd)

	default:
		handled := false
		if mode, ok := s.GameMode.(game.HandlesPackets); ok {
			handled = mode.HandlePacket(&client.Player, packetType, &p)
		}
		if !handled {
			log.Println("unhandled packet", packetType, p, "received on channel", channelID)
			return
		}
	}
}
