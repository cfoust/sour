package ingress

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/watcher"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

const (
	// Server -> client
	InfoOp int = iota
	ServerConnectedOp
	ServerDisconnectedOp
	ServerResponseOp
	AuthSucceededOp
	AuthFailedOp
	ChatOp
	// Client -> server
	ConnectOp
	DisconnectOp
	CommandOp
	DiscordCodeOp
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

type AuthSucceededMessage struct {
	Op         int // AuthSucceededOp
	Code       string
	User       auth.DiscordUser
	PrivateKey string
}

type DiscordCodeMessage struct {
	Op   int // DiscordCodeOp or AuthFailedOp
	Code string
}

type ChatMessage struct {
	Op      int // ChatOp
	Message string
}

type ResponseMessage struct {
	Op       int // ServerResponseOp
	Success  bool
	Response string
	// Uniquely identifies the command so we can send a response
	Id int
}

type ServerConnectedMessage struct {
	Op     int // ServerConnectedOp
	Server string
	// Whether to put the server in the URL or not
	Internal bool
	// Whether this is the user's server
	Owned bool
}

type ServerDisconnectedMessage struct {
	Op      int // ServerDisconnectedOp
	Message string
	Reason  int
}

type GenericMessage struct {
	Op int
}

type WSClient struct {
	host           string
	status         NetworkStatus
	toClient       chan game.GamePacket
	toServer       chan game.GamePacket
	commands       chan ClusterCommand
	authentication chan *auth.AuthUser
	disconnect     chan bool
	send           chan []byte
	closeSlow      func()

	context context.Context
	cancel  context.CancelFunc
}

func NewWSClient() *WSClient {
	return &WSClient{
		status:         NetworkStatusConnected,
		toClient:       make(chan game.GamePacket, CLIENT_MESSAGE_LIMIT),
		toServer:       make(chan game.GamePacket, CLIENT_MESSAGE_LIMIT),
		commands:       make(chan ClusterCommand, CLIENT_MESSAGE_LIMIT),
		authentication: make(chan *auth.AuthUser),
		send:           make(chan []byte, CLIENT_MESSAGE_LIMIT),
		disconnect:     make(chan bool, 1),
	}
}

func (c *WSClient) Host() string {
	return c.host
}

func (c *WSClient) SessionContext() context.Context {
	return c.context
}

func (c *WSClient) NetworkStatus() NetworkStatus {
	return c.status
}

func (c *WSClient) Destroy() {
	c.status = NetworkStatusDisconnected
}

func (c *WSClient) Connect(name string, isHidden bool, shouldCopy bool) {
	packet := ServerConnectedMessage{
		Op:       ServerConnectedOp,
		Server:   name,
		Internal: isHidden,
		Owned:    shouldCopy,
	}

	bytes, _ := cbor.Marshal(packet)
	c.send <- bytes
}

func (c *WSClient) Type() ClientType {
	return ClientTypeWS
}

func (c *WSClient) Send(packet game.GamePacket) <-chan bool {
	done := make(chan bool, 1)
	c.toClient <- packet
	// We don't get ACKs over WS (for now, this is unnecessary)
	done <- true
	return done
}

func (c *WSClient) ReceivePackets() <-chan game.GamePacket {
	return c.toServer
}

func (c *WSClient) ReceiveCommands() <-chan ClusterCommand {
	return c.commands
}

func (c *WSClient) ReceiveAuthentication() <-chan *auth.AuthUser {
	return c.authentication
}

func (c *WSClient) ReceiveDisconnect() <-chan bool {
	return c.disconnect
}

func (c *WSClient) SendGlobalChat(message string) {
	chat := ChatMessage{
		Op:      ChatOp,
		Message: message,
	}
	bytes, _ := cbor.Marshal(chat)
	c.send <- bytes
}

func (c *WSClient) Disconnect(reason int, message string) {
	wsPacket := ServerDisconnectedMessage{
		Op:      ServerDisconnectedOp,
		Message: message,
		Reason:  reason,
	}

	bytes, _ := cbor.Marshal(wsPacket)
	c.send <- bytes
}

type WSIngress struct {
	newClients    chan Connection
	clients       map[*WSClient]struct{}
	mutex         sync.Mutex
	serverWatcher *watcher.Watcher
	httpServer    *http.Server
	auth          *auth.DiscordService
}

func NewWSIngress(newClients chan Connection, auth *auth.DiscordService) *WSIngress {
	return &WSIngress{
		newClients:    newClients,
		clients:       make(map[*WSClient]struct{}),
		serverWatcher: watcher.NewWatcher(),
		auth:          auth,
	}
}

func WriteTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageBinary, msg)
}

func (server *WSIngress) AddClient(s *WSClient) {
	server.mutex.Lock()
	server.clients[s] = struct{}{}
	server.mutex.Unlock()
}

func (server *WSIngress) RemoveClient(client *WSClient) {
	server.mutex.Lock()
	delete(server.clients, client)
	server.mutex.Unlock()
}

func (server *WSIngress) HandleLogin(ctx context.Context, client *WSClient, code string) {
	if server.auth == nil || code == "" {
		client.authentication <- nil
		return
	}

	user, err := server.auth.AuthenticateCode(ctx, code)

	if err != nil {
		log.Error().Err(err).Msg("user failed to log in")
		response := DiscordCodeMessage{
			Op:   AuthFailedOp,
			Code: code,
		}
		bytes, _ := cbor.Marshal(response)
		client.send <- bytes
		return
	}

	response := AuthSucceededMessage{
		Op:         AuthSucceededOp,
		Code:       code,
		User:       user.Discord,
		PrivateKey: user.Keys.Private,
	}
	bytes, _ := cbor.Marshal(response)
	client.send <- bytes
	client.authentication <- user
}

