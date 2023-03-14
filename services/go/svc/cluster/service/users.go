package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server"
	"github.com/cfoust/sour/pkg/utils"

	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type TrackedPacket struct {
	Packet P.Packet
	Error  chan error
}

type ConnectionEvent struct {
	Server *servers.GameServer
}

// The status of the user's connection to their game server.
type UserStatus uint8

const (
	UserStatusConnecting = iota
	UserStatusConnected
	UserStatusDisconnected
)

type User struct {
	*utils.Session

	Id ingress.ClientID
	// Whether the user is connected (or connecting) to a game server
	Status UserStatus
	Name   string

	Connection ingress.Connection

	// Created when the user connects to a server and canceled when they
	// leave, regardless of reason (network or being disconnected by the
	// server)
	// This is NOT the same thing as Client.Connection.SessionContext(), which refers to
	// the lifecycle of the client's ingress connection
	ServerSession utils.Session
	Server        *servers.GameServer
	ServerClient  *server.Client

	Auth  *auth.AuthUser
	Verse *verse.User
	ELO   *ELOState

	SessionUUID string

	// True when the user is loading the map
	delayMessages bool
	messageQueue  []string

	Authentication    chan *auth.AuthUser
	serverConnections chan ConnectionEvent

	to chan TrackedPacket

	// The user's home ID if they're not authenticated.
	TempHomeID string
	// The last server description sent to the user
	lastDescription string
	wasGreeted      bool
	sendingMap      bool
	autoexecKey     string

	Space *verse.SpaceInstance

	From *P.MessageProxy
	To   *P.MessageProxy

	RawFrom *utils.Topic[io.RawPacket]
	RawTo   *utils.Topic[io.RawPacket]

	Mutex deadlock.RWMutex
	o     *UserOrchestrator
}

func (c *User) ReceiveConnections() <-chan ConnectionEvent {
	return c.serverConnections
}

func (c *User) ReceiveAuthentication() <-chan *auth.AuthUser {
	// WS clients do their own auth (for now)
	if c.Connection.Type() == ingress.ClientTypeWS {
		return c.Connection.ReceiveAuthentication()
	}

	return c.Authentication
}

func (u *User) Logger() zerolog.Logger {
	u.Mutex.RLock()
	logger := log.With().
		Uint32("id", uint32(u.Id)).
		Str("session", u.SessionUUID).
		Str("type", u.Connection.DeviceType()).
		Str("name", u.Name).
		Logger()

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
	u.Mutex.RUnlock()

	return logger
}

func (u *User) GetClientNum() int {
	u.Mutex.RLock()
	num := int(u.ServerClient.CN)
	u.Mutex.RUnlock()
	return num
}

func (u *User) GetID() string {
	u.Mutex.RLock()
	auth := u.Auth
	u.Mutex.RUnlock()

	if auth == nil {
		return ""
	}

	return auth.GetID()
}

func (c *User) GetStatus() UserStatus {
	c.Mutex.RLock()
	status := c.Status
	c.Mutex.RUnlock()
	return status
}

func (c *User) DelayMessages() {
	c.Mutex.Lock()
	c.delayMessages = true
	c.Mutex.Unlock()
}

func (c *User) RestoreMessages() {
	c.Mutex.Lock()
	c.delayMessages = false
	c.Mutex.Unlock()
	c.sendQueuedMessages()
}

func (c *User) SendChannel(channel uint8, messages ...P.Message) <-chan error {
	out := make(chan error, 1)
	c.to <- TrackedPacket{
		Packet: P.Packet{
			Channel:  channel,
			Messages: messages,
		},
		Error: out,
	}
	return out
}

func (c *User) SendChannelSync(channel uint8, messages ...P.Message) error {
	return <-c.SendChannel(channel, messages...)
}

func (c *User) Send(messages ...P.Message) <-chan error {
	return c.SendChannel(1, messages...)
}

func (c *User) SendSync(messages ...P.Message) error {
	return c.SendChannelSync(1, messages...)
}

func (c *User) ReceiveToMessages() <-chan TrackedPacket {
	return c.to
}

func (u *User) IsLoggedIn() bool {
	u.Mutex.RLock()
	auth := u.Auth
	u.Mutex.RUnlock()

	return auth != nil
}

func (u *User) GetVerse() *verse.User {
	u.Mutex.RLock()
	user := u.Verse
	u.Mutex.RUnlock()
	return user
}

func (u *User) IsAtHome(ctx context.Context) (bool, error) {
	space := u.GetSpace()
	if space == nil {
		return false, nil
	}

	user := u.GetVerse()
	if user == nil {
		return space.GetID() == u.TempHomeID, nil
	}

	home, err := user.GetHomeID(ctx)
	if err != nil {
		return false, err
	}

	isOwner, err := u.IsOwner(ctx)
	if err != nil {
		return false, err
	}

	return isOwner && space.GetID() == home, nil
}

