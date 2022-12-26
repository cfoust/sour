package clients

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

type CommandResult struct {
	Handled  bool
	Err      error
	Response string
}

type ClusterCommand struct {
	Command  string
	Response chan CommandResult
}

type ClientType uint8

const (
	ClientTypeWS = iota
	ClientTypeENet
)

// The status of the client's connection to the cluster.
type ClientNetworkStatus uint8

const (
	ClientNetworkStatusConnected = iota
	ClientNetworkStatusDisconnected
)

// The status of the client's connection to their game server.
type ClientStatus uint8

const (
	ClientStatusConnecting = iota
	ClientStatusConnected
	ClientStatusDisconnected
)

type NetworkClient interface {
	// Lasts for the duration of the client's connection to its ingress.
	SessionContext() context.Context
	NetworkStatus() ClientNetworkStatus
	Host() string
	Type() ClientType
	// Tell the client that we've connected
	Connect(name string, internal bool, owned bool)
	// Messages going to the client
	Send(packet game.GamePacket)
	// Messages going to the server
	ReceivePackets() <-chan game.GamePacket
	// Clients can issue commands out-of-band
	// Commands sent in ordinary game packets are interpreted anyway
	ReceiveCommands() <-chan ClusterCommand
	// When the client disconnects on its own
	ReceiveDisconnect() <-chan bool
	// When the client authenticates
	ReceiveAuthentication() <-chan *auth.User
	// WS clients can put chat in the chat bar; ENet clients cannot
	SendGlobalChat(message string)
	// Forcibly disconnect this client
	Disconnect(reason int, message string)
	Destroy()
}

type Client struct {
	Id    uint16
	Mutex sync.Mutex

	Name string
	// Whether the client is connected (or connecting) to a game server
	Status    ClientStatus
	User      *auth.User
	Challenge *auth.Challenge
	ELO       *ELOState
	Server    *servers.GameServer

	Connection NetworkClient

	// The ID of the client on the Sauer server
	ClientNum int32

	// True when the user is loading the map
	delayMessages bool
	messageQueue  []string

	// Created when the user connects to a server and canceled when they
	// leave, regardless of reason (network or being disconnected by the
	// server)
	// This is NOT the same thing as Client.Connection.SessionContext(), which refers to
	// the lifecycle of the client's ingress connection
	serverSessionCtx context.Context
	cancel           context.CancelFunc

	Authentication chan *auth.User

	// XXX This is nasty but to make the API nice, Clients have to be able
	// to see the list of clients. This could/should be refactored someday.
	manager *ClientManager
}

func (c *Client) ReceiveAuthentication() <-chan *auth.User {
	// WS clients do their own auth (for now)
	if c.Connection.Type() == ClientTypeWS {
		return c.Connection.ReceiveAuthentication()
	}

	return c.Authentication
}

func (c *Client) GetStatus() ClientStatus {
	c.Mutex.Lock()
	status := c.Status
	c.Mutex.Unlock()
	return status
}

func (c *Client) GetClientNum() int32 {
	c.Mutex.Lock()
	num := c.ClientNum
	c.Mutex.Unlock()
	return num
}

func (c *Client) GetName() string {
	c.Mutex.Lock()
	name := c.Name
	c.Mutex.Unlock()
	return name
}

func (c *Client) GetFormattedName() string {
	name := c.GetName()
	user := c.GetUser()
	
	if user != nil {
		name = game.Blue(name)
	}

	return name
}

func (c *Client) GetServerName() string {
	serverName := "???"
	server := c.GetServer()
	if server != nil {
		serverName = server.GetFormattedReference()
	} else {
		if c.Connection.Type() == ClientTypeWS {
			serverName = "main menu"
		}
	}

	return serverName
}

func (c *Client) GetUser() *auth.User {
	c.Mutex.Lock()
	user := c.User
	c.Mutex.Unlock()
	return user
}