func (server *WSIngress) HandleClient(ctx context.Context, c *websocket.Conn, host string) error {
	client := NewWSClient()

	clientCtx, cancel := context.WithCancel(ctx)

	client.context = clientCtx
	client.cancel = cancel

	server.newClients <- client

	defer cancel()

	server.AddClient(client)
	defer server.RemoveClient(client)

	client.host = host
	client.closeSlow = func() {
		c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}

	logger := log.With().Str("host", host).Logger()

	logger.Info().Msg("client joined")

	go func() {
		for {
			select {
			case packet := <-client.toClient:
				wsPacket := PacketMessage{
					Op:      PacketOp,
					Channel: int(packet.Channel),
					Data:    packet.Data,
					Length:  len(packet.Data),
				}

				bytes, _ := cbor.Marshal(wsPacket)
				client.send <- bytes
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	// Write the first broadcast on connect so they don't have to wait 5s
	broadcast, err := server.BuildBroadcast()
	if err != nil {
		logger.Error().Err(err).Msg("could not build broadcast")
		return err
	}
	client.send <- broadcast

	receive := make(chan []byte)

	defer func() {
		client.disconnect <- true
	}()

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			typ, message, _ := c.Read(ctx)
			if typ != websocket.MessageBinary {
				continue
			}
			receive <- message
		}
	}()

	for {
		select {
		case msg := <-receive:
			var connectMessage ConnectMessage
			if err := cbor.Unmarshal(msg, &connectMessage); err == nil &&
				connectMessage.Op == ConnectOp {
				target := connectMessage.Target

				logger.Info().Str("target", target).
					Msg("client attempting connect")

				client.commands <- ClusterCommand{
					Command: fmt.Sprintf("join %s", target),
					// We don't care here
					Response: make(chan CommandResult),
				}
			}

			var packetMessage PacketMessage
			if err := cbor.Unmarshal(msg, &packetMessage); err == nil &&
				packetMessage.Op == PacketOp {

				client.toServer <- game.GamePacket{
					Channel: uint8(packetMessage.Channel),
					Data:    packetMessage.Data,
				}
			}

			var discordCode DiscordCodeMessage
			if err := cbor.Unmarshal(msg, &discordCode); err == nil &&
				discordCode.Op == DiscordCodeOp {
				server.HandleLogin(ctx, client, discordCode.Code)
			}

			var commandMessage CommandMessage
			if err := cbor.Unmarshal(msg, &commandMessage); err == nil &&
				commandMessage.Op == CommandOp {

				resultChannel := make(chan CommandResult)
				client.commands <- ClusterCommand{
					Command:  commandMessage.Command,
					Response: resultChannel,
				}

				ctx, cancel := context.WithTimeout(ctx, time.Second*10)

				// Go run a command, but don't block
				go func() {
					select {
					case result := <-resultChannel:
						response := result.Response
						err := result.Err

						packet := ResponseMessage{
							Op: ServerResponseOp,
							Id: commandMessage.Id,
						}

						if err == nil {
							packet.Success = true
							packet.Response = response
						} else {
							packet.Success = false
							packet.Response = err.Error()
						}

						bytes, _ := cbor.Marshal(packet)
						client.send <- bytes
						cancel()
					case <-ctx.Done():
						// The command timed out
						return
					}
				}()
			}

			var generic GenericMessage
			err := cbor.Unmarshal(msg, &generic)
			if err == nil && packetMessage.Op == DisconnectOp {
				client.disconnect <- true
			}
		case msg := <-client.send:
			err := WriteTimeout(ctx, time.Second*5, c, msg)
			if err != nil {
				logger.Error().Msg("client missed write timeout; disconnecting")
				return err
			}
		case <-ctx.Done():
			logger.Info().Msg("client left")
			return ctx.Err()
		}
	}
}

func (server *WSIngress) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Ignore the request, this sometimes happens
	if r.URL.Path != "/" {
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})

	if err != nil {
		log.Error().Err(err).Msg("error accepting client connection")
		return
	}

	defer c.Close(websocket.StatusInternalError, "operational fault during relay")

	// We use nginx for ingress everywhere, so check this first
	hostname := r.RemoteAddr

	original, ok := r.Header["X-Forwarded-For"]
	if ok {
		hostname = original[0]
	}

	err = server.HandleClient(r.Context(), c, hostname)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		log.Error().Err(err).Msg("failed to close client port")
		return
	}
}

func (server *WSIngress) Broadcast(msg []byte) {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	for client := range server.clients {
		select {
		case client.send <- msg:
		default:
			go client.closeSlow()
		}
	}
}

func (server *WSIngress) BuildBroadcast() ([]byte, error) {
	servers := server.serverWatcher.Get()

	masterServers := make([]ServerInfo, len(servers))
	index := 0
	for key, server := range servers {
		masterServers[index] = ServerInfo{
			Host:   key.Host,
			Port:   key.Port,
			Info:   server.Info,
			Length: server.Length,
		}
		index++
	}

	infoMessage := InfoMessage{
		Op:     InfoOp,
		Master: masterServers,
	}

	bytes, err := cbor.Marshal(infoMessage)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (server *WSIngress) StartWatcher(ctx context.Context) {
	go server.serverWatcher.Watch()

	broadcastTicker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-broadcastTicker.C:
				bytes, err := server.BuildBroadcast()

				if err != nil {
					log.Error().Err(err).Msg("could not build broadcast")
					return
				}

				server.Broadcast(bytes)
			}
		}
	}()
}

func (server *WSIngress) Shutdown(ctx context.Context) {
	server.httpServer.Shutdown(ctx)
}