func (u *User) GetServer() *servers.GameServer {
	u.Mutex.RLock()
	server := u.Server
	u.Mutex.RUnlock()
	return server
}

func (u *User) ServerSessionContext() context.Context {
	u.Mutex.RLock()
	ctx := u.ServerSession.Ctx()
	u.Mutex.RUnlock()
	return ctx
}

func (u *User) GetSpace() *verse.SpaceInstance {
	u.Mutex.RLock()
	space := u.Space
	u.Mutex.RUnlock()
	return space
}

func (u *User) IsOwner(ctx context.Context) (bool, error) {
	space := u.GetSpace()
	if space == nil {
		return false, nil
	}

	owner, err := space.GetOwner(ctx)
	if err != nil {
		return false, err
	}

	return owner == u.GetID(), nil
}

// SPAAAAAAAAACE
func (u *User) IsInSpace() bool {
	return u.GetSpace() != nil
}

func (u *User) GetServerName() string {
	serverName := "???"

	space := u.GetSpace()
	if space != nil {
		// Cached, but that's OK
		alias := space.Alias
		if alias != "" {
			return alias
		}

		return space.GetID()
	}

	server := u.GetServer()
	if server != nil {
		serverName = server.GetFormattedReference()
	} else {
		if u.Connection.Type() == ingress.ClientTypeWS {
			serverName = "web"
		}
	}

	return serverName
}

func (u *User) GetFormattedName() string {
	name := u.GetName()

	if u.Auth != nil {
		name = game.Blue(name)
	}

	return name
}

func (c *User) sendQueuedMessages() {
	c.Mutex.Lock()
	for _, message := range c.messageQueue {
		c.sendMessage(message)
	}
	c.messageQueue = make([]string, 0)
	c.Mutex.Unlock()
}

func (c *User) sendMessage(message string) {
	c.Send(P.ServerMessage{Text: message})
}

func (u *User) queueMessage(message string) {
	u.Mutex.Lock()
	if u.delayMessages {
		u.messageQueue = append(u.messageQueue, message)
	} else {
		u.sendMessage(message)
	}
	u.Mutex.Unlock()
}

func (u *User) Message(message string) {
	u.queueMessage(fmt.Sprintf("%s %s", game.Magenta("~>"), message))
}

func (u *User) RawMessage(message string) {
	u.queueMessage(message)
}

func (u *User) Reference() string {
	return fmt.Sprintf("%s (%s)", u.GetName(), u.GetServerName())
}

func (u *User) GetFormattedReference() string {
	return fmt.Sprintf("%s (%s)", u.GetFormattedName(), u.GetServerName())
}

func (u *User) GetName() string {
	u.Mutex.RLock()
	name := u.Name
	u.Mutex.RUnlock()
	return name
}

func (u *User) GetAuth() *auth.AuthUser {
	u.Mutex.RLock()
	auth := u.Auth
	u.Mutex.RUnlock()
	return auth
}

func (u *User) AnnounceELO() {
	u.Mutex.RLock()
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
	u.Mutex.RUnlock()

	u.Message(result)
}

