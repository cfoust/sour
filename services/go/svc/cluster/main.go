package main

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/enet"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/manager"
	"github.com/cfoust/sour/pkg/protocol"
	"github.com/cfoust/sour/pkg/watcher"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type GamePacket struct {
	Channel uint8
	Data    []byte
}

type Client struct {
	id   uint16
	host string

	peer      *enet.Peer
	server    *manager.GameServer
	send      chan []byte
	sendPacket chan GamePacket
	closeSlow func()
}

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

const (
	CREATE_SERVER_COOLDOWN = time.Duration(10 * time.Second)
)

type Cluster struct {
	clientMutex sync.Mutex
	clients     map[*Client]struct{}

	createMutex sync.Mutex
	// host -> time a client from that host last created a server. We
	// REALLY don't want clients to be able to DDOS us
	lastCreate map[string]time.Time
	// host -> the server created by that host
	// each host can only have one server at once
	hostServers map[string]*manager.GameServer

	manager       *manager.Manager
	serverCtx     context.Context
	serverMessage chan []byte
	serverWatcher *watcher.Watcher
}

func NewCluster(ctx context.Context, serverPath string) *Cluster {
	server := &Cluster{
		serverCtx:     ctx,
		hostServers:   make(map[string]*manager.GameServer),
		lastCreate:    make(map[string]time.Time),
		clients:       make(map[*Client]struct{}),
		serverWatcher: watcher.NewWatcher(),
		serverMessage: make(chan []byte, 1),
		manager: manager.NewManager(
			serverPath,
			50000,
			51000,
		),
	}

	return server
}

func (server *Cluster) NewClientID() (uint16, error) {
	for attempts := 0; attempts < math.MaxUint16; attempts++ {
		number, _ := rand.Int(rand.Reader, big.NewInt(math.MaxUint16))
		truncated := uint16(number.Uint64())

		taken := false
		for client, _ := range server.clients {
			if client.id == truncated {
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

func (server *Cluster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	err = server.Subscribe(r.Context(), c, hostname)
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

func (server *Cluster) StartPresetServer(ctx context.Context) (*manager.GameServer, error) {
	// Default in development
	configPath := "../server/config/server-init.cfg"

	if envPath, ok := os.LookupEnv("QSERV_LOBBY_CONFIG"); ok {
		configPath = envPath
	}

	gameServer, err := server.manager.NewServer(ctx, configPath)

	return gameServer, err
}

func (server *Cluster) StartServers(ctx context.Context) {
	gameServer, err := server.StartPresetServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	gameServer.Alias = "lobby"

	go gameServer.Start(ctx, server.serverMessage)
	go server.manager.PruneServers(ctx)
}

func (server *Cluster) StartWatcher(ctx context.Context) {
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

func (server *Cluster) PollMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-server.serverMessage:
			p := game.Packet(msg)

			for len(p) > 0 {
				numBytes, ok := p.GetUint()
				if !ok {
					break
				}
				id, ok := p.GetUint()
				if !ok {
					break
				}
				chan_, ok := p.GetUint()
				if !ok {
					break
				}

				data := p[:numBytes]
				p = p[len(data):]

				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.id != uint16(id) {
						continue
					}

					packet := GamePacket{
						Channel: uint8(chan_),
						Data: data,
					}

					client.sendPacket <- packet

					break
				}
				server.clientMutex.Unlock()
			}
		}
	}
}

func (server *Cluster) MoveClient(ctx context.Context, client *Client, targetServer *manager.GameServer) error {
	if targetServer.Status != manager.ServerOK {
		return errors.New("Server is not available")
	}

	if targetServer == client.server {
		return nil
	}

	log.Info().Msgf("swapping from %s to %s", client.server.Id, targetServer.Id)

	// We have 'em!
	client.server.SendDisconnect(client.id)
	targetServer.SendConnect(client.id)
	client.server = targetServer

	return nil
}

func (server *Cluster) RunCommand(ctx context.Context, client *Client, command string) (string, error) {
	logger := log.With().Uint16("clientId", client.id).Logger()
	logger.Info().Msgf("running sour command '%s'", command)

	args := strings.Split(command, " ")

	if len(args) == 0 {
		return "", errors.New("invalid command")
	}

	switch args[0] {
	case "creategame":

		server.createMutex.Lock()
		defer server.createMutex.Unlock()

		lastCreate, hasLastCreate := server.lastCreate[client.host]
		if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
			return "", errors.New("too soon since last server create")
		}

		existingServer, hasExistingServer := server.hostServers[client.host]
		if hasExistingServer {
			server.manager.RemoveServer(existingServer)
		}

		logger.Info().Msg("starting server")

		gameServer, err := server.StartPresetServer(server.serverCtx)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create server")
			return "", errors.New("failed to create server")
		}

		logger = logger.With().Str("server", gameServer.Reference()).Logger()

		go gameServer.Start(server.serverCtx, server.serverMessage)

		tick := time.NewTicker(250 * time.Millisecond)
		for {
			status := gameServer.GetStatus()
			if status == manager.ServerOK {
				logger.Info().Msg("server ok")
				break
			}

			select {
			case <-ctx.Done():
				return "", errors.New("server start timed out")
			case <-tick.C:
				continue
			}
		}

		server.lastCreate[client.host] = time.Now()
		server.hostServers[client.host] = gameServer

		return gameServer.Id, nil
	}

	return "", nil
}

