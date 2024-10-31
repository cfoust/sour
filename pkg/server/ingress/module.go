package ingress

import (
	"github.com/cfoust/sour/pkg/game/io"
	"github.com/cfoust/sour/pkg/utils"
)

// A unique identifier for this client for the lifetime of their session.
type ClientID uint32

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
	Err error
}

type ClusterCommand struct {
	Command  string
	Response chan CommandResult
}

type Connection interface {
	Session() *utils.Session

	// Lasts for the duration of the client's connection to its ingress.
	NetworkStatus() NetworkStatus
	Host() string
	Type() ClientType
	DeviceType() string
	// Tell the client that we've connected
	Connect(name string, isHidden bool, shouldCopy bool)
	// Messages going to the client
	Send(packet io.RawPacket) <-chan error
	// Messages going to the server
	ReceivePackets() <-chan io.RawPacket
	// Clients can issue commands out-of-band
	// Commands sent in ordinary game packets are interpreted anyway
	ReceiveCommands() <-chan ClusterCommand
	// When the client disconnects on its own
	ReceiveDisconnect() <-chan bool
	// Forcibly disconnect this client
	Disconnect(reason int, message string)
	Destroy()
}

var _ Connection = (*WSClient)(nil)
var _ Connection = (*ENetClient)(nil)
