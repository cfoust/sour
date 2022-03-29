package enet

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>


ENetHost* initServer(const char *addr, int port) {
	if (enet_initialize() != 0) {
		fprintf (stderr, "An error occurred while initializing ENet.\n");
		return NULL;
	}
	atexit(enet_deinitialize);

	ENetAddress address;

	// Bind the server to the provided address
	address.host = ENET_HOST_ANY;

	// Bind the server to the provided port
	address.port = port;

	ENetHost* host = enet_host_create(&address, 128, 2, 0, 0);
	if (host == NULL) {
		fprintf(stderr, "An error occurred while trying to create an ENet server host.\n");
		exit(EXIT_FAILURE);
	}

	return host;
}

ENetEvent serviceHost(ENetHost* host) {
	ENetEvent event;

	int e = 0;
	do {
		e = enet_host_service(host, &event, host->peerCount ? 1 : 1000);
	} while (e <= 0 || (event.type == ENET_EVENT_TYPE_RECEIVE && event.packet->dataLength == 0));

	return event;
}
*/
import "C"

import (
	"errors"
)

func NewHost(laddr string, lport int) (*Host, error) {
	cHost := C.initServer(C.CString(laddr), C.int(lport))
	if cHost == nil {
		return nil, errors.New("an error occured initializing the ENet host in C")
	}

	return &Host{
		cHost: cHost,
		peers: map[*C.ENetPeer]*Peer{},
	}, nil
}

type Host struct {
	cHost *C.ENetHost
	peers map[*C.ENetPeer]*Peer
}

func (h *Host) Service() <-chan Event {
	events := make(chan Event)
	go func() {
		for {
			cEvent := C.serviceHost(h.cHost)
			events <- h.eventFromCEvent(&cEvent)
		}
	}()
	return events
}
