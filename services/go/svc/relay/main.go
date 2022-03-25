package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/cfoust/sour/pkg/enet"
	"nhooyr.io/websocket"
)

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>

ENetAddress resolveServer(const char *host, int port) {
	ENetAddress serverAddress = { ENET_HOST_ANY, ENET_PORT_ANY };
	serverAddress.port = port;

	int result = enet_address_set_host(&serverAddress, host);
	if (result < 0) {
		serverAddress.host = ENET_HOST_ANY;
		serverAddress.port = ENET_PORT_ANY;
		return serverAddress;
	}

	return serverAddress;
}

ENetSocket initSocket() {
	ENetSocket sock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
	enet_socket_set_option(sock, ENET_SOCKOPT_NONBLOCK, 1);
	enet_socket_set_option(sock, ENET_SOCKOPT_BROADCAST, 1);
	return sock;
}

void pingServer(ENetSocket socket, ENetAddress address, void * output) {
	ENetAddress serverAddress = { ENET_HOST_ANY, ENET_PORT_ANY };
	serverAddress.host = address.host;
	serverAddress.port = address.port + 1;

	ENetBuffer buf;
	char ping[10];
	ping[0] = 2;
	buf.data = ping;
	buf.dataLength = 10;
	enet_socket_send(socket, &serverAddress, &buf, 1);

	sleep(1);

	enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
	buf.data = output;
	buf.dataLength = 128;
	while(enet_socket_wait(socket, &events, 0) >= 0 && events)
	{
		int len = enet_socket_receive(socket, &serverAddress, &buf, 1);
		if (len <= 0) return;
	}
}

void destroySocket(ENetSocket sock) {
	enet_socket_destroy(sock);
}

*/
import "C"

type ServerInfo struct {
	name  string
	_map  string
	sdesc string
}

type Server struct {
	address *C.ENetAddress
	socket  C.ENetSocket
	host    string
	port    int
	info    *ServerInfo
}

func FetchServers() []Server {
	socket, err := enet.NewSocket("master.sauerbraten.org", 28787)
	if err != nil {
		fmt.Println("Error creating socket")
	}
	socket.SendString("list\n")
	output, length := socket.Receive()
	if length < 0 {
		fmt.Println("Error fetching server list")
		return make([]Server, 0)
	}
	socket.DestroySocket()

	// Collect the list of servers
	servers := make([]Server, 0)
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "addserver") {
			continue
		}
		parts := strings.Split(line, " ")

		if len(parts) != 3 {
			continue
		}

		host := parts[1]
		port, err := strconv.Atoi(parts[2])

		if err != nil {
			continue
		}
		servers = append(servers, Server{
			address: nil,
			host:    host,
			port:    port,
			info:    nil,
		})
	}

	// Resolve them to IPs
	for i, server := range servers {
		address := C.resolveServer(C.CString(server.host), C.int(server.port))
		if address.host == C.ENET_HOST_ANY {
			continue
		}
		(&servers[i]).address = &address
		(&servers[i]).socket = C.initSocket()
	}

	// Fill in information about them
	for _, server := range servers {
		address := server.address
		socket := server.socket
		if address == nil || socket == 0 {
			continue
		}
		result := make([]byte, 128)
		C.pingServer(socket, *address, unsafe.Pointer(&result[0]))
		fmt.Println(result)
		break
	}

	return servers
}

type Client struct {
	send      chan []byte
	closeSlow func()
}

type RelayServer struct {
	clientMessageLimit int

	// logf controls where logs are sent.
	logf func(f string, v ...interface{})

	servers *[]Server

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
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"relay"},
	})

	if err != nil {
		server.logf("%v", err)
		return
	}

	defer c.Close(websocket.StatusInternalError, "operational fault during relay")

	if c.Subprotocol() != "relay" {
		c.Close(websocket.StatusPolicyViolation, "client failed to specify relay protocol")
		return
	}

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
