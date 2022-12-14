package clients

import (
	"sync"
)

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

type GamePacket struct {
	Channel uint8
	Data    []byte
}

type ClusterCommand struct {
	Command  string
	Response chan string
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

func (c *ClientManager) AddClient(client Client) {
	c.mutex.Lock()
	c.Clients[client] = struct{}{}
	c.mutex.Unlock()
}

func (c *ClientManager) RemoveClient(client Client) {
	c.mutex.Lock()
	delete(c.Clients, client)
	c.mutex.Unlock()
}
