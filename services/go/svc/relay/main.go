package main

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

import (
	"fmt"
	"github.com/cfoust/sour/pkg/enet"
	"strconv"
	"strings"
	"unsafe"
)

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

func main() {
	fmt.Println(FetchServers())
}
