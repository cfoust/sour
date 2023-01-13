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
	clients.Client
	Name      string
	Auth      *auth.AuthUser
	Verse     *verse.User
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

	Mutex sync.Mutex
	o     *UserOrchestrator
}

// Valid for the duration of the user's session on the cluster.
func (u *User) Context() context.Context {
	return u.Client.Connection.SessionContext()
}

func (u *User) Logger() zerolog.Logger {
	u.Mutex.Lock()
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
	u.Mutex.Unlock()

	return logger
}

func (u *User) GetServer() *servers.GameServer {
	u.Mutex.Lock()
	server := u.Server
	u.Mutex.Unlock()
	return server
}

func (u *User) ServerSessionContext() context.Context {
	u.Mutex.Lock()
	ctx := u.serverSessionCtx
	u.Mutex.Unlock()
	return ctx
}

func (u *User) GetServerName() string {
	serverName := "???"
	server := u.GetServer()
	if server != nil {
		serverName = server.GetFormattedReference()
	} else {
		if u.Connection.Type() == ingress.ClientTypeWS {
			serverName = "web"
		}
	}

	// TODO space

	return serverName
}

func (u *User) GetFormattedName() string {
	name := u.GetName()

	if u.Auth != nil {
		name = game.Blue(name)
	}

	return name
}

func (u *User) SendServerMessage(message string) {
	u.Client.SendMessage(fmt.Sprintf("%s %s", game.Yellow("sour"), message))
}

func (u *User) Reference() string {
	// TODO space
	u.Mutex.Lock()
	server := u.Server
	reference := u.Name
	if server != nil {
		reference = fmt.Sprintf("%s (%s)", u.Name, server.Reference())
	}
	u.Mutex.Unlock()
	return reference
}

func (u *User) GetName() string {
	u.Mutex.Lock()
	name := u.Name
	u.Mutex.Unlock()
	return name
}

func (u *User) GetAuth() *auth.AuthUser {
	u.Mutex.Lock()
	auth := u.Auth
	u.Mutex.Unlock()
	return auth
}

func (u *User) AnnounceELO() {
	u.Mutex.Lock()
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
	u.Mutex.Unlock()

	u.SendServerMessage(result)
}

func (u *User) HydrateELOState(ctx context.Context, authUser *auth.AuthUser) error {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	elo := NewELOState(u.o.Duels)

	for _, duel := range u.o.Duels {
		state, err := LoadELOState(ctx, u.o.redis, authUser.Discord.Id, duel.Name)
		if err != nil {
			return err
		}

		elo.Ratings[duel.Name] = state
	}

	u.ELO = elo

	return nil
}

func (u *User) SaveELOState(ctx context.Context) error {
	if u.Auth == nil {
		return nil
	}

	u.Mutex.Lock()
	defer u.Mutex.Unlock()

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
	if u.Connection.NetworkStatus() == ingress.NetworkStatusDisconnected {
		log.Warn().Msgf("client not connected to cluster but attempted connect")
		return nil, fmt.Errorf("client not connected to cluster")
	}

	u.DelayMessages()

	connected := make(chan bool, 1)

	log.Info().Str("server", server.Reference()).
		Msg("client connecting to server")

	u.Mutex.Lock()
	if u.Server != nil {
		u.Server.SendDisconnect(u.Id)
		u.cancel()

		// Remove all the other clients from this client's perspective
		u.o.Mutex.Lock()
		clients, ok := u.o.Servers[u.Server]
		if ok {
			for _, otherUser := range clients {
				if u == otherUser {
					continue
				}

				// Send N_CDIS
				otherUser.Mutex.Lock()
				packet := game.Packet{}
				packet.PutInt(int32(game.N_CDIS))
				packet.PutInt(int32(otherUser.Num))
				u.Connection.Send(game.GamePacket{
					Channel: 1,
					Data:    packet,
				})
				otherUser.Mutex.Unlock()
			}
		}
		u.o.Mutex.Unlock()
	}
	u.Server = server
	server.Connecting <- true
	u.Status = clients.ClientStatusConnecting
	sessionCtx, cancel := context.WithCancel(u.Connection.SessionContext())
	u.serverSessionCtx = sessionCtx
	u.cancel = cancel
	u.Mutex.Unlock()

	server.SendConnect(u.Id)

	serverName := server.Reference()
	if target != "" {
		serverName = target
	}
	u.Connection.Connect(serverName, server.Hidden, shouldCopy)

	// Give the client one second to connect.
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		connectCtx, cancel := context.WithTimeout(sessionCtx, time.Second*1)

		defer cancel()
		defer func() {
			<-server.Connecting
		}()

		for {
			if u.GetStatus() == clients.ClientStatusConnected {
				connected <- true
				return
			}

			select {
			case <-tick.C:
				continue
			case <-u.Connection.SessionContext().Done():
				connected <- false
				return
			case <-connectCtx.Done():
				u.RestoreMessages()
				connected <- false
				return
			}
		}
	}()

	return connected, nil
}

// Mark the client's status as disconnected and cancel its session context.
// Called both when the client disconnects from ingress AND when the server kicks them out.
func (u *User) DisconnectFromServer() error {
	u.Mutex.Lock()
	if u.Server != nil {
		u.Server.SendDisconnect(u.Client.Id)
	}
	u.Server = nil
	u.Client.Status = clients.ClientStatusDisconnected
	if u.cancel != nil {
		u.cancel()
	}
	u.Mutex.Unlock()

	return nil
}

type UserOrchestrator struct {
	Duels   []config.DuelType
	Users   []*User
	Servers map[*servers.GameServer][]*User
	Mutex   sync.Mutex

	redis *redis.Client
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
	case <-user.Context().Done():
		user.DisconnectFromServer()
		return
	case <-ctx.Done():
		return
	}
}

func (u *UserOrchestrator) AddUser(ctx context.Context, client *clients.Client) *User {
	u.Mutex.Lock()
	user := User{
		Client: *client,
		Name:   "unnamed",
		o:      u,
	}
	u.Users = append(u.Users, &user)
	u.Mutex.Unlock()

	go u.PollUser(ctx, &user)
	return &user
}

func (u *UserOrchestrator) FindUser(id uint16) *User {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()
	for _, user := range u.Users {
		if user.Client.Id != uint16(id) {
			continue
		}

		return user
	}

	return nil
}
