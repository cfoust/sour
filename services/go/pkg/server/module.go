package server

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/relay"
	"github.com/cfoust/sour/pkg/server/maprot"
	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/mastermode"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/role"
	"github.com/cfoust/sour/pkg/server/protocol/weapon"
)

type Server struct {
	*Config
	*State
	relay            *relay.Relay
	Clients          *ClientManager
	MapRotation      *maprot.Rotation
	PendingMapChange *time.Timer
	callbacks        chan<- func()
	rng              *rand.Rand

	// non-standard stuff
	Commands        *ServerCommands
	KeepTeams       bool
	CompetitiveMode bool
	ReportStats     bool
}

func New(conf *Config, commands ...*ServerCommand) (*Server, <-chan func()) {
	callbacks := make(chan func())
	clients := &ClientManager{}

	s := &Server{
		Config: conf,
		State: &State{
			MasterMode: mastermode.Auth,
			UpSince:    time.Now(),
			NumClients: clients.NumberOfClientsConnected,
		},
		relay:       relay.New(),
		Clients:     clients,
		MapRotation: maprot.NewRotation(conf.MapPools),
		callbacks:   callbacks,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	s.Commands = NewCommands(s, commands...)

	return s, callbacks
}

func (s *Server) GameDuration() time.Duration { return s.Config.GameDuration }

func (s *Server) Connect() {
	log.Println("connecting")
	client := s.Clients.Add()
	client.Positions, client.Packets = s.relay.AddClient(client.CN, func(channel uint8, payload []byte) {
		panic("TODO")
	})
	client.Send(
		nmc.ServerInfo,
		client.CN,
		protocol.Version,
		client.SessionID,
		false, // password protection is not used by this implementation
		s.ServerDescription,
		s.AuthDomain,
	)
	log.Println("informed about server")
}

// checks the server state and decides wether the client has to authenticate to join the game.
func (s *Server) TryJoin(c *Client, name string, playerModel int32, authDomain, authName string) {
	c.Name = name
	c.Model = playerModel
	s.Join(c)
}

// Puts a client into the current game, using the data the client provided with his nmc.TryJoin packet.
func (s *Server) Join(c *Client) {
	c.Joined = true

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
		c.Send(nmc.InitFlags, flagMode.FlagsInitPacket()...)
	}
	s.Clients.InformOthersOfJoin(c)

	uniqueName := s.Clients.UniqueName(c)
	log.Println(cubecode.SanitizeString(fmt.Sprintf("%s connected", uniqueName)))

	c.Send(nmc.ServerMessage, s.MessageOfTheDay)
	c.Send(nmc.RequestAuth, s.StatsServerAuthDomain)
}

func (s *Server) Broadcast(typ nmc.ID, args ...interface{}) {
	s.Clients.Broadcast(typ, args...)
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

	client.Packets.Publish(nmc.ConfirmSpawn, client.ToWire())

	if clock, competitive := s.GameMode.(game.Competitive); competitive {
		clock.Spawned(&client.Player)
	}
}

func (s *Server) Disconnect(client *Client, reason disconnectreason.ID) {
	s.GameMode.Leave(&client.Player)
	s.Clock.Leave(&client.Player)
	s.relay.RemoveClient(client.CN)
	s.Clients.Disconnect(client, reason)
	s.Clients.ForEach(func(c *Client) { log.Printf("%#v\n", c) })
	client.Reset()
	if len(s.Clients.PrivilegedUsers()) == 0 {
		s.Unsupervised()
	}
	if s.Clients.NumberOfClientsConnected() == 0 {
		s.Empty()
	}
}

