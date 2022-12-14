package ingress

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/enet"
	"github.com/cfoust/sour/pkg/protocol"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/watcher"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type ENetClient struct {
	id         uint16
	peer       *enet.Peer
	cancel     context.CancelFunc
	toClient   chan clients.GamePacket
	toServer   chan clients.GamePacket
	commands   chan clients.ClusterCommand
	disconnect chan bool
}

func NewENetClient(cancel context.CancelFunc) *ENetClient {
	return &ENetClient{
		cancel:     cancel,
		toClient:   make(chan clients.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		toServer:   make(chan clients.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		commands:   make(chan clients.ClusterCommand, clients.CLIENT_MESSAGE_LIMIT),
		disconnect: make(chan bool, 1),
	}
}

func (c *ENetClient) Id() uint16 {
	return c.id
}

func (c *ENetClient) Type() clients.ClientType {
	return clients.ClientTypeENet
}

func (c *ENetClient) Reference() string {
	return fmt.Sprintf("enet:%d", c.id)
}

func (c *ENetClient) SetId(id uint16) {
	c.id = id
}

func (c *ENetClient) Send(packet clients.GamePacket) {
	c.toClient <- packet
}

func (c *ENetClient) ReceivePackets() <-chan clients.GamePacket {
	return c.toServer
}

func (c *ENetClient) ReceiveCommands() <-chan clients.ClusterCommand {
	return c.commands
}

func (c *ENetClient) ReceiveDisconnect() <-chan bool {
	return c.disconnect
}

func (c *ENetClient) Poll(ctx context.Context) {
	for {
		select {
		case packet := <-c.toClient:
			c.peer.Send(packet.Channel, packet.Data)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (c *ENetClient) Disconnect() {
	c.cancel()
}

type ENetIngress struct {
	manager *clients.ClientManager
	clients map[*ENetClient]struct{}
	host    *enet.Host
	mutex   sync.Mutex
}

func NewENetIngress(manager *clients.ClientManager) *ENetIngress {
	return &ENetIngress{
		manager: manager,
		clients: make(map[*ENetClient]struct{}),
	}
}

func (server *ENetIngress) Serve(port int) error {
	host, err := enet.NewHost("", port)
	if err != nil {
		return err
	}
	server.host = host
	return nil
}

func (server *ENetIngress) FindClientForPeer(peer *enet.Peer) *ENetClient {
	var target *ENetClient = nil

	server.mutex.Lock()
	for client, _ := range server.clients {
		if client.peer == nil || peer.CPeer != client.peer.CPeer {
			continue
		}

		target = client
		break
	}
	server.mutex.Unlock()

	return target
}

func (server *ENetIngress) AddClient(s *ENetClient) {
	server.mutex.Lock()
	server.clients[s] = struct{}{}
	server.mutex.Unlock()
}

func (server *ENetIngress) RemoveClient(client *ENetClient) {
	server.mutex.Lock()
	delete(server.clients, client)
	server.mutex.Unlock()
}

func (server *ENetIngress) Poll(ctx context.Context) {
	events := server.host.Service()

	for {
		select {
		case event := <-events:
			switch event.Type {
			case enet.EventTypeConnect:
				ctx, cancel := context.WithCancel(ctx)

				client := NewENetClient(cancel)
				client.peer = event.Peer

				err := server.manager.AddClient(client)
				if err != nil {
					log.Error().Err(err).Msg("failed to accept enet client")
				}

				server.AddClient(client)

				logger := log.With().Uint16("clientId", client.id).Logger()
				logger.Info().Msg("client joined (desktop)")

				go client.Poll(ctx)

				// TODO
				//client.server = server.manager.Servers[0]
				//client.server.SendConnect(client.id)
				break

			case enet.EventTypeReceive:
				target := server.FindClientForPeer(event.Peer)

				if target == nil {
					continue
				}

				target.toServer <- clients.GamePacket{
					Channel: event.ChannelID,
					Data:    event.Packet.Data,
				}

				break
			case enet.EventTypeDisconnect:
				target := server.FindClientForPeer(event.Peer)

				if target == nil {
					continue
				}

				server.RemoveClient(target)
				target.disconnect <- true

				server.manager.RemoveClient(target)
				break
			}
		case <-ctx.Done():
			return
		}
	}

}

func (server *ENetIngress) Shutdown() {
	server.host.Shutdown()
}

type WSClient struct {
	id         uint16
	host       string
	toClient   chan clients.GamePacket
	toServer   chan clients.GamePacket
	commands   chan clients.ClusterCommand
	disconnect chan bool
	send       chan []byte
	closeSlow  func()
}

func NewWSClient() *WSClient {
	return &WSClient{
		toClient:   make(chan clients.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		toServer:   make(chan clients.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		commands:   make(chan clients.ClusterCommand, clients.CLIENT_MESSAGE_LIMIT),
		send:       make(chan []byte, clients.CLIENT_MESSAGE_LIMIT),
		disconnect: make(chan bool, 1),
	}
}

func (c *WSClient) Id() uint16 {
	return c.id
}

func (c *WSClient) Type() clients.ClientType {
	return clients.ClientTypeENet
}

func (c *WSClient) Reference() string {
	return fmt.Sprintf("ws:%d", c.id)
}

func (c *WSClient) SetId(id uint16) {
	c.id = id
}

func (c *WSClient) Send(packet clients.GamePacket) {
	c.toClient <- packet
}

func (c *WSClient) ReceivePackets() <-chan clients.GamePacket {
	return c.toServer
}

func (c *WSClient) ReceiveCommands() <-chan clients.ClusterCommand {
	return c.commands
}

func (c *WSClient) ReceiveDisconnect() <-chan bool {
	return c.disconnect
}

func (c *WSClient) Disconnect() {
}

type WSIngress struct {
	manager       *clients.ClientManager
	clients       map[*WSClient]struct{}
	mutex         sync.Mutex
	serverWatcher *watcher.Watcher
	httpServer    *http.Server
}

func NewWSIngress(manager *clients.ClientManager) *WSIngress {
	return &WSIngress{
		manager:       manager,
		clients:       make(map[*WSClient]struct{}),
		serverWatcher: watcher.NewWatcher(),
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

func (server *WSIngress) HandleClient(ctx context.Context, c *websocket.Conn, host string) error {
	client := NewWSClient()
	err := server.manager.AddClient(client)
	if err != nil {
		log.Error().Err(err).Msg("failed to accept ws client")
	}

	server.AddClient(client)
	defer server.RemoveClient(client)

	client.host = host
	client.closeSlow = func() {
		c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}

	logger := log.With().Uint16("clientId", client.id).Str("host", host).Logger()

	logger.Info().Msg("client joined")

	go func() {
		for {
			select {
			case packet := <-client.toClient:
				wsPacket := protocol.PacketMessage{
					Op:      protocol.PacketOp,
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

	defer server.manager.RemoveClient(client)

	// Write the first broadcast on connect so they don't have to wait 5s
	broadcast, err := server.BuildBroadcast()
	if err != nil {
		logger.Error().Err(err).Msg("could not build broadcast")
		return err
	}
	client.send <- broadcast

	receive := make(chan []byte)

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
			var connectMessage protocol.ConnectMessage
			if err := cbor.Unmarshal(msg, &connectMessage); err == nil &&
				connectMessage.Op == protocol.ConnectOp {
				target := connectMessage.Target

				logger.Info().Str("target", target).
					Msg("client attempting connect")

				client.commands <- clients.ClusterCommand{
					Command: fmt.Sprintf("join %s", target),
					// We don't care here
					Response: make(chan clients.CommandResult),
				}

				//if client.server != nil && client.server.IsReference(target) {
				//break
				//}

				//for _, gameServer := range server.manager.Servers {
				//if !gameServer.IsReference(target) || gameServer.Status != servers.ServerOK {
				//continue
				//}

				//client.server = gameServer

				//logger.Info().Str("server", gameServer.Reference()).
				//Msg("client connecting to server")

				//gameServer.SendConnect(client.id)

				//packet := protocol.GenericMessage{
				//Op: protocol.ServerConnectedOp,
				//}

				//bytes, _ := cbor.Marshal(packet)
				//client.send <- bytes

				//break
				//}
			}

			var packetMessage protocol.PacketMessage
			if err := cbor.Unmarshal(msg, &packetMessage); err == nil &&
				packetMessage.Op == protocol.PacketOp {

				client.toServer <- clients.GamePacket{
					Channel: uint8(packetMessage.Channel),
					Data:    packetMessage.Data,
				}
			}

			var commandMessage protocol.CommandMessage
			if err := cbor.Unmarshal(msg, &commandMessage); err == nil &&
				commandMessage.Op == protocol.CommandOp {

				resultChannel := make(chan clients.CommandResult)
				client.commands <- clients.ClusterCommand{
					Command:  commandMessage.Command,
					Response: resultChannel,
				}

				ctx, cancel := context.WithTimeout(ctx, time.Second*10)

				// Go run a command, but don't block
				go func() {
					select {
					case result := <-resultChannel:
						cancel()
						response := result.Response
						err := result.Err

						packet := protocol.ResponseMessage{
							Op: protocol.ServerResponseOp,
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
					case <-ctx.Done():
						// The command timed out
						return
					}
				}()
			}

			var generic protocol.GenericMessage
			err := cbor.Unmarshal(msg, &generic)
			if err == nil && packetMessage.Op == protocol.DisconnectOp {
				client.disconnect <- true
			}
		case msg := <-client.send:
			log.Printf("sending msg to client")
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

	masterServers := make([]protocol.ServerInfo, len(servers))
	index := 0
	for key, server := range servers {
		masterServers[index] = protocol.ServerInfo{
			Host:   key.Host,
			Port:   key.Port,
			Info:   server.Info,
			Length: server.Length,
		}
		index++
	}

	infoMessage := protocol.InfoMessage{
		Op:     protocol.InfoOp,
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

func (server *WSIngress) Serve(ctx context.Context, port int) error {
	listen, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Error().Err(err).Msg("failed to bind WebSocket port")
		return err
	}

	log.Printf("listening on http://%v", listen.Addr())

	httpServer := &http.Server{
		Handler: server,
	}

	server.httpServer = httpServer

	go server.StartWatcher(ctx)

	return httpServer.Serve(listen)
}

func (server *WSIngress) Shutdown(ctx context.Context) {
	server.httpServer.Shutdown(ctx)
}
