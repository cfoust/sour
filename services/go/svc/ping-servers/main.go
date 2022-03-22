package main

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>


ENetSocket initSocket() {
	ENetSocket sock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
	enet_socket_set_option(sock, ENET_SOCKOPT_NONBLOCK, 1);
	enet_socket_set_option(sock, ENET_SOCKOPT_BROADCAST, 1);
	return sock;
}

void pingServer(ENetSocket socket, const char *host, int port) {
	fprintf(stdout, "pinging %s %d\n", host, port);
	ENetAddress serverAddress = { ENET_HOST_ANY, ENET_PORT_ANY };
	serverAddress.port = port;

	int result = enet_address_set_host(&serverAddress, host);
	if (result < 0) return;

	ENetBuffer buf;
	char ping[10];
	ping[0] = 2;
	buf.data = ping;
	buf.dataLength = 10;
	enet_socket_send(socket, &serverAddress, &buf, 1);
	fprintf(stdout, "Sent ping\n");

	sleep(1);

	enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
	char response[8192];
	buf.data = response;
	buf.dataLength = 8192;
	while(enet_socket_wait(socket, &events, 0) >= 0 && events)
	{
		int len = enet_socket_receive(socket, &serverAddress, &buf, 1);
		fprintf(stdout, "len=%d\n", len);
		if (len <= 0) return;  
		fprintf(stdout, "Got pong\n");
	}
	fprintf(stdout, "events=%d\n", events);
}

void destroySocket(ENetSocket sock) {
	enet_socket_destroy(sock);
}

*/
import "C"

func main() {
	pingSocket := C.initSocket()
	C.pingServer(pingSocket, C.CString("localhost"), 28787)
	C.destroySocket(pingSocket)
}
