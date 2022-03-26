package enet

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>


ENetSocket _initSocket(const char *host, int port) {
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

void _destroySocket(ENetSocket sock) {
	enet_socket_destroy(sock);
}

int sendSocket(ENetSocket sock, const void * req, size_t numBytes) {
	int reqlen = numBytes;
	ENetBuffer buf;
	while(reqlen > 0)
	{
		enet_uint32 events = ENET_SOCKET_WAIT_SEND;
		if(enet_socket_wait(sock, &events, 250) >= 0 && events) 
		{
			buf.data = (void *)req;
			buf.dataLength = reqlen;
			int sent = enet_socket_send(sock, NULL, &buf, 1);
			if(sent < 0) break;
			req += sent;
			reqlen -= sent;
			if(reqlen <= 0) break;
		}
	}
}

int receiveSocket(ENetSocket sock, void * data, size_t maxSize) {
	ENetBuffer buf;
	size_t dataLength = 0;
	for (;;)
	{
		enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
		if(enet_socket_wait(sock, &events, 250) >= 0 && events)
		{
			if(dataLength >= maxSize) return -1;
			buf.data = data + dataLength;
			buf.dataLength = maxSize - dataLength;
			int recv = enet_socket_receive(sock, NULL, &buf, 1);
			if(recv <= 0) break;
			dataLength += recv;
		}
	}

	return dataLength;
}

*/
import "C"

import (
	"errors"
	"unsafe"
)

type Socket struct {
	cSocket C.ENetSocket
}

func NewSocket(host string, port int) (*Socket, error) {
	cSocket := C._initSocket(C.CString(host), C.int(port))
	if cSocket == C.ENET_SOCKET_NULL {
		return nil, errors.New("an error occured initializing the ENet socket in C")
	}

	return &Socket{
		cSocket: cSocket,
	}, nil
}

func (sock *Socket) DestroySocket() {
	C._destroySocket(sock.cSocket)
}

func (sock *Socket) SendString(str string) error {
	sock.Send([]byte(str))
	return nil
}

func (sock *Socket) Send(payload []byte) {
	if len(payload) == 0 {
		return
	}

	C.sendSocket(sock.cSocket, unsafe.Pointer(&payload[0]), C.size_t(len(payload)))
}

func (sock *Socket) Receive() (string, int) {
	buf := make([]byte, 8192)
	length := C.receiveSocket(sock.cSocket, unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	return string(buf), int(length)
}
