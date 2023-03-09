package service

import (
	"context"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/commands"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	CREATE_SERVER_COOLDOWN = time.Duration(10 * time.Second)
)

type Cluster struct {
	// State
	createMutex sync.Mutex
	// host -> time a client from that host last created a server. We
	// REALLY don't want clients to be able to DDOS us
	lastCreate map[string]time.Time
	// host -> the server created by that host
	// each host can only have one server at once
	hostServers   map[string]*servers.GameServer
	started       time.Time
	authDomain    string
	settings      config.ClusterSettings
	serverCtx     context.Context
	serverMessage chan []byte

	commands *commands.CommandGroup[*User]

	// Services
	Users   *UserOrchestrator
	auth    *auth.DiscordService
	servers *servers.ServerManager
	matches *Matchmaker
	redis   *redis.Client
	spaces  *verse.SpaceManager
	verse   *verse.Verse
	assets  *assets.AssetFetcher
}

func NewCluster(
	ctx context.Context,
	serverManager *servers.ServerManager,
	maps *assets.AssetFetcher,
	settings config.ClusterSettings,
	authDomain string,
	auth *auth.DiscordService,
	redis *redis.Client,
) *Cluster {
	v := verse.NewVerse(redis)
	server := &Cluster{
		Users:         NewUserOrchestrator(redis, settings.Matchmaking.Duel),
		serverCtx:     ctx,
		settings:      settings,
		authDomain:    authDomain,
		hostServers:   make(map[string]*servers.GameServer),
		commands:      commands.NewCommandGroup[*User]("cluster", game.ColorOrange),
		lastCreate:    make(map[string]time.Time),
		matches:       NewMatchmaker(serverManager, settings.Matchmaking.Duel),
		serverMessage: make(chan []byte, 1),
		servers:       serverManager,
		started:       time.Now(),
		auth:          auth,
		redis:         redis,
		verse:         v,
		spaces:        verse.NewSpaceManager(v, serverManager, maps),
		assets:        maps,
	}

	server.registerCommands()

	return server
}

func (server *Cluster) GetServerInfo() *servers.ServerInfo {
	info := server.servers.GetServerInfo()

	settings := server.settings.ServerInfo

	info.TimeLeft = int32(settings.TimeLeft)
	info.MaxClients = 999
	info.GameSpeed = int32(settings.GameSpeed)
	info.Map = settings.Map
	info.Description = settings.Description

	return info
}

func (server *Cluster) GetTeamInfo() *servers.TeamInfo {
	info := servers.TeamInfo{
		IsDeathmatch: true,
		GameMode:     0,
		TimeLeft:     9999,
		Scores:       make([]servers.TeamScore, 0),
	}
	return &info
}

// We need client information, so this is not on the ServerManager like GetServerInfo is
func (server *Cluster) GetClientInfo() []*servers.ClientExtInfo {
	info := make([]*servers.ClientExtInfo, 0)

	server.servers.Mutex.Lock()

	for _, gameServer := range server.servers.Servers {
		clients := gameServer.GetClientInfo()
		for _, client := range clients {
			newClient := *client

			// TODO do we still want client ids here?

			info = append(info, &newClient)
		}
	}

	server.servers.Mutex.Unlock()

	return info
}

func (server *Cluster) GetUptime() int {
	return int(time.Now().Sub(server.started).Round(time.Second) / time.Second)
}

func (server *Cluster) PollServers(ctx context.Context) {
	forceDisconnects := server.servers.ReceiveKicks()
	gamePackets := server.servers.ReceivePackets()

	for {
		select {
		case event := <-forceDisconnects:
			user := server.Users.FindUser(event.Client)

			if user == nil {
				continue
			}

			logger := user.Logger()
			logger.Info().Msgf("user forcibly disconnected %d %s", event.Reason, event.Text)

			user.DisconnectFromServer()

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			user.Connection.Disconnect(int(event.Reason), event.Text)
		case p := <-gamePackets:
			messages := p.Messages
			gameServer := p.Server

			user := server.Users.FindUser(p.Client)

			if user == nil {
				continue
			}

			if user.GetServer() != gameServer {
				continue
			}

			channel := uint8(p.Channel)

			out := make([]P.Message, 0)

			for _, message := range messages {
				newMessage, err := gameServer.From.Process(
					ctx,
					channel,
					message,
				)
				if err != nil {
					log.Error().Err(err).Msgf("failed to process message")
					continue
				}

				if newMessage == nil {
					continue
				}

				out = append(out, newMessage)
			}

			for _, message := range out {
				type_ := message.Type()
				if !P.IsSpammyMessage(type_) {
					log.Debug().
						Str("type", message.Type().String()).
						Msg("server -> client")
				}
			}

			user.SendChannel(channel, out...)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) StartServers(ctx context.Context) {
	go server.PollServers(ctx)
	for _, presetSpace := range server.settings.Spaces {
		server.spaces.StartPresetSpace(ctx, presetSpace)
	}
	go server.servers.PruneServers(ctx)
	go server.matches.Poll(ctx)
}

func (server *Cluster) PollUsers(ctx context.Context, newConnections chan ingress.Connection) {
	for {
		select {
		case connection := <-newConnections:
			user, err := server.Users.AddUser(ctx, connection)
			if err != nil {
				log.Error().Err(err).Msgf("failed to add user")
				continue
			}

			go server.PollUser(ctx, user)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) Shutdown() {
	server.servers.Shutdown()
}
