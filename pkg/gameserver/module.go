package gameserver

import (
	"fmt"
	"math/rand"
	"time"

	G "github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/commands"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/deadline"
	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/geom"
	"github.com/cfoust/sour/pkg/gameserver/protocol/cubecode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/mastermode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"

	"github.com/rs/zerolog/log"
)

type ServerPacket struct {
	Session  uint32
	Channel  uint8
	Messages []P.Message
}

type InputPacket = ServerPacket

type Server struct {
	*Config
	*State
	relay SyncRelay
	out   OutputBuffer

	Description string

	Clients *ClientManager

	Commands *commands.CommandGroup[*Client]

	pendingMapChange deadline.Deadline
	pendingMap       string
	rng              *rand.Rand
	gameClock        int64 // current server time in ms

	Broadcasts [][]P.Message

	// non-standard stuff
	KeepTeams       bool
	CompetitiveMode bool
	ReportStats     bool
}

// Implement game.Server interface

func (s *Server) GameDuration() int64 {
	return int64(s.Config.MatchLength) * 1000
}

func (s *Server) GameClock() int64 {
	return s.gameClock
}

func (s *Server) Broadcast(messages ...P.Message) {
	s.Clients.BroadcastTo(&s.out, messages...)
	s.Broadcasts = append(s.Broadcasts, messages)
}

func (s *Server) Message(message string) {
	s.Broadcast(P.ServerMessage{Text: message})
}

func (s *Server) UniqueName(p *game.Player) string {
	c := s.Clients.GetClientByCN(p.CN)
	if c == nil {
		return p.Name
	}
	return s.Clients.UniqueName(c)
}

func (s *Server) ForEachPlayer(f func(p *game.Player)) {
	s.Clients.ForEach(func(c *Client) {
		f(&c.Player)
	})
}

func (s *Server) NumberOfPlayers() (n int) {
	s.Clients.ForEach(func(c *Client) {
		if !c.Joined || c.State == playerstate.Spectator {
			return
		}
		n++
	})
	return
}

// ClientMessage sends a server message to a specific client.
func (s *Server) ClientMessage(c *Client, text string) {
	s.out.Send(c.SessionID, P.ServerMessage{Text: text})
}

