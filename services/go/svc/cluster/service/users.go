package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type User struct {
	Name      string
	Client    *clients.Client
	Auth      *auth.AuthUser
	Challenge *auth.Challenge
	ELO       *ELOState

	Server *servers.GameServer
	Space  *verse.SpaceInstance

	// Created when the user connects to a server and canceled when they
	// leave, regardless of reason (network or being disconnected by the
	// server)
	// This is NOT the same thing as Client.Connection.SessionContext(), which refers to
	// the lifecycle of the client's ingress connection
	serverSessionCtx context.Context
	cancel           context.CancelFunc

	mutex sync.Mutex
	o     *UserOrchestrator
}

func (u *User) Logger() zerolog.Logger {
	u.mutex.Lock()
	logger := log.With().Uint16("client", u.Client.Id).Str("name", u.Name).Logger()

	if u.Auth != nil {
		discord := u.Auth.Discord
		logger = logger.With().
			Str("discord", fmt.Sprintf(
				"%s#%s",
				discord.Username,
				discord.Discriminator,
			)).Logger()
	}

	if u.Server != nil {
		logger = logger.With().Str("server", u.Server.Reference()).Logger()
	}
	u.mutex.Unlock()

	return logger
}

func (u *User) GetServer() *servers.GameServer {
	u.mutex.Lock()
	server := u.Server
	u.mutex.Unlock()
	return server
}

func (u *User) GetClient() *clients.Client {
	u.mutex.Lock()
	client := u.Client
	u.mutex.Unlock()
	return client
}

func (u *User) ServerSessionContext() context.Context {
	u.mutex.Lock()
	ctx := u.serverSessionCtx
	u.mutex.Unlock()
	return ctx
}

func (u *User) GetServerName() string {
	serverName := "???"
	server := u.GetServer()
	if server != nil {
		serverName = server.GetFormattedReference()
	} else {
		client := u.GetClient()
		if client.Connection.Type() == ingress.ClientTypeWS {
			serverName = "web"
		}
	}

	// TODO space

	return serverName
}

func (u *User) Reference() string {
	// TODO
	u.mutex.Lock()
	server := u.Server
	reference := u.Name
	if server != nil {
		reference = fmt.Sprintf("%s (%s)", u.Name, server.Reference())
	}
	u.mutex.Unlock()
	return reference
}

func (u *User) AnnounceELO() {
	u.mutex.Lock()
	result := "ratings: "
	for _, duel := range u.o.Duels {
		name := duel.Name
		state := u.ELO.Ratings[name]
		result += fmt.Sprintf(
			"%s %d (%s-%s-%s) ",
			name,
			state.Rating,
			game.Green(fmt.Sprint(state.Wins)),
			game.Yellow(fmt.Sprint(state.Draws)),
			game.Red(fmt.Sprint(state.Losses)),
		)
	}
	u.mutex.Unlock()

	u.Client.SendServerMessage(result)
}

func (c *User) HydrateELOState(ctx context.Context, user *auth.AuthUser) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	elo := NewELOState(c.o.Duels)

	for _, duel := range c.o.Duels {
		state, err := LoadELOState(ctx, c.o.redis, user.Discord.Id, duel.Name)
		if err != nil {
			return err
		}

		elo.Ratings[duel.Name] = state
	}

	c.ELO = elo

	return nil
}

