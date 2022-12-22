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
	"github.com/cfoust/sour/svc/cluster/servers"

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
	// Get a string identifier for this client for logging purposes.
	// This does not have to be unique.
	Reference() string
	NetworkStatus() ClientNetworkStatus
	Host() string
	Type() ClientType
	// Tell the client that we've connected
	Connect()
	// Messages going to the client
	Send(packet game.GamePacket)
	// Messages going to the server
	ReceivePackets() <-chan game.GamePacket
	// Clients can issue commands out-of-band
	// Commands sent in ordinary game packets are interpreted anyway
	ReceiveCommands() <-chan ClusterCommand
	// When the client disconnects on its own
	ReceiveDisconnect() <-chan bool
	// Forcibly disconnect this client
	Disconnect(reason int, message string)
	Destroy()
}

type Client struct {
	Id         uint16
	Connection NetworkClient

	// The ID of the client on the Sauer server
	ClientNum int32
	Server    *servers.GameServer
	// Whether the client is connected (or connecting) to a game server
	Status ClientStatus

	Mutex sync.Mutex

	// Created when the user connects to a server and canceled when they
	// leave, regardless of reason (network or being disconnected by the
	// server)
	// This is NOT the same thing as Client.SessionContext(), which refers to
	// the lifecycle of the client's ingress connection
	serverSessionCtx context.Context
	cancel           context.CancelFunc

	// XXX This is nasty but to make the API nice, Clients have to be able
	// to see the list of clients. This could/should be refactored someday.
	manager          *ClientManager
}

func (c *Client) GetStatus() ClientStatus {
	c.Mutex.Lock()
	status := c.Status
	c.Mutex.Unlock()
	return status
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

func (c *Client) SendServerMessage(message string) {
	packet := game.Packet{}
	packet.PutInt(int32(game.N_SERVMSG))
	message = fmt.Sprintf("%s %s", game.Yellow("sour"), message)
	packet.PutString(message)
	c.Connection.Send(game.GamePacket{
		Channel: 1,
		Data:    packet,
	})
}

func (c *Client) ConnectToServer(server *servers.GameServer) (<-chan bool, error) {
	if c.Connection.NetworkStatus() == ClientNetworkStatusDisconnected {
		log.Warn().Msgf("client not connected to cluster but attempted connect")
		return nil, fmt.Errorf("client not connected to cluster")
	}

	connected := make(chan bool, 1)

	log.Info().Str("server", server.Reference()).
		Msg("client connecting to server")

	c.Mutex.Lock()
	if c.Server != nil {
		c.Server.SendDisconnect(c.Id)

		// Remove all the other clients from this client's perspective
		c.manager.Mutex.Lock()
		for client, _ := range c.manager.state {
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
	c.Connection.Connect()

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
	state      map[*Client]struct{}
	Mutex      sync.Mutex
	newClients chan *Client
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		state:      make(map[*Client]struct{}),
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
		for client, _ := range c.state {
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
		Id:         id,
		Connection: networkClient,
		Status:     ClientStatusDisconnected,
		manager:    c,
	}

	c.Mutex.Lock()
	c.state[&client] = struct{}{}
	c.Mutex.Unlock()

	c.newClients <- &client

	return nil
}

func (c *ClientManager) RemoveClient(networkClient NetworkClient) {
	c.Mutex.Lock()

	for client, _ := range c.state {
		if client.Connection != networkClient {
			continue
		}

		client.DisconnectFromServer()
		delete(c.state, client)
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
	for client, _ := range c.state {
		if client.Id != uint16(id) {
			continue
		}

		return client
	}

	return nil
}
