package servers

import (
	_ "embed"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type GameServer struct {
	*gameserver.Server

	// Lifecycle
	Session utils.Session

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

	// Buffered channel for incoming packets from the cluster layer.
	// The Step ticker goroutine drains this.
	incoming chan gameserver.InputPacket
}

func (server *GameServer) Ctx() <-chan struct{} {
	return server.Session.Ctx().Done()
}

func (server *GameServer) Cancel() {
	server.Session.Cancel()
}

func (server *GameServer) NumClients() int {
	return server.Clients.GetNumClients()
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

// SendPacket queues a packet to be processed in the next Step() call.
func (server *GameServer) SendPacket(packet gameserver.InputPacket) {
	select {
	case server.incoming <- packet:
	default:
		// Drop packet if buffer is full
		log.Warn().Str("server", server.Reference()).Msg("incoming packet buffer full, dropping packet")
	}
}

// ConnectClient registers a new client. Output packets are routed through
// the manager's packets channel. Returns the server-side Client.
func (server *GameServer) ConnectClient(sessionID uint32) *gameserver.Client {
	server.Mutex.Lock()
	server.LastEvent = time.Now()
	server.Mutex.Unlock()

	outputs := server.Server.Connect(sessionID)
	for _, pkt := range outputs {
		server.packets <- ClientPacket{
			Client:   ingress.ClientID(pkt.Session),
			Channel:  pkt.Channel,
			Messages: pkt.Messages,
			Server:   server,
		}
	}

	return server.Clients.GetClientByID(sessionID)
}

// LeaveClient removes a client and routes output packets.
func (server *GameServer) LeaveClient(sessionID uint32) {
	outputs := server.Server.Leave(sessionID)
	for _, pkt := range outputs {
		server.packets <- ClientPacket{
			Client:   ingress.ClientID(pkt.Session),
			Channel:  pkt.Channel,
			Messages: pkt.Messages,
			Server:   server,
		}
	}
}

func (s *GameServer) GetServerInfo() *ServerInfo {
	clock := s.Server.GameClock()
	return &ServerInfo{
		NumClients:   int32(s.NumClients()),
		GamePaused:   s.Clock.Paused(),
		GameMode:     int32(s.GameMode.ID()),
		TimeLeft:     int32(s.Clock.TimeLeft(clock) / time.Second),
		MaxClients:   64,
		PasswordMode: 0,
		GameSpeed:    100,
		Map:          s.Map,
		Description:  s.Description,
	}
}

func (s *GameServer) GetClientInfo() []*ClientExtInfo {
	clients := make([]*ClientExtInfo, 0)

	s.Clients.ForEach(func(c *gameserver.Client) {
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
	clock := s.Server.GameClock()
	return &TeamInfo{
		IsDeathmatch: false,
		GameMode:     int(s.GameMode.ID()),
		TimeLeft:     int(s.Clock.TimeLeft(clock) / time.Second),
	}
}

func (s *GameServer) GetUptime() int {
	return int(time.Since(s.Started).Round(time.Second) / time.Second)
}

var _ InfoProvider = (*GameServer)(nil)
