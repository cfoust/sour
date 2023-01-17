package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/go-redis/redis/v9"
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
	startTime     time.Time
	authDomain    string
	settings      config.ClusterSettings
	serverCtx     context.Context
	serverMessage chan []byte

	// Services
	Clients   *clients.ClientManager
	Users     *UserOrchestrator
	MapSender *MapSender
	auth      *auth.DiscordService
	manager   *servers.ServerManager
	matches   *Matchmaker
	redis     *redis.Client
	spaces    *verse.SpaceManager
	verse     *verse.Verse
	assets    *assets.AssetFetcher
}

func NewCluster(
	ctx context.Context,
	serverManager *servers.ServerManager,
	maps *assets.AssetFetcher,
	sender *MapSender,
	settings config.ClusterSettings,
	authDomain string,
	auth *auth.DiscordService,
	redis *redis.Client,
) *Cluster {
	clients := clients.NewClientManager()
	v := verse.NewVerse(redis)
	server := &Cluster{
		Users:         NewUserOrchestrator(redis, settings.Matchmaking.Duel),
		MapSender:     sender,
		serverCtx:     ctx,
		settings:      settings,
		authDomain:    authDomain,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		Clients:       clients,
		matches:       NewMatchmaker(serverManager, settings.Matchmaking.Duel),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
		startTime:     time.Now(),
		auth:          auth,
		redis:         redis,
		verse:         v,
		spaces:        verse.NewSpaceManager(v, serverManager, maps),
		assets:        maps,
	}

	return server
}

func (server *Cluster) GetServerInfo() *servers.ServerInfo {
	info := server.manager.GetServerInfo()

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

	server.manager.Mutex.Lock()

	for _, gameServer := range server.manager.Servers {
		clients := gameServer.GetClientInfo()
		for _, client := range clients {
			newClient := *client

			// TODO do we still want client ids here?

			info = append(info, &newClient)
		}
	}

	server.manager.Mutex.Unlock()

	return info
}

func (server *Cluster) GetUptime() int {
	return int(time.Now().Sub(server.startTime).Round(time.Second) / time.Second)
}

func (server *Cluster) PollServers(ctx context.Context) {
	connects := server.manager.ReceiveConnects()
	forceDisconnects := server.manager.ReceiveKicks()
	gamePackets := server.manager.ReceivePackets()
	names := server.manager.ReceiveNames()

	for {
		select {
		case join := <-connects:
			user := server.Users.FindUser(join.Client)

			if user == nil {
				continue
			}

			user.Mutex.Lock()
			if user.Server != nil {
				instance := server.spaces.FindInstance(user.Server)
				if instance != nil {
					user.Space = instance
				}
				user.Status = clients.ClientStatusConnected
				user.Num = join.Num
			}
			user.Mutex.Unlock()

			logger := user.Logger()
			logger.Info().Msg("connected to server")

			isHome, err := user.IsAtHome(ctx)
			if err != nil {
				logger.Warn().Err(err).Msg("failed seeing if user was at home")
				continue
			}

			if isHome {
				space := user.GetSpace()
				message := fmt.Sprintf(
					"welcome to your home (space %s).",
					space.GetID(),
				)

				if user.IsLoggedIn() {
					user.SendServerMessage(message)
					user.SendServerMessage("editing by others is disabled. say #edit to enable it.")
				} else {
					user.SendServerMessage(message + " anyone can edit it. because you are not logged in, it will be deleted in 4 hours")
				}
			}

		case event := <-names:
			user := server.Users.FindUser(event.Client)

			if user == nil {
				continue
			}

			user.Mutex.Lock()
			user.Name = event.Name
			user.Mutex.Unlock()

			logger := user.Logger()
			logger.Info().Msg("client has new name")
			server.NotifyNameChange(ctx, user, event.Name)

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
		case packet := <-gamePackets:
			user := server.Users.FindUser(packet.Client)

			if user == nil {
				continue
			}

			if user.GetServer() != packet.Server {
				continue
			}

			logger := user.Logger()

			parseData := packet.Packet.Data
			messages, err := game.Read(parseData, false)
			if err != nil {
				logger.Debug().
					Err(err).
					Msg("cluster -> client (failed to decode message)")

				// Forward it anyway
				user.Send(game.GamePacket{
					Channel: uint8(packet.Packet.Channel),
					Data:    packet.Packet.Data,
				})
				continue
			}

			channel := uint8(packet.Packet.Channel)

			// Sometimes clients are expecting messages to follow
			// each other directly; this is one of those cases
			// (arbitrary message passing between clients) and took
			// me too many hours of debugging
			if len(messages) > 0 && messages[0].Type() == game.N_CLIENT {
				logger.Debug().
					Str("type", game.N_CLIENT.String()).
					Msgf("cluster -> client (%d messages)", len(messages)-1)

				user.Send(game.GamePacket{
					Channel: channel,
					Data:    packet.Packet.Data,
				})
				continue
			}

			// As opposed to client -> server, we don't actually need to do any filtering
			// so we won't repackage the messages individually
			for _, message := range messages {
				logger.Debug().
					Str("type", message.Type().String()).
					Msg("cluster -> client")

				// Inject the auth domain to N_SERVINFO so the
				// client sends us N_CONNECT with their name
				// field filled
				if message.Type() == game.N_SERVINFO {
					info := message.Contents().(*game.ServerInfo)
					info.Domain = server.authDomain
					p := game.Packet{}
					p.PutInt(int32(game.N_SERVINFO))
					p.Put(*info)
					user.Send(game.GamePacket{
						Channel: channel,
						Data:    p,
					})
					continue
				}

				if message.Type() == game.N_SPAWNSTATE {
					state := message.Contents().(*game.SpawnState)
					user.Mutex.Lock()
					user.LifeSequence = state.LifeSequence
					user.Mutex.Unlock()
				}

				user.Send(game.GamePacket{
					Channel: channel,
					Data:    message.Data(),
				})
			}

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
	go server.manager.PruneServers(ctx)
	go server.matches.Poll(ctx)
}

func (server *Cluster) PollUsers(ctx context.Context, newConnections chan ingress.Connection) {
	newClients := server.Clients.ReceiveClients()

	for {
		select {
		case connection := <-newConnections:
			server.Clients.AddClient(connection)
		case client := <-newClients:
			user := server.Users.AddUser(ctx, client)
			go server.PollUser(ctx, user)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
	server.MapSender.Shutdown()
}
