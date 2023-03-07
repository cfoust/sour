package servers

import (
	_ "embed"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/server"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type GameServer struct {
	*server.Server

	Id string
	// Another way for the client to refer to this server
	Alias string

	Entities []maps.Entity

	// Whether this map was in our assets (ie can we send it to the client)
	IsBuiltMap bool

	Hidden bool

	// The last time a client connected
	LastEvent time.Time
	Started   time.Time

	Mutex deadlock.RWMutex

	From *P.MessageProxy
	To   *P.MessageProxy

	kicks   chan ClientKick
	packets chan ClientPacket
}

func (server *GameServer) GetEntities() []maps.Entity {
	server.Mutex.RLock()
	teleports := server.Entities
	server.Mutex.RUnlock()
	return teleports
}

// Whether this string is a reference to this server (either an alias or an id).
func (server *GameServer) IsReference(reference string) bool {
	return server.Id == reference || server.Alias == reference
}

func (server *GameServer) Reference() string {
	if server.Alias != "" {
		return server.Alias
	}
	return server.Id
}

func (server *GameServer) GetFormattedReference() string {
	reference := server.Reference()
	if server.Hidden {
		reference = "???"
	}
	return reference
}

func (server *GameServer) Logger() zerolog.Logger {
	return log.With().Str("server", server.Reference()).Logger()
}

func (server *GameServer) Shutdown() {
	server.Cancel()
}

func (s *GameServer) GetServerInfo() *ServerInfo {
	return &ServerInfo{
		NumClients:   int32(s.NumClients()),
		GamePaused:   s.Clock.Paused(),
		GameMode:     int32(s.GameMode.ID()),
		TimeLeft:     int32(s.Clock.TimeLeft() / time.Second),
		MaxClients:   64,
		PasswordMode: 0,
		GameSpeed:    100,
		Map:          s.Map,
		Description:  s.ServerDescription,
	}
}

func (s *GameServer) GetClientInfo() []*ClientExtInfo {
	clients := make([]*ClientExtInfo, 0)

	s.Clients.ForEach(func(c *server.Client) {
		clients = append(clients, &ClientExtInfo{
			Client:    int(c.CN),
			Ping:      int(c.Ping),
			Name:      c.Name,
			Team:      c.Team.Name,
			Frags:     c.Frags,
			Flags:     c.Flags,
			Deaths:    c.Deaths,
			TeamKills: c.Teamkills,
			Damage:    c.Damage,
			Health:    c.Health,
			Armour:    c.Armour,
			GunSelect: int32(c.SelectedWeapon.ID),
			Privilege: int32(c.Role),
			State:     int32(c.State),
			Ip0:       0,
			Ip1:       0,
			Ip2:       0,
		})
	})

	return clients
}

func (s *GameServer) GetTeamInfo() *TeamInfo {
	// TODO get team scores

	return &TeamInfo{
		IsDeathmatch: false,
		GameMode:     int(s.GameMode.ID()),
		TimeLeft:     int(s.Clock.TimeLeft() / time.Second),
	}
}

func (s *GameServer) GetUptime() int {
	return int(time.Now().Sub(s.Started).Round(time.Second) / time.Second)
}

var _ InfoProvider = (*GameServer)(nil)
