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

	"github.com/cfoust/sour/pkg/watcher"
	"github.com/fxamacker/cbor/v2"
	"nhooyr.io/websocket"
)

type Client struct {
	send      chan []byte
	closeSlow func()
}

type RelayServer struct {
	clientMessageLimit int

	// logf controls where logs are sent.
	logf func(f string, v ...interface{})

	clientMutex sync.Mutex
	clients     map[*Client]struct{}

	serverWatcher *watcher.Watcher
}

func NewRelayServer(serverWatcher *watcher.Watcher) *RelayServer {
	server := &RelayServer{
		clientMessageLimit: 16,
		logf:               log.Printf,
		clients:            make(map[*Client]struct{}),
		serverWatcher:      serverWatcher,
	}

	return server
}

func (server *RelayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (server *RelayServer) Subscribe(ctx context.Context, c *websocket.Conn) error {
	ctx = c.CloseRead(ctx)

	client := &Client{
		send: make(chan []byte, server.clientMessageLimit),
		closeSlow: func() {
			c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

	server.AddClient(client)
	defer server.RemoveClient(client)

	broadcast, err := server.BuildBroadcast()
	if err != nil {
		server.logf("%v", err)
		return err
	}
	err = WriteTimeout(ctx, time.Second*5, c, broadcast)

	for {
		select {
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

func (server *RelayServer) Broadcast(msg []byte) {
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

func (server *RelayServer) BuildBroadcast() ([]byte, error) {
	servers := server.serverWatcher.Get()
	serverArray := make([]AggregatedServer, len(servers))
	index := 0
	for key, server := range servers {
		serverArray[index] = AggregatedServer{
			Host:   key.Host,
			Port:   key.Port,
			Info:   server.Info,
			Length: server.Length,
		}
		index++
	}

	bytes, err := cbor.Marshal(serverArray)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// addSubscriber registers a subscriber.
func (server *RelayServer) AddClient(s *Client) {
	server.clientMutex.Lock()
	server.clients[s] = struct{}{}
	server.clientMutex.Unlock()
}

// deleteSubscriber deletes the given client.
func (server *RelayServer) RemoveClient(client *Client) {
	server.clientMutex.Lock()
	delete(server.clients, client)
	server.clientMutex.Unlock()
}

func WriteTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageBinary, msg)
}

type AggregatedServer struct {
	Host   string
	Port   int
	Info   []byte
	Length int
}

func main() {
	l, _ := net.Listen("tcp", "0.0.0.0:29999")
	log.Printf("listening on http://%v", l.Addr())

	serverWatcher := watcher.NewWatcher()
	go serverWatcher.Watch()

	server := NewRelayServer(serverWatcher)
	httpServer := &http.Server{
		Handler: server,
	}

	broadcastTicker := time.NewTicker(10 * time.Second)
	broadcastChannel := make(chan bool)
	go func() {
		for {
			select {
			case <-broadcastChannel:
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

	httpServer.Shutdown(context.Background())
	broadcastTicker.Stop()
	broadcastChannel <- true
}
