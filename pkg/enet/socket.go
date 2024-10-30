package enet

/*
#cgo LDFLAGS: -L./enet -lenet
#cgo CFLAGS: -I./enet/include
#include <enet/enet.h>
#include <stdio.h>
#include <stdlib.h>

ENetSocket _initConnectSocket(const char *host, int port) {
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

ENetSocket _initDatagramSocket(int port) {
	ENetAddress address = { ENET_HOST_ANY, ENET_PORT_ANY };
        address.port = port;

	ENetSocket sock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
	if (sock == ENET_SOCKET_NULL) return sock;

	int result = enet_socket_bind(sock, &address);
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

	return 0;
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

int receiveDatagram(ENetSocket sock, ENetAddress * addr, void * data, size_t maxSize) {
	ENetBuffer buf;

        buf.data = data;
        buf.dataLength = maxSize;

	return enet_socket_receive(sock, addr, &buf, 1);
}

void sendDatagram(ENetSocket sock, ENetAddress * addr, void * data, size_t length) {
	ENetBuffer buf;

        buf.data = data;
        buf.dataLength = length;

	enet_socket_send(sock, addr, &buf, 1);
}

*/
import "C"

import (
	"errors"
	"unsafe"
)

type ENetSocketType int

type Socket struct {
	cSocket C.ENetSocket
}

func NewConnectSocket(host string, port int) (*Socket, error) {
	cSocket := C._initConnectSocket(C.CString(host), C.int(port))
	if cSocket == C.ENET_SOCKET_NULL {
		return nil, errors.New("an error occured initializing the ENet socket in C")
	}

	return &Socket{
		cSocket: cSocket,
	}, nil
}

func NewDatagramSocket(port int) (*Socket, error) {
	cSocket := C._initDatagramSocket(C.int(port))
	if cSocket == C.ENET_SOCKET_NULL {
		return nil, errors.New("an error occured initializing the ENet datagram socket in C")
	}

	return &Socket{
		cSocket: cSocket,
	}, nil
}

func (sock *Socket) DestroySocket() {
	if sock.cSocket == 0 {
		return
	}
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

func (sock *Socket) SendDatagram(address C.ENetAddress, data []byte) {
	// Copy the data to avoid a segfault
	buf := make([]byte, len(data))
	copy(buf, data)
	C.sendDatagram(sock.cSocket, &address, unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
}

type SocketMessage struct {
	Address C.ENetAddress
	Data    []byte
}

func (sock *Socket) Service() <-chan SocketMessage {
	out := make(chan SocketMessage)
	go func() {
		buf := make([]byte, 5000)
		for {
			address := C.ENetAddress{}
			numBytes := C.receiveDatagram(
				sock.cSocket,
				&address,
				unsafe.Pointer(&buf[0]),
				C.size_t(len(buf)),
			)

			if numBytes < 0 {
				continue
			}

			copied := make([]byte, numBytes)
			copy(copied, buf)
			out <- SocketMessage{
				Address: address,
				Data:    copied,
			}
		}
	}()

	return out
}