func (c *Client) ServerSessionContext() context.Context {
	c.Mutex.Lock()
	ctx := c.serverSessionCtx
	c.Mutex.Unlock()
	return ctx
}

func (c *Client) GetServer() *servers.GameServer {
	c.Mutex.Lock()
	server := c.Server
	c.Mutex.Unlock()
	return server
}

func (c *Client) DelayMessages() {
	c.Mutex.Lock()
	c.delayMessages = true
	c.Mutex.Unlock()
}

func (c *Client) RestoreMessages() {
	c.Mutex.Lock()
	c.delayMessages = false
	c.Mutex.Unlock()
	c.sendQueuedMessages()
}

func (c *Client) sendQueuedMessages() {
	c.Mutex.Lock()
	for _, message := range c.messageQueue {
		c.sendMessage(message)
	}
	c.messageQueue = make([]string, 0)
	c.Mutex.Unlock()
}

func (c *Client) Reference() string {
	c.Mutex.Lock()
	server := c.Server
	reference := c.Name
	if server != nil {
		reference = fmt.Sprintf("%s (%s)", c.Name, server.Reference())
	}
	c.Mutex.Unlock()
	return reference
}

func (c *Client) sendMessage(message string) {
	packet := game.Packet{}
	packet.PutInt(int32(game.N_SERVMSG))
	packet.PutString(message)
	c.Connection.Send(game.GamePacket{
		Channel: 1,
		Data:    packet,
	})
}

func (c *Client) AnnounceELO() {
	c.Mutex.Lock()
	result := "ratings: "
	for _, duel := range c.manager.Duels {
		name := duel.Name
		state := c.ELO.Ratings[name]
		result += fmt.Sprintf(
			"%s %d (%s-%s-%s) ",
			name,
			state.Rating,
			game.Green(fmt.Sprint(state.Wins)),
			game.Yellow(fmt.Sprint(state.Draws)),
			game.Red(fmt.Sprint(state.Losses)),
		)
	}
	c.Mutex.Unlock()

	c.SendServerMessage(result)
}

func (c *Client) HydrateELOState(ctx context.Context, user *auth.User) error {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	elo := NewELOState(c.manager.Duels)

	for _, duel := range c.manager.Duels {
		state, err := LoadELOState(ctx, c.manager.redis, user.Discord.Id, duel.Name)
		if err != nil {
			return err
		}

		elo.Ratings[duel.Name] = state
	}

	c.ELO = elo

	return nil
}

