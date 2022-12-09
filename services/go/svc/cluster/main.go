package main

import (
	"context"
	"errors"
	"log"
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
	"nhooyr.io/websocket"
)

type Client struct {
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
	clients       map[*Client]struct{}
	serverWatcher *watcher.Watcher
	manager       *manager.Manager
}

func NewCluster() *Cluster {
	server := &Cluster{
		logf:          log.Printf,
		clients:       make(map[*Client]struct{}),
		serverWatcher: watcher.NewWatcher(),
		manager: manager.NewManager(
			"../server/qserv",
			50000,
			51000,
		),
	}

	return server
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

func (server *Cluster) Subscribe(ctx context.Context, c *websocket.Conn) error {
	client := &Client{
		send: make(chan []byte, CLIENT_MESSAGE_LIMIT),
		closeSlow: func() {
			c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

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
			if err := cbor.Unmarshal(msg, &connectMessage); err == nil {
				log.Print(connectMessage)
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
	serverArray := make([]protocol.ServerInfo, len(servers))
	index := 0
	for key, server := range servers {
		serverArray[index] = protocol.ServerInfo{
			Host:   key.Host,
			Port:   key.Port,
			Info:   server.Info,
			Length: server.Length,
		}
		index++
	}

	infoMessage := protocol.InfoMessage{
		Op:      protocol.InfoOp,
		Master:  serverArray,
		Cluster: make([]string, 0),
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
	l, _ := net.Listen("tcp", "0.0.0.0:29999")
	log.Printf("listening on http://%v", l.Addr())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := NewCluster()

	httpServer := &http.Server{
		Handler: cluster,
	}

	go cluster.StartWatcher(ctx)

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