func (server *Cluster) EmptyClient() (*Client, error) {
	server.clientMutex.Lock()
	defer server.clientMutex.Unlock()

	id, err := server.NewClientID()

	if err != nil {
		return nil, err
	}

	client := &Client{
		id: id,
	}

	return client, nil
}

func (server *Cluster) Subscribe(ctx context.Context, c *websocket.Conn, host string) error {
	client, err := server.EmptyClient()

	if err != nil {
		return err
	}

	client.host = host
	client.send = make(chan []byte, CLIENT_MESSAGE_LIMIT)
	client.closeSlow = func() {
		c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
	}

	logger := log.With().Uint16("clientId", client.id).Str("host", host).Logger()

	logger.Info().Msg("client joined")

	gamePacketChannel := make(chan GamePacket, CLIENT_MESSAGE_LIMIT)
	client.sendPacket = gamePacketChannel

	go func() {
		for {
			select {
			case packet := <-gamePacketChannel:
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


	server.AddClient(client)
	defer server.RemoveClient(client)

	// Write the first broadcast on connect so they don't have to wait 5s
	broadcast, err := server.BuildBroadcast()
	if err != nil {
		logger.Error().Err(err).Msg("could not build broadcast")
		return err
	}
	err = WriteTimeout(ctx, time.Second*5, c, broadcast)

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

				if client.server != nil && client.server.IsReference(target) {
					break
				}

				for _, gameServer := range server.manager.Servers {
					if !gameServer.IsReference(target) || gameServer.Status != manager.ServerOK {
						continue
					}

					client.server = gameServer

					logger.Info().Str("server", gameServer.Reference()).
						Msg("client connecting to server")

					gameServer.SendConnect(client.id)

					packet := protocol.GenericMessage{
						Op: protocol.ServerConnectedOp,
					}

					bytes, _ := cbor.Marshal(packet)
					client.send <- bytes

					break
				}
			}

			var packetMessage protocol.PacketMessage
			if err := cbor.Unmarshal(msg, &packetMessage); err == nil &&
				packetMessage.Op == protocol.PacketOp {
				target := client.server
				if target == nil {
					break
				}

				target.SendData(
					client.id,
					uint32(packetMessage.Channel),
					packetMessage.Data,
				)
			}

			var commandMessage protocol.CommandMessage
			if err := cbor.Unmarshal(msg, &commandMessage); err == nil &&
				commandMessage.Op == protocol.CommandOp {

				type CommandResult struct {
					err      error
					response string
				}

				resultChannel := make(chan CommandResult)

				ctx, cancel := context.WithTimeout(ctx, time.Second*10)

				go func() {
					response, err := server.RunCommand(ctx, client, commandMessage.Command)
					resultChannel <- CommandResult{
						err:      err,
						response: response,
					}
				}()

				// Go run a command, but don't block
				go func() {
					select {
					case result := <-resultChannel:
						cancel()
						response := result.response
						err := result.err

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
				target := client.server
				if target == nil {
					break
				}

				logger.Info().Str("server", client.server.Reference()).Msg("client disconnected from server")
				client.server = nil
				target.SendDisconnect(client.id)
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

func (server *Cluster) Broadcast(msg []byte) {
	server.clientMutex.Lock()
	defer server.clientMutex.Unlock()

	for client := range server.clients {
		if client.peer != nil {
			continue
		}

		select {
		case client.send <- msg:
		default:
			go client.closeSlow()
		}
	}
}

func (server *Cluster) BuildBroadcast() ([]byte, error) {
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

	clusterServers := make([]string, 0)
	for _, clusterServer := range server.manager.Servers {
		clusterServers = append(clusterServers, clusterServer.Id)
	}

	infoMessage := protocol.InfoMessage{
		Op:      protocol.InfoOp,
		Master:  masterServers,
		Cluster: clusterServers,
	}

	bytes, err := cbor.Marshal(infoMessage)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (server *Cluster) AddClient(s *Client) {
	server.clientMutex.Lock()
	server.clients[s] = struct{}{}
	server.clientMutex.Unlock()
}

func (server *Cluster) RemoveClient(client *Client) {
	if client.server != nil {
		client.server.SendDisconnect(client.id)
	}

	server.clientMutex.Lock()
	delete(server.clients, client)
	server.clientMutex.Unlock()
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
}

func WriteTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageBinary, msg)
}

func (server *Cluster) EnetSend(ctx context.Context) <-chan []byte {
	sendChannel := make(chan []byte, CLIENT_MESSAGE_LIMIT)

	go func() {
		for {
			select {
			case msg := <-sendChannel:
				log.Print(msg)
				continue
			case <-ctx.Done():
				return
			}
		}
	}()

	return sendChannel
}

func (server *Cluster) PollEnet(ctx context.Context, host *enet.Host) {
	events := host.Service()

	for {
		select {
		case event := <-events:
			switch event.Type {
			case enet.EventTypeConnect:
				client, err := server.EmptyClient()
				if err != nil {
					log.Error().Err(err).Msg("failed to accept enet client")
				}

				client.peer = event.Peer

				logger := log.With().Uint16("clientId", client.id).Logger()
				logger.Info().Msg("client joined (desktop)")

				gamePacketChannel := make(chan GamePacket, CLIENT_MESSAGE_LIMIT)
				client.sendPacket = gamePacketChannel

				go func() {
					for {
						select {
						case packet := <-gamePacketChannel:
							client.peer.Send(packet.Channel, packet.Data)
							continue
						case <-ctx.Done():
							return
						}
					}
				}()

				client.server = server.manager.Servers[0]
				client.server.SendConnect(client.id)

				server.AddClient(client)
				break
			case enet.EventTypeReceive:
				peer := event.Peer

				var target *Client = nil
				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.peer == nil || peer.CPeer != client.peer.CPeer {
						continue
					}

					target = client
					break
				}
				server.clientMutex.Unlock()
				if target == nil || target.server == nil {
					break
				}

				target.server.SendData(
					target.id,
					uint32(event.ChannelID),
					event.Packet.Data,
				)

				break
			case enet.EventTypeDisconnect:
				peer := event.Peer

				var target *Client = nil
				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.peer == nil || peer.CPeer != client.peer.CPeer {
						continue
					}

					target = client
					break
				}
				server.clientMutex.Unlock()
				if target == nil {
					break
				}

				if target.server != nil {
					target.server.SendDisconnect(target.id)
				}

				server.RemoveClient(target)
				break
			}
		case <-ctx.Done():
			return
		}
	}

}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	serverPath := "../server/qserv"
	if envPath, ok := os.LookupEnv("QSERV_PATH"); ok {
		serverPath = envPath
	}

	l, err := net.Listen("tcp", "0.0.0.0:29999")
	if err != nil {
		log.Error().Err(err).Msg("failed to bind WebSocket port")
		return
	}

	log.Printf("listening on http://%v", l.Addr())

	enetHost, err := enet.NewHost("0.0.0.0", 28785)
	if err != nil {
		log.Error().Err(err).Msg("failed to bind ENet ingress")
	}

	log.Printf("listening on udp:%d", 28785)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := NewCluster(ctx, serverPath)

	httpServer := &http.Server{
		Handler: cluster,
	}

	go cluster.PollEnet(ctx, enetHost)

	go cluster.StartServers(ctx)
	go cluster.StartWatcher(ctx)
	go cluster.PollMessages(ctx)

	errc := make(chan error, 1)
	go func() {
		errc <- httpServer.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, os.Kill)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	httpServer.Shutdown(ctx)
	cluster.Shutdown()
	enetHost.Shutdown()
}