func New(conf *Config) *Server {
	s := &Server{
		Config: conf,
		State: &State{
			MasterMode: mastermode.Auth,
		},
		relay:    NewSyncRelay(),
		Clients:  &ClientManager{},
		Commands: commands.NewCommandGroup[*Client]("server", G.ColorBlue),
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	return s
}

// Connect registers a new client. Returns output packets (including ServerInfo).
func (s *Server) Connect(sessionId uint32) []ServerPacket {
	existing := s.Clients.GetClientByID(sessionId)
	if existing != nil {
		log.Error().Uint32("session", sessionId).Msg("client already connected")
		return nil
	}

	client := s.Clients.Add(sessionId)
	client.server = s
	s.relay.AddClient(client.CN, client.SessionID)

	s.out.Send(client.SessionID,
		P.ServerInfo{
			Client:      int32(client.CN),
			Protocol:    P.PROTOCOL_VERSION,
			SessionId:   int32(client.SessionID),
			HasPassword: false,
			Description: s.Description,
			Domain:      "",
		},
	)

	return s.out.Drain()
}

// Leave removes a client. Returns output packets (disconnect notifications).
func (s *Server) Leave(sessionId uint32) []ServerPacket {
	client := s.Clients.GetClientByID(sessionId)
	if client == nil {
		return nil
	}
	s.Disconnect(client, disconnectreason.None)
	return s.out.Drain()
}

// Step advances the server clock by dtMs milliseconds, processes all
// incoming packets, ticks timers, flushes the relay, and returns output.
func (s *Server) Step(dtMs int64, incoming []InputPacket) []ServerPacket {
	s.gameClock += dtMs

	// 1. Process incoming packets grouped by sender, matching the C++
	// server's processevents() (server.cpp:2440) which iterates clients
	// in order and flushes each client's event queue completely before
	// moving to the next. This prevents cross-client interleaving where
	// one client's respawn increments their LifeSequence before another
	// client's shot (referencing the old LifeSequence) is validated.
	bySession := make(map[uint32][]InputPacket)
	for _, pkt := range incoming {
		bySession[pkt.Session] = append(bySession[pkt.Session], pkt)
	}
	s.Clients.ForEach(func(c *Client) {
		pkts, ok := bySession[c.SessionID]
		if !ok {
			return
		}
		for _, pkt := range pkts {
			for _, message := range pkt.Messages {
				s.HandlePacket(c, pkt.Channel, message)
			}
		}
	})

	// 2. Tick game clock (intermission, etc.)
	if s.State.Clock != nil {
		s.State.Clock.Tick(s.gameClock)
	}

	// 3. Check pending map change (from Intermission)
	if s.pendingMapChange.Expired(s.gameClock) {
		s.pendingMapChange.Stop()
		s.StartGame(s.StartMode(s.GameMode.ID()), s.pendingMap)
	}

	// 4. Flush relay (batched position/packet distribution)
	s.relay.Flush(&s.out)

	return s.out.Drain()
}

// PendingMap returns and clears any pending map change notification.
func (s *Server) PendingMap() string {
	m := s.pendingMap
	if m != "" {
		s.pendingMap = ""
	}
	return m
}

func (s *Server) RefreshServerInfo() {
	s.Clients.ForEach(func(c *Client) {
		s.out.Send(c.SessionID,
			P.ServerInfo{
				Client:      int32(c.CN),
				Protocol:    P.PROTOCOL_VERSION,
				SessionId:   int32(c.SessionID),
				HasPassword: false,
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
	s.Broadcast(P.TimeUp{Remaining: int32(s.State.Clock.TimeLeft(s.gameClock) / time.Second)})
}

func (s *Server) BroadcastTime(seconds int) {
	s.Broadcast(P.TimeUp{Remaining: int32(seconds)})
}

func (s *Server) Pause() {
	s.State.Clock.Pause(s.gameClock, nil)
}

func (s *Server) Resume() {
	s.State.Clock.Resume(s.gameClock, nil)
}

func (s *Server) ForceRespawn(target *Client) {
	s.Clients.ForEach(func(c *Client) {
		if target != nil && c != target {
			return
		}
		if c.State == playerstate.Spectator {
			return
		}
		s.Spawn(c)
		s.out.Send(c.SessionID, P.SpawnState{Client: int32(c.CN), EntityState: c.ToWire()})
	})
}

func (s *Server) ResetPlayers(resetFrags bool) {
	s.Clients.ForEach(func(c *Client) {
		c.Die(s.gameClock)

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
	if c.Joined {
		return
	}
	c.Name = name
	c.Model = playerModel
	s.Join(c)
}

func (s *Server) Join(c *Client) {
	c.Joined = true

	if s.MasterMode == mastermode.Locked {
		c.State = playerstate.Spectator
	} else {
		c.State = playerstate.Dead
		s.Spawn(c)
	}

	if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
		teamedMode.Join(&c.Player)
	}
	s.SendWelcome(c)
	if flagMode, ok := s.GameMode.(game.FlagMode); ok {
		s.out.Send(c.SessionID, flagMode.FlagsInitPacket())
	}
	s.Clients.InformOthersOfJoin(c, &s.out)
}

func (s *Server) Spawn(client *Client) {
	client.Spawn(s.gameClock)
	s.GameMode.Spawn(&client.PlayerState)
}

func (s *Server) ConfirmSpawn(client *Client, lifeSequence, _weapon int32) {
	if client.State != playerstate.Dead || lifeSequence != client.LifeSequence || client.LastSpawnAttempt == -1 {
		return
	}

	client.State = playerstate.Alive
	client.SelectedWeapon = weapon.ByID(weapon.ID(_weapon))
	client.LastSpawnAttempt = -1

	s.relay.AddPacket(client.CN, P.SpawnResponse{
		EntityState: client.ToWire(),
	})

	if notifier, ok := s.State.Clock.(game.SpawnNotifier); ok {
		notifier.Spawned(&client.Player)
	}
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.GameMode.Leave(&client.Player)
	s.State.Clock.Leave(&client.Player)
	s.Clients.Disconnect(client, &s.out, reason)
	s.relay.RemoveClient(client.CN)
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
	s.State.Clock.Resume(s.gameClock, nil)
	s.MasterMode = mastermode.Auth
	s.KeepTeams = false
	s.CompetitiveMode = false
	s.ReportStats = true
}

func (s *Server) Empty() {}

func (s *Server) Intermission() {
	s.State.Clock.Stop()

	allMaps := make([]string, 0)
	allMaps = append(allMaps, s.Maps...)
	allMaps = append(allMaps, s.DefaultMap)
	nextMap := allMaps[s.rng.Uint32()%uint32(len(allMaps))]

	s.pendingMap = nextMap
	s.pendingMapChange.Set(s.gameClock, 10000)

	s.Message("next up: " + nextMap)
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
	if s.State.Clock != nil {
		s.State.Clock.CleanUp()
	}
	if s.CompetitiveMode {
		s.State.Clock = game.NewCompetitiveClock(s, mode)
	} else if mode.ID() == gamemode.CoopEdit {
		s.State.Clock = game.NewEndlessClock(s, mode)
	} else {
		s.State.Clock = game.NewCasualClock(s, mode)
	}

	s.pendingMapChange.Stop()

	s.Map = mapname
	s.GameMode = mode

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

	s.State.Clock.Start(s.gameClock)

	s.MapChange()
}

func (s *Server) SetMasterMode(c *Client, mm mastermode.ID) {
	if mm < mastermode.Open || mm > mastermode.Private {
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
	s.Broadcast(P.MasterMode{MasterMode: int32(mm)})
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}

func (s *Server) HandleShoot(client *Client, wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit) {
	s.Clients.RelayTo(
		client, &s.out,
		P.ShotFX{
			Client: int32(client.CN),
			Gun:    int32(wpn.ID),
			Id:     id,
			From:   P.Vec{X: from.X(), Y: from.Y(), Z: from.Z()},
			To:     P.Vec{X: to.X(), Y: to.Y(), Z: to.Z()},
		},
	)
	client.LastShot = s.gameClock
	client.DamagePotential += wpn.Damage * wpn.Rays
	if wpn.ID != weapon.Saw {
		client.Ammo[wpn.ID]--
	}
	switch wpn.ID {
	case weapon.GrenadeLauncher, weapon.RocketLauncher:
		// wait for Explode
	default:
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
			s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
		}
	}
}

func (s *Server) HandleExplode(client *Client, millis int32, wpn weapon.Weapon, id int32, hits []hit) {
	s.Clients.RelayTo(
		client, &s.out,
		P.ExplodeFX{
			Client: int32(client.CN),
			Gun:    int32(wpn.ID),
			Id:     id,
		},
	)

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

		for j := range hits[:i] {
			if hits[j].target == h.target {
				continue hits
			}
		}

		damage := float64(wpn.Damage)
		damage *= (1 - h.distance/weapon.ExplosionDistanceScale/wpn.ExplosionRadius)
		if target == client {
			damage *= weapon.ExplosionSelfDamageScale
		}

		s.applyDamage(client, target, int32(damage), wpn.ID, h.dir)
	}
}

func (s *Server) applyDamage(attacker, victim *Client, damage int32, wpnID weapon.ID, dir *geom.Vector) {
	victim.ApplyDamage(&attacker.Player, damage, wpnID, dir)
	s.Broadcast(
		P.Damage{
			Client:    int32(victim.CN),
			Aggressor: int32(attacker.CN),
			Damage:    damage,
			Armour:    victim.Armour,
			Health:    victim.Health,
		},
	)
	if !dir.IsZero() {
		dir = dir.Scale(geom.DNF)
		hitPush := P.HitPush{
			Client: int32(victim.CN),
			Gun:    int32(wpnID),
			Damage: damage,
			From:   P.Vec{X: dir.X(), Y: dir.Y(), Z: dir.Z()},
		}
		if victim.Health <= 0 {
			s.Broadcast(hitPush)
		} else {
			s.out.Send(victim.SessionID, hitPush)
		}
	}
	if victim.Health <= 0 {
		s.GameMode.HandleFrag(s.gameClock, &attacker.Player, &victim.Player)
	}
}
