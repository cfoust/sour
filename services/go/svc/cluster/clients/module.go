package clients

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sync"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/servers"
)

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

type CommandResult struct {
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

type Client interface {
	// Get a string identifier for this client for logging purposes.
	// This does not have to be unique.
	Reference() string
	Id() uint16
	Host() string
	SetId(newId uint16)
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
}

type ClientState struct {
	Server *servers.GameServer
	Mutex  sync.Mutex
}

type ClientBundle struct {
	Client Client
	State  *ClientState
}

type ClientManager struct {
	Clients    map[Client]*ClientState
	Mutex      sync.Mutex
	newClients chan ClientBundle
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Clients:    make(map[Client]*ClientState),
		newClients: make(chan ClientBundle, 16),
	}
}

func (c *ClientManager) newClientID() (uint16, error) {
	for attempts := 0; attempts < math.MaxUint16; attempts++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		truncated := uint16(number.Uint64())

		taken := false
		for client, _ := range c.Clients {
			if client.Id() == truncated {
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

func (c *ClientManager) AddClient(client Client) error {
	id, err := c.newClientID()
	if err != nil {
		return err
	}

	client.SetId(id)

	state := &ClientState{}

	c.Mutex.Lock()
	c.Clients[client] = state
	c.Mutex.Unlock()

	c.newClients <- ClientBundle{
		Client: client,
		State:  state,
	}

	return nil
}

func (c *ClientManager) RemoveClient(client Client) {
	c.Mutex.Lock()
	delete(c.Clients, client)
	c.Mutex.Unlock()
}

func (c *ClientManager) ReceiveClients() <-chan ClientBundle {
	return c.newClients
}

func (c *ClientManager) FindClient(id uint16) Client {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	for client, _ := range c.Clients {
		if client.Id() != uint16(id) {
			continue
		}

		return client
	}

	return nil
}
