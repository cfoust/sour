package clients

import (
	"crypto/rand"
	"errors"
	"fmt"
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

type Client interface {
	// Get a string identifier for this client for logging purposes.
	// This does not have to be unique.
	Reference() string
	NetworkStatus() ClientNetworkStatus
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
	Destroy()
}

type ClientState struct {
	Server *servers.GameServer
	Mutex  sync.Mutex
	Status ClientStatus
}

func (s *ClientState) GetStatus() ClientStatus {
	s.Mutex.Lock()
	status := s.Status
	s.Mutex.Unlock()

	return status
}

type ClientBundle struct {
	Client Client
	State  *ClientState
}

type ClientManager struct {
	state      map[Client]*ClientState
	Mutex      sync.Mutex
	newClients chan ClientBundle
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		state:      make(map[Client]*ClientState),
		newClients: make(chan ClientBundle, 16),
	}
}

func (c *ClientManager) newClientID() (uint16, error) {
	for attempts := 0; attempts < math.MaxUint16; attempts++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		truncated := uint16(number.Uint64())

		taken := false
		for client, _ := range c.state {
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

func (c *ClientManager) GetState(client Client) *ClientState {
	c.Mutex.Lock()
	state, ok := c.state[client]
	c.Mutex.Unlock()

	if !ok {
		return nil
	}
	return state
}

func (c *ClientManager) ConnectClient(server *servers.GameServer, client Client) error {
	state := c.GetState(client)
	if state == nil {
		return fmt.Errorf("could not find state for client")
	}

	state.Mutex.Lock()
	if state.Server != nil {
		state.Server.SendDisconnect(client.Id())
	}
	state.Server = server
	state.Status = ClientStatusConnecting
	state.Mutex.Unlock()

	server.SendConnect(client.Id())
	client.Connect()

	return nil
}

func (c *ClientManager) AddClient(client Client) error {
	id, err := c.newClientID()
	if err != nil {
		return err
	}

	client.SetId(id)

	state := &ClientState{}

	c.Mutex.Lock()
	c.state[client] = state
	c.Mutex.Unlock()

	c.newClients <- ClientBundle{
		Client: client,
		State:  state,
	}

	return nil
}

func SendServerMessage(client Client, message string) {
	packet := game.Packet{}
	packet.PutInt(int32(game.N_SERVMSG))
	message = fmt.Sprintf("%s %s", game.Yellow("sour"), message)
	packet.PutString(message)
	client.Send(game.GamePacket{
		Channel: 1,
		Data:    packet,
	})
}

func (c *ClientManager) RemoveClient(client Client) {
	c.Mutex.Lock()
	delete(c.state, client)
	c.Mutex.Unlock()
}

func (c *ClientManager) ReceiveClients() <-chan ClientBundle {
	return c.newClients
}

func (c *ClientManager) FindClient(id uint16) Client {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	for client, _ := range c.state {
		if client.Id() != uint16(id) {
			continue
		}

		return client
	}

	return nil
}