func (s *Server) Kick(client *Client, victim *Client, reason string) {
	if client.Role <= victim.Role {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s kicked %s", s.Clients.UniqueName(client), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Clients.Broadcast(nmc.ServerMessage, msg)
	s.Disconnect(victim, disconnectreason.Kick)
}

func (s *Server) AuthKick(client *Client, rol role.ID, domain, name string, victim *Client, reason string) {
	if rol <= victim.Role {
		client.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	msg := fmt.Sprintf("%s as '%s' [%s] kicked %s", s.Clients.UniqueName(client), cubecode.Magenta(name), cubecode.Green(domain), s.Clients.UniqueName(victim))
	if reason != "" {
		msg += " for: " + reason
	}
	s.Clients.Broadcast(nmc.ServerMessage, msg)
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
	s.MapRotation.ClearQueue()
	s.StartGame(s.StartMode(s.FallbackGameModeID), s.Map)
}

func (s *Server) Intermission() {
	s.Clock.Stop()

	nextMap := s.MapRotation.NextMap(s.GameMode, s.GameMode, s.Map)

	s.PendingMapChange = time.AfterFunc(10*time.Second, func() {
		s.StartGame(s.StartMode(s.GameMode.ID()), nextMap)
	})

	s.Clients.Broadcast(nmc.ServerMessage, "next up: "+nextMap)
}

// Returns the number of connected clients playgin (i.e. joined and not spectating)
func (s *Server) NumberOfPlayers() (n int) {
	s.Clients.ForEach(func(c *Client) {
		if !c.Joined || c.State == playerstate.Spectator {
			return
		}
		n++
	})
	return
}

func (s *Server) ReAuthClients(domain string) {
	s.Clients.ForEach(func(c *Client) {
		if _, ok := c.Authentications[domain]; ok {
			delete(c.Authentications, domain)
			c.Send(nmc.RequestAuth, domain)
		}
	})
}

func (s *Server) StartGame(mode game.Mode, mapname string) {
	if s.Clock != nil {
		s.Clock.CleanUp()
	}
	if s.CompetitiveMode {
		s.Clock = game.NewCompetitiveClock(s, mode)
	} else {
		s.Clock = game.NewCasualClock(s, mode)
	}

	// stop any pending map change
	if s.PendingMapChange != nil {
		s.PendingMapChange.Stop()
		s.PendingMapChange = nil
	}

	if mapname == "" {
		mapname = s.MapRotation.NextMap(mode, s.GameMode, s.Map)
	}

	s.Map = mapname
	s.GameMode = mode

	if teamedMode, ok := s.GameMode.(game.TeamMode); ok {
		s.ForEachPlayer(teamedMode.Join)
	}

	s.Broadcast(nmc.MapChange, s.Map, s.GameMode.ID(), s.GameMode.NeedsMapInfo())
	s.Clock.Start()
	s.MapChange()

	s.Clients.Broadcast(nmc.ServerMessage, s.MessageOfTheDay)
}

func (s *Server) SetMasterMode(c *Client, mm mastermode.ID) {
	if mm < mastermode.Open || mm > mastermode.Private {
		log.Println("invalid mastermode", mm, "requested")
		return
	}
	if mm == mastermode.Open {
		c.Send(nmc.ServerMessage, cubecode.Fail("'open' mode is not supported by this server"))
		return
	}
	if c.Role == role.None {
		c.Send(nmc.ServerMessage, cubecode.Fail("you can't do that"))
		return
	}
	s.MasterMode = mm
	s.Clients.Broadcast(nmc.MasterMode, mm)
}

type hit struct {
	target       uint32
	lifeSequence int32
	distance     float64
	rays         int32
	dir          *geom.Vector
}

func (s *Server) HandleShoot(client *Client, wpn weapon.Weapon, id int32, from, to *geom.Vector, hits []hit) {
	from = from.Mul(geom.DMF)
	to = to.Mul(geom.DMF)

	s.Clients.Relay(
		client,
		nmc.ShotEffects,
		client.CN,
		wpn.ID,
		id,
		from.X(),
		from.Y(),
		from.Z(),
		to.X(),
		to.Y(),
		to.Z(),
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
		nmc.ExplodeEffects,
		client.CN,
		wpn.ID,
		id,
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
	s.Clients.Broadcast(nmc.Damage, victim.CN, attacker.CN, damage, victim.Armour, victim.Health)
	// TODO: setpushed ???
	if !dir.IsZero() {
		dir = dir.Scale(geom.DNF)
		typ, p := nmc.HitPush, []interface{}{victim.CN, wpnID, damage, dir.X(), dir.Y(), dir.Z()}
		if victim.Health <= 0 {
			s.Clients.Broadcast(typ, p...)
		} else {
			victim.Send(typ, p...)
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
