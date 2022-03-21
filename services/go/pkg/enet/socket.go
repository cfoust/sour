package enet

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>


ENetSocket initSocket(const char *host, int port) {
	ENetAddress serverAddress = { ENET_HOST_ANY, ENET_PORT_ANY };
        serverAddress.port = port;

	int result = enet_address_set_host(&serverAddress, host);
	if (result < 0) return ENET_SOCKET_NULL;

	ENetSocket sock = enet_socket_create(ENET_SOCKET_TYPE_STREAM);

	enet_socket_set_option(sock, ENET_SOCKOPT_NONBLOCK, 1);

	result = enet_socket_connect(sock, &serverAddress);
	if (result < 0) return ENET_SOCKET_NULL;

	return sock;
}

void destroySocket(ENetSocket sock) {
    enet_socket_destroy(sock);
}

*/
import "C"

import (
	"errors"
)

type Socket struct {
	cSocket C.ENetSocket
}

func NewSocket(host string, port int) (*Socket, error) {
	cSocket := C.initSocket(C.CString(host), C.int(port))
	if cSocket == C.ENET_SOCKET_NULL {
		return nil, errors.New("an error occured initializing the ENet socket in C")
	}

	return &Socket{
		cSocket: cSocket,
	}, nil
}

func (sock *Socket) DestroySocket() {
	C.destroySocket(sock.cSocket)
}