func (c *Client) SaveELOState(ctx context.Context) error {
	if c.User == nil {
		return nil
	}

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	for matchType, state := range c.ELO.Ratings {
		err := state.SaveState(ctx, c.manager.redis, c.User.Discord.Id, matchType)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) SendMessage(message string) {
	c.Mutex.Lock()
	if c.delayMessages {
		c.messageQueue = append(c.messageQueue, message)
	} else {
		c.sendMessage(message)
	}
	c.Mutex.Unlock()
}

func (c *Client) SendServerMessage(message string) {
	c.SendMessage(fmt.Sprintf("%s %s", game.Yellow("sour"), message))
}

func (c *Client) ConnectToServer(server *servers.GameServer, internal bool, owned bool) (<-chan bool, error) {
	if c.Connection.NetworkStatus() == ClientNetworkStatusDisconnected {
		log.Warn().Msgf("client not connected to cluster but attempted connect")
		return nil, fmt.Errorf("client not connected to cluster")
	}

	c.DelayMessages()

	connected := make(chan bool, 1)

	log.Info().Str("server", server.Reference()).
		Msg("client connecting to server")

	c.Mutex.Lock()
	if c.Server != nil {
		c.Server.SendDisconnect(c.Id)
		c.cancel()

		// Remove all the other clients from this client's perspective
		c.manager.Mutex.Lock()
		for client, _ := range c.manager.State {
			if client == c || client.GetServer() != c.Server {
				continue
			}

			// Send N_CDIS
			client.Mutex.Lock()
			packet := game.Packet{}
			packet.PutInt(int32(game.N_CDIS))
			packet.PutInt(int32(client.ClientNum))
			c.Connection.Send(game.GamePacket{
				Channel: 1,
				Data:    packet,
			})
			client.Mutex.Unlock()
		}
		c.manager.Mutex.Unlock()
	}
	c.Server = server
	server.Connecting <- true
	c.Status = ClientStatusConnecting
	sessionCtx, cancel := context.WithCancel(c.Connection.SessionContext())
	c.serverSessionCtx = sessionCtx
	c.cancel = cancel
	c.Mutex.Unlock()

	server.SendConnect(c.Id)
	c.Connection.Connect(server.Reference(), internal, owned)

	// Give the client one second to connect.
	go func() {
		tick := time.NewTicker(50 * time.Millisecond)
		connectCtx, cancel := context.WithTimeout(sessionCtx, time.Second*1)

		defer cancel()
		defer func() {
			<-server.Connecting
		}()

		for {
			if c.GetStatus() == ClientStatusConnected {
				connected <- true
				return
			}

			select {
			case <-tick.C:
				continue
			case <-c.Connection.SessionContext().Done():
				connected <- false
				return
			case <-connectCtx.Done():
				c.RestoreMessages()
				connected <- false
				return
			}
		}
	}()

	return connected, nil
}

// Mark the client's status as disconnected and cancel its session context.
// Called both when the client disconnects from ingress AND when the server kicks them out.
func (c *Client) DisconnectFromServer() error {
	c.Mutex.Lock()
	if c.Server != nil {
		c.Server.SendDisconnect(c.Id)
	}
	c.Server = nil
	c.Status = ClientStatusDisconnected
	if c.cancel != nil {
		c.cancel()
	}
	c.Mutex.Unlock()

	return nil
}

type ClientManager struct {
	Duels      []config.DuelType
	State      map[*Client]struct{}
	Mutex      sync.Mutex
	redis      *redis.Client
	newClients chan *Client
}

func NewClientManager(redis *redis.Client, duels []config.DuelType) *ClientManager {
	return &ClientManager{
		Duels:      duels,
		redis:      redis,
		State:      make(map[*Client]struct{}),
		newClients: make(chan *Client, 16),
	}
}

func (c *ClientManager) newClientID() (uint16, error) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	for attempts := 0; attempts < math.MaxUint16; attempts++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		truncated := uint16(number.Uint64())

		taken := false
		for client, _ := range c.State {
			if client.Id == truncated {
				taken = true
			}
		}
		if taken {
			continue
		}

		return truncated, nil
	}

	return 0, errors.New("Failed to assign client ID")
}

func (c *ClientManager) AddClient(networkClient NetworkClient) error {
	id, err := c.newClientID()
	if err != nil {
		return err
	}

	client := Client{
		Id:             id,
		Name:           "unnamed",
		ELO:            NewELOState(c.Duels),
		Connection:     networkClient,
		Status:         ClientStatusDisconnected,
		manager:        c,
		delayMessages:  false,
		messageQueue:   make([]string, 0),
		Authentication: make(chan *auth.User, 1),
	}

	c.Mutex.Lock()
	c.State[&client] = struct{}{}
	c.Mutex.Unlock()

	c.newClients <- &client

	return nil
}

func (c *ClientManager) RemoveClient(networkClient NetworkClient) {
	c.Mutex.Lock()

	for client, _ := range c.State {
		if client.Connection != networkClient {
			continue
		}

		client.DisconnectFromServer()
		delete(c.State, client)
		break
	}

	c.Mutex.Unlock()
}

func (c *ClientManager) ReceiveClients() <-chan *Client {
	return c.newClients
}

func (c *ClientManager) FindClient(id uint16) *Client {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	for client, _ := range c.State {
		if client.Id != uint16(id) {
			continue
		}

		return client
	}

	return nil
}
