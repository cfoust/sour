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
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/manager"
	"github.com/cfoust/sour/pkg/protocol"
	"github.com/cfoust/sour/pkg/watcher"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"
)

type WSClient struct {
	id        uint16
	server    *manager.GameServer
	send      chan []byte
	closeSlow func()
}

const (
	CLIENT_MESSAGE_LIMIT int = 16
)

type Cluster struct {
	// logf controls where logs are sent.
	logf          func(f string, v ...interface{})
	clientMutex   sync.Mutex
	clients       map[*WSClient]struct{}
	serverWatcher *watcher.Watcher
	manager       *manager.Manager
	serverMessage chan []byte
}

func NewCluster() *Cluster {
	server := &Cluster{
		logf:          log.Printf,
		clients:       make(map[*WSClient]struct{}),
		serverWatcher: watcher.NewWatcher(),
		serverMessage: make(chan []byte, 1),
		manager: manager.NewManager(
			"../server/qserv",
			50000,
			51000,
		),
	}

	return server
}

func (server *Cluster) NewClientID() (uint16, error) {
	server.clientMutex.Lock()
	defer server.clientMutex.Unlock()

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
		server.logf("%v", err)
		return
	}

	defer c.Close(websocket.StatusInternalError, "operational fault during relay")

	err = server.Subscribe(r.Context(), c)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		server.logf("%v", err)
		return
	}
}

func (server *Cluster) StartServers(ctx context.Context) {
	gameServer, err := server.manager.NewServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create server")
	}

	gameServer.Alias = "lobby"

	go gameServer.Start(ctx, server.serverMessage)
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
					server.logf("%v", err)
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
			p := protocol.Packet(msg)

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

					packet := protocol.PacketMessage{
						Op:      protocol.PacketOp,
						Channel: int(chan_),
						Data:    data,
						Length:  len(data),
					}

					bytes, _ := cbor.Marshal(packet)
					client.send <- bytes

					break
				}
				server.clientMutex.Unlock()
			}
		}
	}
}

func (server *Cluster) MoveClient(ctx context.Context, client *WSClient, targetServer *manager.GameServer) error {
	if targetServer.Status != manager.ServerOK {
		return errors.New("Server is not available")
	}

	if targetServer == client.server {
		return nil
	}

	log.Info().Msgf("Swapping from %s to %s", client.server.Id, targetServer.Id)

	// We have 'em!
	client.server.SendDisconnect(client.id)
	targetServer.SendConnect(client.id)
	client.server = targetServer

	return nil
}

func (server *Cluster) Subscribe(ctx context.Context, c *websocket.Conn) error {
	client := &WSClient{
		send: make(chan []byte, CLIENT_MESSAGE_LIMIT),
		closeSlow: func() {
			c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

	id, err := server.NewClientID()
	if err != nil {
		return err
	}

	client.id = id

	log.Info().Uint16("id", id).Msg("client joined")

	server.AddClient(client)
	defer server.RemoveClient(client)

	// Write the first broadcast on connect so they don't have to wait 5s
	broadcast, err := server.BuildBroadcast()
	if err != nil {
		server.logf("%v", err)
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

				log.Info().Uint("clientId", uint(client.id)).
					Str("target", target).
					Msg("client attempting connect")

				if client.server != nil && client.server.IsReference(target) {
					break
				}

				for _, gameServer := range server.manager.Servers {
					if !gameServer.IsReference(target) || gameServer.Status != manager.ServerOK {
						continue
					}

					client.server = gameServer

					log.Info().Uint("clientId", uint(client.id)).
						Str("reference", gameServer.Reference()).
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

			var generic protocol.GenericMessage
			err := cbor.Unmarshal(msg, &generic)
			if err == nil && packetMessage.Op == protocol.DisconnectOp {
				target := client.server
				if target == nil {
					break
				}

				client.server = nil
				log.Info().Msgf("Client %d disconnected", client.id)
				target.SendDisconnect(client.id)
			}

		case msg := <-client.send:
			err := WriteTimeout(ctx, time.Second*5, c, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (server *Cluster) Broadcast(msg []byte) {
	server.clientMutex.Lock()
	defer server.clientMutex.Unlock()

	for client := range server.clients {
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

func (server *Cluster) AddClient(s *WSClient) {
	server.clientMutex.Lock()
	server.clients[s] = struct{}{}
	server.clientMutex.Unlock()
}

func (server *Cluster) RemoveClient(client *WSClient) {
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

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	l, _ := net.Listen("tcp", "0.0.0.0:29999")
	log.Printf("listening on http://%v", l.Addr())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := NewCluster()

	httpServer := &http.Server{
		Handler: cluster,
	}

	go cluster.StartServers(ctx)
	go cluster.StartWatcher(ctx)
	go cluster.PollMessages(ctx)

	errc := make(chan error, 1)
	go func() {
		errc <- httpServer.Serve(l)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	httpServer.Shutdown(ctx)
	cluster.Shutdown()
}
