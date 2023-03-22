package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server"
	"github.com/cfoust/sour/pkg/utils"

	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/state"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
	"gorm.io/gorm"
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

	Auth       *state.User
	sessionLog *state.Session

	ELO *ELOState

	// True when the user is loading the map
	delayMessages bool
	messageQueue  []string

	Authentication    chan *state.User
	serverConnections chan ConnectionEvent

	to chan TrackedPacket

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

	Mutex      deadlock.RWMutex
	queueMutex deadlock.RWMutex
	o          *UserOrchestrator
}

func (c *User) ReceiveConnections() <-chan ConnectionEvent {
	return c.serverConnections
}

func (c *User) ReceiveAuthentication() <-chan *state.User {
	// WS clients do their own auth (for now)
	if c.Connection.Type() == ingress.ClientTypeWS {
		return c.Connection.ReceiveAuthentication()
	}

	return c.Authentication
}

func (u *User) GetSessionID() string {
	return u.sessionLog.UUID[:5]
}

func (u *User) Logger() zerolog.Logger {
	u.Mutex.RLock()
	logger := log.With().
		Uint32("id", uint32(u.Id)).
		Str("session", u.GetSessionID()).
		Str("type", u.Connection.DeviceType()).
		Str("name", u.Name).
		Logger()

	if u.Auth != nil {
		logger = logger.With().
			Str("discord", fmt.Sprintf(
				"%s#%s",
				u.Auth.Username,
				u.Auth.Discriminator,
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

	return auth.UUID
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

func (u *User) IsAtHome(ctx context.Context) (bool, error) {
	if u.Auth == nil {
		return false, nil
	}

	entity, err := u.GetSpaceEntity(ctx)
	if err != nil {
		return false, nil
	}

	return u.Auth.HomeID == entity.ID, nil
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

func (u *User) GetHomeSpace(ctx context.Context) (*state.Space, error) {
	auth := u.GetAuth()
	if auth == nil {
		return nil, fmt.Errorf("user is not logged in")
	}

	var space state.Space
	query := state.Space{}
	query.ID = auth.HomeID
	err := u.o.db.WithContext(ctx).Where(query).First(&space).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// We only create home spaces on demand
	if err == gorm.ErrRecordNotFound {
		userSpace, err := u.o.verse.NewSpace(ctx, auth)
		if err != nil {
			return nil, err
		}

		err = userSpace.SetDescription(
			ctx,
			fmt.Sprintf("%s [home]", u.GetName()),
		)
		if err != nil {
			return nil, err
		}

		space, err := userSpace.GetSpace(ctx)
		if err != nil {
			return nil, err
		}

		auth.HomeID = space.ID
		err = u.o.db.WithContext(ctx).Save(&auth).Error
		if err != nil {
			return nil, err
		}

		return space, nil
	}

	return &space, nil
}

func (u *User) GetSpaceEntity(ctx context.Context) (*state.Space, error) {
	instance := u.GetSpace()
	if instance == nil {
		return nil, fmt.Errorf("user not in space")
	}

	if instance.Space == nil {
		return nil, fmt.Errorf("space is not a user space")
	}

	space, err := instance.Space.GetSpace(ctx)
	if err != nil {
		return nil, err
	}

	return space, nil
}

func (u *User) IsOwner(ctx context.Context) (bool, error) {
	if u.Auth == nil {
		return false, nil
	}

	entity, err := u.GetSpaceEntity(ctx)
	if err != nil {
		return false, nil
	}

	return u.Auth.ID == entity.OwnerID, nil
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
	c.queueMutex.Lock()
	for _, message := range c.messageQueue {
		c.sendMessage(message)
	}
	c.messageQueue = make([]string, 0)
	c.queueMutex.Unlock()
}

func (c *User) sendMessage(message string) {
	c.Send(P.ServerMessage{Text: message})
}

func (u *User) queueMessage(message string) {
	u.Mutex.RLock()
	delayed := u.delayMessages
	u.Mutex.RUnlock()

	if delayed {
		u.queueMutex.Lock()
		u.messageQueue = append(u.messageQueue, message)
		u.queueMutex.Unlock()
		return
	}

	u.sendMessage(message)
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

func (u *User) GetAuth() *state.User {
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

func (u *User) HydrateELOState(ctx context.Context, authUser *state.User) error {
	u.Mutex.Lock()
	defer u.Mutex.Unlock()

	elo := NewELOState(u.o.Duels)

	for _, duel := range u.o.Duels {
		state, err := LoadELOState(ctx, u.o.db, authUser, duel.Name)

		if err == nil {
			elo.Ratings[duel.Name] = state
			continue
		}

		return err
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
		err := state.SaveState(ctx, u.o.db, u.Auth, matchType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *User) LogVisit(ctx context.Context) error {
	visit := state.Visit{
		SessionID: u.sessionLog.ID,
	}
	visit.Start = time.Now()

	if auth := u.GetAuth(); auth != nil {
		visit.UserID = auth.ID
	}

	if server := u.GetServer(); server != nil {
		if server.Alias != "" {
			visit.Location = server.Alias
		}
	}

	// TODO maps?

	if space := u.GetSpace(); space != nil && space.Space != nil {
		entity, err := space.Space.GetSpace(ctx)
		if err == nil {
			visit.SpaceID = entity.ID
			visit.MapPointerID = entity.MapPointerID
		}
	}

	err := u.o.db.WithContext(ctx).Save(&visit).Error
	if err != nil {
		return err
	}

	<-u.ServerSession.Ctx().Done()

	visit.End = time.Now()
	return u.o.db.WithContext(ctx).Save(&visit).Error
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

				u.Send(
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

	db    *gorm.DB
	verse *verse.Verse
}

func NewUserOrchestrator(db *gorm.DB, verse *verse.Verse, duels []config.DuelType) *UserOrchestrator {
	return &UserOrchestrator{
		Duels:   duels,
		Users:   make([]*User, 0),
		Servers: make(map[*servers.GameServer][]*User),
		db:      db,
		verse:   verse,
	}
}

func (u *UserOrchestrator) PollUser(ctx context.Context, user *User) {
	select {
	case <-user.Ctx().Done():
		logger := user.Logger()
		u.RemoveUser(user)

		user.sessionLog.End = time.Now()
		err := u.db.WithContext(ctx).Save(user.sessionLog).Error
		if err != nil {
			logger.Error().Err(err).Msg("failed to set session end")
		}
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

func getAddress(ctx context.Context, db *gorm.DB, address string) (*state.Host, error) {
	db = db.WithContext(ctx)

	var host state.Host
	err := db.Where(state.Host{
		UUID: address,
	}).First(&host).Error
	if err == nil {
		return &host, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	host = state.Host{
		UUID: address,
	}

	err = db.Create(&host).Error
	if err != nil {
		return nil, err
	}

	return &host, nil
}

func (u *UserOrchestrator) AddUser(ctx context.Context, connection ingress.Connection) (*User, error) {
	id, err := u.newSessionID()
	if err != nil {
		return nil, err
	}

	host, err := getAddress(ctx, u.db, utils.HashString(connection.Host()))
	if err != nil {
		return nil, err
	}

	sessionID := utils.HashString(fmt.Sprintf("%d-%s", id, connection.Host()))
	sessionLog := state.Session{
		HostID: host.ID,
		UUID:   sessionID,
		Device: connection.DeviceType(),
	}
	sessionLog.Start = time.Now()

	err = u.db.WithContext(ctx).Save(&sessionLog).Error
	if err != nil {
		return nil, err
	}

	u.Mutex.Lock()
	user := User{
		Id:                id,
		Status:            UserStatusDisconnected,
		Connection:        connection,
		Session:           connection.Session(),
		sessionLog:        &sessionLog,
		ELO:               NewELOState(u.Duels),
		Name:              "unnamed",
		From:              P.NewMessageProxy(true),
		To:                P.NewMessageProxy(false),
		to:                make(chan TrackedPacket, 1000),
		Authentication:    make(chan *state.User),
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
