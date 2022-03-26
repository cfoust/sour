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
}

func NewRelayServer() *RelayServer {
	server := &RelayServer{
		clientMessageLimit: 16,
		logf:               log.Printf,
		clients:            make(map[*Client]struct{}),
	}

	return server
}

func (server *RelayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})

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
	return c.Write(ctx, websocket.MessageText, msg)
}

func main() {
	l, _ := net.Listen("tcp", "localhost:1233")
	log.Printf("listening on http://%v", l.Addr())

	server := NewRelayServer()
	httpServer := &http.Server{
		Handler: server,
	}

	serverChannel := make(chan watcher.Servers)
	watcher := watcher.NewWatcher()

	go watcher.Watch(serverChannel)

	announceTicker := time.NewTicker(1 * time.Second)
	announceChannel := make(chan bool)
	go func() {
		for {
			select {
			case <-announceChannel:
				return
			case <-announceTicker.C:
				server.Broadcast(make([]byte, 2))

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
	announceTicker.Stop()
	announceChannel <- true
}