func (u *User) SaveELOState(ctx context.Context) error {
	if u.Auth == nil {
		return nil
	}

	u.mutex.Lock()
	defer u.mutex.Unlock()

	for matchType, state := range u.ELO.Ratings {
		err := state.SaveState(ctx, u.o.redis, u.Auth.Discord.Id, matchType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *User) ConnectToSpace(server *servers.GameServer, id string) (<-chan bool, error) {
	return u.ConnectToServer(server, id, false, true)
}

func (u *User) Connect(server *servers.GameServer) (<-chan bool, error) {
	return u.ConnectToServer(server, "", false, false)
}

func (u *User) ConnectToServer(server *servers.GameServer, target string, shouldCopy bool, isSpace bool) (<-chan bool, error) {
	client := u.GetClient()
	if client.Connection.NetworkStatus() == ingress.NetworkStatusDisconnected {
		log.Warn().Msgf("client not connected to cluster but attempted connect")
		return nil, fmt.Errorf("client not connected to cluster")
	}

	client.DelayMessages()

	connected := make(chan bool, 1)

	log.Info().Str("server", server.Reference()).
		Msg("client connecting to server")

	u.mutex.Lock()
	if u.Server != nil {
		u.Server.SendDisconnect(client.Id)
		u.cancel()

		// Remove all the other clients from this client's perspective
		u.o.mutex.Lock()
		clients, ok := u.o.Servers[u.Server]
		if ok {
			for _, otherUser := range clients {
				otherClient := otherUser.GetClient()
				if client == otherClient {
					continue
				}

				// Send N_CDIS
				otherUser.mutex.Lock()
				packet := game.Packet{}
				packet.PutInt(int32(game.N_CDIS))
				packet.PutInt(int32(otherClient.Num))
				client.Connection.Send(game.GamePacket{
					Channel: 1,
					Data:    packet,
				})
				otherUser.mutex.Unlock()
			}
		}
		u.o.mutex.Unlock()
	}
	u.Server = server
	server.Connecting <- true
	client.Status = clients.ClientStatusConnecting
	sessionCtx, cancel := context.WithCancel(client.Connection.SessionContext())
	u.serverSessionCtx = sessionCtx
	u.cancel = cancel
	u.mutex.Unlock()

	server.SendConnect(client.Id)

	serverName := server.Reference()
	if target != "" {
		serverName = target
	}
	client.Connection.Connect(serverName, server.Hidden, shouldCopy)

	// Give the client one second to connect.
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		connectCtx, cancel := context.WithTimeout(sessionCtx, time.Second*1)

		defer cancel()
		defer func() {
			<-server.Connecting
		}()

		for {
			if client.GetStatus() == clients.ClientStatusConnected {
				connected <- true
				return
			}

			select {
			case <-tick.C:
				continue
			case <-client.Connection.SessionContext().Done():
				connected <- false
				return
			case <-connectCtx.Done():
				client.RestoreMessages()
				connected <- false
				return
			}
		}
	}()

	return connected, nil
}

// Mark the client's status as disconnected and cancel its session context.
// Called both when the client disconnects from ingress AND when the server kicks them out.
func (c *User) DisconnectFromServer() error {
	c.mutex.Lock()
	if c.Server != nil {
		c.Server.SendDisconnect(c.Client.Id)
	}
	c.Server = nil
	c.Client.Status = clients.ClientStatusDisconnected
	if c.cancel != nil {
		c.cancel()
	}
	c.mutex.Unlock()

	return nil
}

type UserOrchestrator struct {
	Duels   []config.DuelType
	Users   []*User
	Servers map[*servers.GameServer][]*User
	redis   *redis.Client
	mutex   sync.Mutex
}

func NewUserOrchestrator(redis *redis.Client, duels []config.DuelType) *UserOrchestrator {
	return &UserOrchestrator{
		Duels:   duels,
		Users:   make([]*User, 0),
		Servers: make(map[*servers.GameServer][]*User),
		redis:   redis,
	}
}

func (u *UserOrchestrator) PollUser(ctx context.Context, user *User) {

	select {
	case <-user.Client.Connection.SessionContext().Done():
		user.DisconnectFromServer()
		return
	case <-ctx.Done():
		return
	}
}

func (u *UserOrchestrator) AddUser(ctx context.Context, client *clients.Client) {
	u.mutex.Lock()
	user := User{
		Name:   "unnamed",
		Client: client,
		o:      u,
	}
	u.Users = append(u.Users, &user)
	u.mutex.Unlock()

	go u.PollUser(ctx, &user)
}
