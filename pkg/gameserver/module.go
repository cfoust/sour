package gameserver

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/cfoust/sour/pkg/chanlock"
	G "github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/commands"
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
	"github.com/cfoust/sour/pkg/gameserver/relay"
	"github.com/cfoust/sour/pkg/utils"

	"github.com/rs/zerolog/log"
)

type ServerPacket struct {
	// Either the sender (if incoming) or the recipient (if outgoing)
	Session  uint32
	Channel  uint8
	Messages []P.Message
}

type MapEdit struct {
	Client  uint32
	Message P.Message
}

type Incoming <-chan ServerPacket
type Outgoing chan<- ServerPacket

type Server struct {
	utils.Session

	*Config
	*State
	relay *relay.Relay

	Description string

	Clients *ClientManager

	Commands *commands.CommandGroup[*Client]

	pendingMapChange *time.Timer
	rng              *rand.Rand

	incoming chan ServerPacket
	outgoing chan ServerPacket
	maps     chan string

	Broadcasts *utils.Topic[[]P.Message]
	Edits      *utils.Topic[MapEdit]

	// non-standard stuff
	KeepTeams       bool
	CompetitiveMode bool
	ReportStats     bool
}

func New(ctx context.Context, conf *Config) *Server {
	broadcasts := utils.NewTopic[[]P.Message]()

	clients := &ClientManager{
		broadcasts: broadcasts,
	}

	incoming := make(chan ServerPacket)
	outgoing := make(chan ServerPacket)

	s := &Server{
		Session:    utils.NewSession(ctx),
		Broadcasts: broadcasts,
		Commands:   commands.NewCommandGroup[*Client]("server", G.ColorBlue),
		Edits:      utils.NewTopic[MapEdit](),
		Config:     conf,
		State: &State{
			MasterMode: mastermode.Auth,
			UpSince:    time.Now(),
			NumClients: clients.GetNumClients,
		},
		relay:    relay.New(),
		Clients:  clients,
		incoming: incoming,
		outgoing: outgoing,
		maps:     make(chan string, 1),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return s
}

func (s *Server) Poll(ctx context.Context) {
	chanLock := chanlock.New()
	health := chanLock.Poll(s.Ctx())

	for {
		select {
		case <-s.Ctx().Done():
			return
		case <-health:
			continue
		case msg := <-s.incoming:
			client := s.Clients.GetClientByID(msg.Session)
			if client == nil {
				continue
			}

			for _, message := range msg.Messages {
				s.HandlePacket(client, msg.Channel, message)
			}
		}
	}
}

func (s *Server) Incoming() chan<- ServerPacket {
	return s.incoming
}

func (s *Server) Outgoing() <-chan ServerPacket {
	return s.outgoing
}

func (s *Server) ReceiveMaps() <-chan string {
	return s.maps
}

func (s *Server) GameDuration() time.Duration {
	return time.Duration(s.Config.MatchLength) * time.Second
}

func (s *Server) Connect(sessionId uint32) (*Client, <-chan bool) {
	existing := s.Clients.GetClientByID(sessionId)
	if existing != nil {
		log.Error().Msgf("client %d already connected")
		return nil, nil
	}

	connected := make(chan bool, 1)

	client := s.Clients.Add(sessionId, s.outgoing)
	client.connected = connected
	client.server = s
	client.Positions, client.Packets = s.relay.AddClient(client.CN, func(channel uint8, payload []P.Message) {
		s.outgoing <- ServerPacket{
			Session:  client.SessionID,
			Channel:  channel,
			Messages: payload,
		}
	})

	if client.Positions == nil {
		log.Error().Msgf("client %d had no channels")
		return nil, nil
	}

	client.Send(
		P.ServerInfo{
			Client:      int32(client.CN),
			Protocol:    P.PROTOCOL_VERSION,
			SessionId:   int32(client.SessionID),
			HasPassword: false, // password protection is not used by this implementation
			Description: s.Description,
			Domain:      "",
		},
	)

	return client, connected
}

// Send the server info to clients again, which updates the description on the
// scoreboard.
func (s *Server) RefreshServerInfo() {
	s.Clients.ForEach(func(c *Client) {
		c.Send(
			P.ServerInfo{
				Client:      int32(c.CN),
				Protocol:    P.PROTOCOL_VERSION,
				SessionId:   int32(c.SessionID),
				HasPassword: false, // password protection is not used by this implementation
				Description: s.Description,
				Domain:      "",
			},
		)
	})
}

func (s *Server) SetDescription(description string) {
	s.Description = description
	s.RefreshServerInfo()
}

func (s *Server) RefreshTime() {
	s.Broadcast(P.TimeUp{Remaining: int32(s.Clock.TimeLeft() / time.Second)})
}

func (s *Server) BroadcastTime(seconds int) {
	s.Broadcast(P.TimeUp{Remaining: int32(seconds)})
}

func (s *Server) Pause() {
	s.Clock.Pause(nil)
}

func (s *Server) Resume() {
	s.Clock.Pause(nil)
}

// Forcibly respawn a player. Passing nil respawns all non-spectating players.
func (s *Server) ForceRespawn(target *Client) {
	s.Clients.ForEach(func(c *Client) {
		if target != nil && c != target {
			return
		}

		if c.State == playerstate.Spectator {
			return
		}

		s.Spawn(c)
		c.Send(P.SpawnState{Client: int32(c.CN), EntityState: c.ToWire()})
	})
}

// Kill all players, reset their scores (if resetFrags is true), and respawn them.
func (s *Server) ResetPlayers(resetFrags bool) {
	s.Clients.ForEach(func(c *Client) {
		c.Die()

		if resetFrags {
			c.Frags = 0
			c.Deaths = 0
			c.Teamkills = 0
			c.Team.Frags = 0
		}

		s.Broadcast(P.Died{
			Client:      int32(c.CN),
			Killer:      int32(c.CN),
			KillerFrags: c.Frags,
			VictimFrags: c.Team.Frags,
		})
	})
}

func (s *Server) TryJoin(c *Client, name string, playerModel int32, authDomain, authName string) {
	// ignore this if the user has already joined
	if c.Joined {
		return
	}

	c.Name = name
	c.Model = playerModel
	s.Join(c)
}

// Puts a client into the current game, using the data the client provided with his nmc.TryJoin packet.
func (s *Server) Join(c *Client) {
	c.Joined = true
	c.connected <- true

	if s.MasterMode == mastermode.Locked {
		c.State = playerstate.Spectator
	} else {
		c.State = playerstate.Dead
		s.Spawn(c)
	}

	if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
		teamedMode.Join(&c.Player) // may set client's team
	}
	s.SendWelcome(c) // tells client about her team
	if flagMode, ok := s.GameMode.(game.FlagMode); ok {
		c.Send(flagMode.FlagsInitPacket())
	}
	s.Clients.InformOthersOfJoin(c)
}

