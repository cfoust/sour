package gameserver

import (
	"fmt"
	"log"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/geom"
	"github.com/cfoust/sour/pkg/gameserver/protocol/cubecode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/mastermode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
)

func mapVec(v P.Vec) *geom.Vector {
	return geom.NewVector(v.X, v.Y, v.Z)
}

func mapHits(hits []P.Hit) []hit {
	result := make([]hit, 0)
	for _, hit_ := range hits {
		result = append(result, hit{
			uint32(hit_.Target),
			int32(hit_.LifeSequence),
			hit_.Distance,
			int32(hit_.Rays),
			mapVec(hit_.Direction),
		})
	}
	return result
}

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

func (s *Server) HandlePacket(client *Client, channelID uint8, message P.Message) {
	if client == nil || channelID > 1 {
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
		s.Broadcast(message)
		return
	}

	switch packetType {

	case P.N_POS:
		msg := message.(P.Pos)
		if client.State == playerstate.Alive {
			msg.State.LifeSequence = client.LifeSequence
			s.relay.SetPosition(client.CN, msg)
			client.Position = mapVec(msg.State.O)
		}
		return

	case P.N_JUMPPAD:
		msg := message.(P.JumpPad)
		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg, &s.out)
		}

	case P.N_TELEPORT:
		msg := message.(P.Teleport)
		if client.State == playerstate.Alive {
			s.relay.FlushPositionAndSend(client.CN, msg, &s.out)
		}

	case P.N_ADDBOT, P.N_DELBOT:
		client.Message("bots currently not supported")

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
		victim := s.Clients.GetClientByCN(uint32(msg.Victim))
		if victim == nil {
			return
		}
		s.Kick(client, victim, msg.Reason)

	case P.N_MASTERMODE:
		msg := message.(P.MasterMode)
		s.SetMasterMode(client, mastermode.ID(msg.MasterMode))

	case P.N_SPECTATOR:
		msg := message.(P.Spectator)
		spectator := s.Clients.GetClientByCN(uint32(msg.Client))
		if spectator == nil {
			return
		}
		toggle := msg.Spectating

		if client.Role == role.None {
			if spectator != client {
				client.Message(cubecode.Fail("you can't do that"))
				return
			}
			if client.State == playerstate.Spectator && s.MasterMode >= mastermode.Locked {
				client.Message(cubecode.Fail("you can't do that"))
				return
			}
		}
		alreadySpec := spectator.State == playerstate.Spectator
		if (alreadySpec && toggle) || (!alreadySpec && !toggle) {
			return
		}

		if toggle {
			if client.State == playerstate.Alive {
				s.GameMode.HandleFrag(s.gameClock, &spectator.Player, &spectator.Player)
			}
			s.GameMode.Leave(&spectator.Player)
			s.State.Clock.Leave(&spectator.Player)
			spectator.State = playerstate.Spectator
		} else {
			spectator.State = playerstate.Dead
			if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
				teamedMode.Join(&spectator.Player)
			}
		}
		s.Broadcast(P.Spectator{Client: int32(spectator.CN), Spectating: toggle})

	case P.N_MAPVOTE:
		msg := message.(P.MapVote)
		mapname := msg.Map
		if mapname == "" {
			mapname = s.Map
		}
		modeID := gamemode.ID(msg.Mode)
		if !gamemode.Valid(modeID) {
			client.Message(cubecode.Fail(fmt.Sprintf("%s is not implemented on this server", modeID)))
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

	case P.N_PING:
		msg := message.(P.Ping)
		s.out.Send(client.SessionID, P.Pong{Cmillis: msg.Cmillis})

	case P.N_CLIENTPING:
		msg := message.(P.ClientPing)
		client.Ping = int32(msg.Ping)
		s.relay.AddPacket(client.CN, P.ClientPing{Ping: int32(client.Ping)})

	case P.N_TEXT:
		s.relay.AddPacket(client.CN, message.(P.Text))

	case P.N_SAYTEAM:
		msg := message.(P.SayTeam).Text
		s.Clients.SendToTeam(client, &s.out, P.SayTeam{Text: msg})

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
		s.relay.AddPacket(client.CN, msg)

	case P.N_SWITCHTEAM:
		msg := message.(P.SwitchTeam)
		if client.Team.Name == msg.Team {
			return
		}
		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}
		teamMode.ChangeTeam(&client.Player, msg.Team, false)

	case P.N_SETTEAM:
		msg := message.(P.SetTeam)
		victim := s.Clients.GetClientByCN(uint32(msg.Client))
		if victim == nil || victim.Team.Name == msg.Team || client.Role == role.None {
			return
		}
		teamMode, ok := s.GameMode.(game.TeamMode)
		if !ok {
			return
		}
		teamMode.ChangeTeam(&victim.Player, msg.Team, true)

	case P.N_MAPCRC:
		// TODO

	case P.N_TRYSPAWN:
		if !client.Joined || client.State != playerstate.Dead || client.LastSpawnAttempt != -1 || !s.GameMode.CanSpawn(s.gameClock, &client.Player) {
			return
		}
		s.Spawn(client)
		s.out.Send(client.SessionID, P.SpawnState{Client: int32(client.CN), EntityState: client.ToWire()})

	case P.N_SPAWN:
		msg := message.(P.SpawnRequest)
		s.ConfirmSpawn(client, int32(msg.LifeSequence), int32(msg.GunSelect))

	case P.N_GUNSELECT:
		msg := message.(P.GunSelect)
		selected, ok := client.SelectWeapon(weapon.ID(msg.GunSelect))
		if !ok {
			break
		}
		s.relay.AddPacket(client.CN, P.GunSelect{GunSelect: int32(selected.ID)})

	case P.N_TAUNT:
		s.relay.AddPacket(client.CN, message)

	case P.N_SHOOT:
		msg := message.(P.Shoot)
		wpn := weapon.ByID(weapon.ID(msg.Gun))
		if s.gameClock < client.GunReloadEnd || client.Ammo[wpn.ID] <= 0 {
			return
		}
		from := mapVec(msg.From)
		to := mapVec(msg.To)
		if dist := geom.Distance(from, to); dist > wpn.Range+1.0 {
			return
		}
		s.HandleShoot(client, wpn, int32(msg.Id), from, to, mapHits(msg.Hits))

	case P.N_EXPLODE:
		msg := message.(P.Explode)
		wpn := weapon.ByID(weapon.ID(msg.Gun))
		s.HandleExplode(client, int32(msg.Cmillis), wpn, int32(msg.Id), mapHits(msg.Hits))

	case P.N_SUICIDE:
		s.GameMode.HandleFrag(s.gameClock, &client.Player, &client.Player)

	case P.N_SOUND:
		s.relay.AddPacket(client.CN, message.(P.Sound))

	case P.N_PAUSEGAME:
		msg := message.(P.PauseGame)
		if s.MasterMode < mastermode.Locked {
			if client.Role == role.None {
				return
			}
		}
		if msg.Paused {
			s.State.Clock.Pause(s.gameClock, &client.Player)
		} else {
			s.State.Clock.Resume(s.gameClock, &client.Player)
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
