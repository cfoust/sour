package protocol

const (
	// Server -> client
	InfoOp int = iota
	ServerConnectedOp
	ServerDisconnectedOp
	ServerResponseOp
	// Client -> server
	ConnectOp
	DisconnectOp
	CommandOp
	// server -> client OR client -> server
	PacketOp
)

type ServerInfo struct {
	Host   string
	Port   int
	Info   []byte
	Length int
}

// Contains information on servers this cluster contains and real ones from the
// master.
type InfoMessage struct {
	Op int // InfoOp
	// All of the servers from the master (real Sauerbraten servers.)
	Master []ServerInfo
	// All of the servers this cluster hosts.
	Cluster []string
}

// Contains a packet from the server a client is connected to.
type PacketMessage struct {
	Op      int // ServerPacketOp or ClientPacketOp
	Channel int
	Data    []byte
	Length  int
}

// Connect the client to a server
type ConnectMessage struct {
	Op int // ConnectOp
	// One of the servers hosted by the cluster
	Target string
}

// Issuing a cluster command on behalf of the user.
type CommandMessage struct {
	Op      int // CommandOp
	Command string
	// Uniquely identifies the command so we can send a response
	Id int
}

type ResponseMessage struct {
	Op       int // ServerResponseOp
	Success  bool
	Response string
	// Uniquely identifies the command so we can send a response
	Id int
}

type GenericMessage struct {
	Op int
}