func (s *Server) Message(message string) {
	s.Broadcast(P.ServerMessage{Text: message})
}

func (s *Server) Broadcast(messages ...P.Message) {
	s.Clients.Broadcast(messages...)
}

func (s *Server) UniqueName(p *game.Player) string {
	return s.Clients.UniqueName(s.Clients.GetClientByCN(p.CN))
}

func (s *Server) Spawn(client *Client) {
	client.Spawn()
	s.GameMode.Spawn(&client.PlayerState)
}

func (s *Server) ConfirmSpawn(client *Client, lifeSequence, _weapon int32) {
	if client.State != playerstate.Dead || lifeSequence != client.LifeSequence || client.LastSpawnAttempt.IsZero() {
		// client may not spawn
		return
	}

	client.State = playerstate.Alive
	client.SelectedWeapon = weapon.ByID(weapon.ID(_weapon))
	client.LastSpawnAttempt = time.Time{}

	client.Packets.Publish(P.SpawnResponse{
		EntityState: client.ToWire(),
	})

	if clock, competitive := s.GameMode.(game.Competitive); competitive {
		clock.Spawned(&client.Player)
	}
}

func (s *Server) Leave(sessionId uint32) {
	client := s.Clients.GetClientByID(sessionId)
	if client == nil {
		return
	}

	s.Disconnect(client, disconnectreason.None)
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.GameMode.Leave(&client.Player)
	s.Clock.Leave(&client.Player)
	s.Clients.Disconnect(client, reason)
	err := s.relay.RemoveClient(client.CN)
	if err != nil {
		log.Error().Err(err).Msgf("could not disconnect %d", client.SessionID)
	}
	if len(s.Clients.PrivilegedUsers()) == 0 {
		s.Unsupervised()
	}
	if s.Clients.GetNumClients() == 0 {
		s.Empty()
	}
}