func (u *User) HydrateELOState(ctx context.Context, authUser *auth.AuthUser) error {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	elo := NewELOState(u.o.Duels)

	for _, duel := range u.o.Duels {
		state, err := LoadELOState(ctx, u.o.redis, authUser.Discord.Id, duel.Name)

		if err == nil {
			elo.Ratings[duel.Name] = state
			continue
		}

		if err != redis.Nil {
			return err
		}
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

	logger := u.Logger()

	logger.Info().Msg("connect to server")

	oldServer := u.GetServer()
	if oldServer != nil {
		oldServer.Leave(uint32(u.Id))
		u.ServerSession.Cancel()

		// Remove all the other clients from this client's perspective
		u.o.Mutex.Lock()
		users, ok := u.o.Servers[oldServer]
		if ok {
			newUsers := make([]*User, 0)
			for _, otherUser := range users {
				if u == otherUser {
					continue
				}

				otherUser.Send(
					P.ClientDisconnected{
						Client: int32(otherUser.GetClientNum()),
					},
				)
				newUsers = append(newUsers, otherUser)
			}
			u.o.Servers[u.Server] = newUsers
		}
		u.o.Mutex.Unlock()
	}

	space := u.GetSpace()
	if space != nil {
		if space.Editing != nil {
			space.Editing.ClearClipboard(u.Id)
		}
	}

	u.Mutex.Lock()
	u.Space = nil
	u.Server = server
	u.Status = UserStatusConnecting
	u.ServerSession = utils.NewSession(u.Session.Ctx())
	u.Mutex.Unlock()

	connected := make(chan bool, 1)

	serverClient, serverConnected := server.Connect(uint32(u.Id))
	u.ServerClient = serverClient

	serverName := server.Reference()
	if target != "" {
		serverName = target
	}
	u.Connection.Connect(serverName, server.Hidden, shouldCopy)

	// Give the client one second to connect.
	go func() {
		connectCtx, cancel := context.WithTimeout(u.ServerSession.Ctx(), time.Second*1)
		defer cancel()

		select {
		case <-serverConnected:
			u.Mutex.Lock()
			u.Status = UserStatusConnected
			u.Mutex.Unlock()

			u.o.Mutex.Lock()
			users, ok := u.o.Servers[server]
			newUsers := make([]*User, 0)
			if ok {
				for _, otherUser := range users {
					if u == otherUser {
						continue
					}

					newUsers = append(newUsers, otherUser)
				}
			}
			newUsers = append(newUsers, u)
			u.o.Servers[u.Server] = newUsers
			u.o.Mutex.Unlock()

			connected <- true
			u.serverConnections <- ConnectionEvent{
				Server: server,
			}

		case <-u.Session.Ctx().Done():
			connected <- false
		case <-connectCtx.Done():
			u.RestoreMessages()
			connected <- false
		}
	}()

	return connected, nil
}

// Mark the client's status as disconnected and cancel its session context.
// Called both when the client disconnects from ingress AND when the server kicks them out.
func (u *User) DisconnectFromServer() error {
	logger := u.Logger()
	logger.Info().Str("host", u.Connection.Host()).Msg("user disconnected")

	server := u.GetServer()
	if server != nil {
		server.Leave(uint32(u.Id))
	}

	u.Mutex.Lock()
	u.Server = nil
	u.Space = nil
	u.Status = UserStatusDisconnected
	u.Mutex.Unlock()

	u.ServerSession.Cancel()

	return nil
}

type UserOrchestrator struct {
	Duels   []config.DuelType
	Users   []*User
	Servers map[*servers.GameServer][]*User
	Mutex   deadlock.RWMutex

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
	case <-user.Session.Ctx().Done():
		u.RemoveUser(user)
		return
	case <-ctx.Done():
		return
	}
}

func (u *UserOrchestrator) newSessionID() (ingress.ClientID, error) {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	for attempts := 0; attempts < math.MaxUint16; attempts++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		truncated := ingress.ClientID(number.Uint64())

		taken := false
		for _, user := range u.Users {
			if user.Id == truncated {
				taken = true
			}
		}
		if taken {
			continue
		}

		return truncated, nil
	}

	return 0, fmt.Errorf("Failed to assign client ID")
}

func (u *UserOrchestrator) AddUser(ctx context.Context, connection ingress.Connection) (*User, error) {
	id, err := u.newSessionID()
	if err != nil {
		return nil, err
	}

	sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(
		fmt.Sprintf("%d-%s", id, connection.Host()),
	)))[:5]

	u.Mutex.Lock()
	user := User{
		Id:                id,
		SessionUUID:       sessionID,
		Status:            UserStatusDisconnected,
		Connection:        connection,
		Session:           connection.Session(),
		ELO:               NewELOState(u.Duels),
		Name:              "unnamed",
		From:              P.NewMessageProxy(true),
		To:                P.NewMessageProxy(false),
		to:                make(chan TrackedPacket, 1000),
		Authentication:    make(chan *auth.AuthUser),
		serverConnections: make(chan ConnectionEvent),
		ServerSession:     utils.NewSession(ctx),
		o:                 u,
		RawFrom:           utils.NewTopic[io.RawPacket](),
		RawTo:             utils.NewTopic[io.RawPacket](),
	}
	u.Users = append(u.Users, &user)
	u.Mutex.Unlock()

	go u.PollUser(ctx, &user)

	logger := user.Logger()
	logger.Info().Str("host", user.Connection.Host()).Msg("user joined")

	return &user, nil
}

func (u *UserOrchestrator) RemoveUser(user *User) {
	u.Mutex.Lock()

	newUsers := make([]*User, 0)
	for _, other := range u.Users {
		if other == user {
			continue
		}
		newUsers = append(newUsers, other)
	}
	u.Users = newUsers

	for server, users := range u.Servers {
		serverUsers := make([]*User, 0)
		for _, other := range users {
			if other == user {
				continue
			}
			serverUsers = append(serverUsers, other)
		}

		u.Servers[server] = serverUsers
	}

	u.Mutex.Unlock()
}

func (u *UserOrchestrator) FindUser(id ingress.ClientID) *User {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()
	for _, user := range u.Users {
		if user.Id != id {
			continue
		}

		return user
	}

	return nil
}
