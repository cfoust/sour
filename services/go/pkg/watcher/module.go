package watcher

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/enet"
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

// By default Sauerbraten seems to only allow one game server per IP address
// (or at least hostname) which is a little weird.
type Address struct {
	hostname string
	port     int
}

type ServerInfo struct {
	name  string
	_map  string
	sdesc string
}

type Server struct {
	address *C.ENetAddress
	socket  C.ENetSocket
	info    *ServerInfo
}

type Servers map[Address]Server

type Watcher struct {
	serverMutex sync.Mutex
	servers     Servers
}

func NewWatcher() *Watcher {
	watcher := &Watcher{
		servers: make(Servers),
	}

	return watcher
}

func FetchServers() (Servers, error) {
	socket, err := enet.NewSocket("master.sauerbraten.org", 28787)
	defer socket.DestroySocket()
	if err != nil {
		fmt.Println("Error creating socket")
		return make(Servers), err
	}

	err = socket.SendString("list\n")
	if err != nil {
		fmt.Println("Error listing servers")
		return make(Servers), err
	}

	output, length := socket.Receive()
	if length < 0 {
		fmt.Println("Error receiving server list")
		return make(Servers), errors.New("Failed to receive server list")
	}

	// Collect the list of servers
	servers := make(Servers)
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

		servers[Address{host, port}] = Server{
			address: nil,
			info:    nil,
		}
	}

	// Resolve them to IPs
	for address, server := range servers {
		enetAddress := C.resolveServer(C.CString(address.hostname), C.int(address.port+1))
		if enetAddress.host == C.ENET_HOST_ANY {
			continue
		}

		server.address = &enetAddress
		server.socket = C.initSocket()
		servers[address] = server
	}

	return servers, nil
}

func (watcher *Watcher) UpdateServerList() {
	newServers, err := FetchServers()
	if err != nil {
		fmt.Println("Failed to fetch servers")
		return
	}
	fmt.Println(newServers)
}

func (watcher *Watcher) Watch(out chan Servers) error {
	done := make(chan bool)

	go watcher.UpdateServerList()

	// We update the list of servers every minute
	serverListTicker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-serverListTicker.C:
				go watcher.UpdateServerList()

			}
		}
	}()

	return nil
}

//func FetchServers() []Server {

//// Fill in information about them
//for _, server := range servers {
//address := server.address
//socket := server.socket
//if address == nil || socket == 0 {
//continue
//}
//result := make([]byte, 128)
//C.pingServer(socket, *address, unsafe.Pointer(&result[0]))
//fmt.Println(result)
//break
//}

//return servers
//}
