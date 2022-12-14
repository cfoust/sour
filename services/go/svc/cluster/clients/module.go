package clients

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sync"
)

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

type GamePacket struct {
	Channel uint8
	Data    []byte
}

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
	SetId(newId uint16)
	Type() ClientType
	// Messages going to the client
	Send(packet GamePacket)
	// Messages going to the server
	ReceivePackets() <-chan GamePacket
	// Clients can issue commands out-of-band
	// Commands sent in ordinary game packets are interpreted anyway
	ReceiveCommands() <-chan ClusterCommand
	// When the client disconnects on its own
	ReceiveDisconnect() <-chan bool
	// Forcibly disconnect this client
	Disconnect()
}

type ClientManager struct {
	Clients map[Client]struct{}
	mutex   sync.Mutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Clients: make(map[Client]struct{}),
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

	c.mutex.Lock()
	c.Clients[client] = struct{}{}
	c.mutex.Unlock()
	return nil
}

func (c *ClientManager) RemoveClient(client Client) {
	c.mutex.Lock()
	delete(c.Clients, client)
	c.mutex.Unlock()
}