func (s *Server) Kick(client *Client, victim *Client, reason string) {
	if client.Role <= victim.Role {
		client.Message(cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s kicked %s", s.Clients.UniqueName(client), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Message(msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) AuthKick(client *Client, rol role.ID, domain, name string, victim *Client, reason string) {
	if rol <= victim.Role {
		client.Message(cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s as '%s' [%s] kicked %s", s.Clients.UniqueName(client), cubecode.Magenta(name), cubecode.Green(domain), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Message(msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) Unsupervised() {
	s.Clock.Resume(nil)
	s.MasterMode = mastermode.Auth
	s.KeepTeams = false
	s.CompetitiveMode = false
	s.ReportStats = true
}

func (s *Server) Empty() {
	// We don't want to do this anymore
	// s.StartGame(s.StartMode(s.FallbackGameModeID), s.Map)
}

func (s *Server) Intermission() {
	s.Clock.Stop()

	allMaps := make([]string, 0)
	allMaps = append(allMaps, s.Maps...)
	allMaps = append(allMaps, s.DefaultMap)
	nextMap := allMaps[s.rng.Uint32()%uint32(len(allMaps))]

	s.pendingMapChange = time.AfterFunc(10*time.Second, func() {
		s.StartGame(s.StartMode(s.GameMode.ID()), nextMap)
	})

	s.Message("next up: " + nextMap)
}

// Returns the number of connected clients playing (i.e. joined and not spectating)
func (s *Server) NumberOfPlayers() (n int) {
	s.Clients.ForEach(func(c *Client) {
		if !c.Joined || c.State == playerstate.Spectator {
			return
		}
		n++
	})
	return
}

func (s *Server) EmptyMap() {
	s.StartGame(s.StartMode(gamemode.CoopEdit), "")
}

func (s *Server) ChangeMap(mode int32, map_ string) {
	s.StartGame(s.StartMode(gamemode.ID(mode)), map_)
}

func (s *Server) SetMode(mode int32) {
	s.StartGame(s.StartMode(gamemode.ID(mode)), s.Map)
}

func (s *Server) SetMap(map_ string) {
	s.StartGame(s.GameMode, map_)
}

func (s *Server) StartGame(mode game.Mode, mapname string) {
	if s.Clock != nil {
		s.Clock.CleanUp()
	}
	if s.CompetitiveMode {
		s.Clock = game.NewCompetitiveClock(s, mode)
	} else if mode.ID() == gamemode.CoopEdit {
		s.Clock = game.NewEndlessClock(s, mode)
	} else {
		s.Clock = game.NewCasualClock(s, mode)
	}

	// stop any pending map change
	if s.pendingMapChange != nil {
		s.pendingMapChange.Stop()
		s.pendingMapChange = nil
	}

	s.Map = mapname
	s.GameMode = mode

	s.maps <- mapname

	if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
		s.ForEachPlayer(teamedMode.Join)
	}

	s.Broadcast(
		P.MapChange{
			Name:     s.Map,
			Mode:     int32(s.GameMode.ID()),
			HasItems: s.GameMode.NeedsMapInfo(),
		},
	)

	s.Clock.Start()

	s.MapChange()
}

func (s *Server) SetMasterMode(c *Client, mm mastermode.ID) {
	if mm < mastermode.Open || mm > mastermode.Private {
		log.Info().Msgf("invalid mastermode %d requested", mm)
		return
	}
	if mm == mastermode.Open {
		c.Message(cubecode.Fail("'open' mode is not supported by this server"))
		return
	}
	if c.Role == role.None {
		c.Message(cubecode.Fail("you can't do that"))
		return
	}
	s._SetMasterMode(mm)
}

func (s *Server) SetPublicServer(mm mastermode.ID) {
	s._SetMasterMode(mm)
}

func (s *Server) _SetMasterMode(mm mastermode.ID) {
	s.MasterMode = mm
	s.Clients.Broadcast(P.MasterMode{int32(mm)})
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}

func (s *Server) HandleShoot(client *Client, wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit) {
	s.Clients.Relay(
		client,
		P.ShotFX{
			Client: int32(client.CN),
			Gun:    int32(wpn.ID),
			Id:     id,
			From:   P.Vec{X: from.X(), Y: from.Y(), Z: from.Z()},
			To:     P.Vec{X: to.X(), Y: to.Y(), Z: to.Z()},
		},
	)
	client.LastShot = time.Now()
	client.DamagePotential += wpn.Damage * wpn.Rays // TODO: quad damage
	if wpn.ID != weapon.Saw {
		client.Ammo[wpn.ID]--
	}
	switch wpn.ID {
	case weapon.GrenadeLauncher, weapon.RocketLauncher:
		// wait for nmc.Explode pkg
	default:
		// apply damage
		rays := int32(0)
		for _, h := range hits {
			target := s.Clients.GetClientByCN(h.target)
			if target == nil ||
				target.State != playerstate.Alive ||
				target.LifeSequence != h.lifeSequence ||
				h.rays < 1 ||
				h.distance > wpn.Range+1.0 {
				continue
			}

			rays += h.rays
			if rays > wpn.Rays {
				continue
			}

			damage := h.rays * wpn.Damage
			// TODO: quad damage

			s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
		}
	}
}

func (s *Server) HandleExplode(client *Client, millis int32, wpn weapon.Weapon, id int32, hits []hit) {
	// TODO: delete stored projectile

	s.Clients.Relay(
		client,
		P.ExplodeFX{
			int32(client.CN),
			int32(wpn.ID),
			id,
		},
	)

	// apply damage
hits:
	for i, h := range hits {
		target := s.Clients.GetClientByCN(h.target)
		if target == nil ||
			target.State != playerstate.Alive ||
			target.LifeSequence != h.lifeSequence ||
			h.distance < 0 ||
			h.distance > wpn.ExplosionRadius {
			continue
		}

		// avoid duplicates
		for j := range hits[:i] {
			if hits[j].target == h.target {
				continue hits
			}
		}

		damage := float64(wpn.Damage)
		// TODO: quad damage
		damage *= (1 - h.distance/weapon.ExplosionDistanceScale/wpn.ExplosionRadius)
		if target == client {
			damage *= weapon.ExplosionSelfDamageScale
		}

		s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
	}
}

func (s *Server) applyDamage(attacker, victim *Client, damage int32, wpnID weapon.ID, dir *geom.Vector) {
	victim.ApplyDamage(&attacker.Player, damage, wpnID, dir)
	s.Clients.Broadcast(
		P.Damage{
			int32(victim.CN),
			int32(attacker.CN),
			damage,
			victim.Armour,
			victim.Health,
		},
	)
	// TODO: setpushed ???
	if !dir.IsZero() {
		dir = dir.Scale(geom.DNF)
		hitPush := P.HitPush{
			int32(victim.CN), int32(wpnID), damage,
			P.Vec{dir.X(), dir.Y(), dir.Z()},
		}
		if victim.Health <= 0 {
			s.Clients.Broadcast(hitPush)
		} else {
			victim.Send(hitPush)
		}
	}
	if victim.Health <= 0 {
		s.GameMode.HandleFrag(&attacker.Player, &victim.Player)
	}
}

func (s *Server) ForEachPlayer(f func(p *game.Player)) {
	s.Clients.ForEach(func(c *Client) {
		f(&c.Player)
	})
}
