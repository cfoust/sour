package ingress

import (
	"context"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
)

type ClientType uint8

const (
	ClientTypeWS = iota
	ClientTypeENet
)

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

// The status of the client's connection to the cluster.
type NetworkStatus uint8

const (
	NetworkStatusConnected = iota
	NetworkStatusDisconnected
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

type Connection interface {
	// Lasts for the duration of the client's connection to its ingress.
	SessionContext() context.Context
	NetworkStatus() NetworkStatus
	Host() string
	Type() ClientType
	// Tell the client that we've connected
	Connect(name string, isHidden bool, shouldCopy bool)
	// Messages going to the client
	Send(packet game.GamePacket) <-chan bool
	// Messages going to the server
	ReceivePackets() <-chan game.GamePacket
	// Clients can issue commands out-of-band
	// Commands sent in ordinary game packets are interpreted anyway
	ReceiveCommands() <-chan ClusterCommand
	// When the client disconnects on its own
	ReceiveDisconnect() <-chan bool
	// When the client authenticates
	ReceiveAuthentication() <-chan *auth.AuthUser
	// WS clients can put chat in the chat bar; ENet clients cannot
	SendGlobalChat(message string)
	// Forcibly disconnect this client
	Disconnect(reason int, message string)
	Destroy()
}
