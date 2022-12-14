package watcher

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

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

void pingServer(ENetSocket socket, ENetAddress address) {
	ENetBuffer buf;
	char ping[10];
	ping[0] = 2;
	buf.data = ping;
	buf.dataLength = 1;
	enet_socket_send(socket, &address, &buf, 1);

}

int receiveServer(ENetSocket socket, ENetAddress address, void * output) {
	enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
	ENetBuffer buf;
	buf.data = output;
	buf.dataLength = 128;
	while(enet_socket_wait(socket, &events, 0) >= 0 && events)
	{
		int len = enet_socket_receive(socket, &address, &buf, 1);
		return len;
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
	Host string
	Port int
}

type ServerInfo struct {
	name  string
	_map  string
	sdesc string
}

type Server struct {
	address *C.ENetAddress `cbor:"-"`
	socket  C.ENetSocket   `cbor:"-"`
	Info    []byte         `cbor:"info"`
	Length  int
}

type Servers map[Address]Server

type Watcher struct {
	serverMutex  sync.Mutex
	servers      Servers
}

func NewWatcher() *Watcher {
	watcher := &Watcher{
		servers: make(Servers),
	}

	return watcher
}

func FetchServers() ([]Address, error) {
	var servers []Address

	socket, err := enet.NewSocket("master.sauerbraten.org", 28787)
	defer socket.DestroySocket()
	if err != nil {
		fmt.Println("Error creating socket")
		return servers, err
	}

	err = socket.SendString("list\n")
	if err != nil {
		fmt.Println("Error listing servers")
		return servers, err
	}

	output, length := socket.Receive()
	if length < 0 {
		fmt.Println("Error receiving server list")
		return servers, errors.New("Failed to receive server list")
	}

	// Collect the list of servers
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

		servers = append(servers, Address{host, port})
	}

	return servers, nil
}

func (watcher *Watcher) UpdateServerList() {
	newServers, err := FetchServers()
	if err != nil {
		fmt.Println("Failed to fetch servers")
		return
	}

	watcher.serverMutex.Lock()
	oldServers := watcher.servers
	// We want to preserve the sockets from the old servers as
	// pings may have arrived as this operation happened
	for _, key := range newServers {
		if _, exists := oldServers[key]; !exists {
			server := Server{
				address: nil,
				Info:    make([]byte, 256),
			}

			enetAddress := C.resolveServer(C.CString(key.Host), C.int(key.Port+1))
			if enetAddress.host == C.ENET_HOST_ANY {
				continue
			}

			server.address = &enetAddress
			server.socket = C.initSocket()
			oldServers[key] = server
		}
	}

	// TODO(cfoust): 03/31/22 Detect when a server goes away and remove its fd

	watcher.servers = oldServers
	watcher.serverMutex.Unlock()
}

func (watcher *Watcher) PingServers() {
	watcher.serverMutex.Lock()
	for _, server := range watcher.servers {
		address := server.address
		socket := server.socket
		if address == nil || socket == 0 {
			continue
		}
		C.pingServer(socket, *address)
	}
	watcher.serverMutex.Unlock()
}

func (watcher *Watcher) ReceivePings() {
	watcher.serverMutex.Lock()
	for key, server := range watcher.servers {
		address := server.address
		socket := server.socket
		if address == nil || socket == 0 {
			continue
		}
		result := make([]byte, 256)
		bytesRead := C.receiveServer(socket, *address, unsafe.Pointer(&result[0]))
		if bytesRead <= 0 {
			continue
		}
		server.Info = result
		server.Length = int(bytesRead)
		watcher.servers[key] = server
	}
	watcher.serverMutex.Unlock()
}

func (watcher *Watcher) Get() Servers {
	watcher.serverMutex.Lock()
	servers := watcher.servers
	watcher.serverMutex.Unlock()
	return servers
}

func (watcher *Watcher) Watch() error {
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

	// We send pings every 5 seconds, but don't block while waiting for results
	pingTicker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-pingTicker.C:
				go watcher.PingServers()

			}
		}
	}()

	// Every second we just check for any pings that came back
	receiveTicker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-receiveTicker.C:
				go watcher.ReceivePings()

			}
		}
	}()

	return nil
}
