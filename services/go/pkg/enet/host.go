package enet

/*
#cgo LDFLAGS: -lenet
#include <stdio.h>
#include <stdlib.h>
#include <enet/enet.h>

ENetHost* initClient(const char *addr, int port) {
	if (enet_initialize() != 0) {
		fprintf (stderr, "An error occurred while initializing ENet.\n");
		return NULL;
	}
	atexit(enet_deinitialize);

	ENetAddress address;

	address.host = ENET_HOST_ANY;

	address.port = port;

	enet_address_set_host(&address, addr);

	ENetHost* host = enet_host_create(NULL, 2, 3, 0, 0);
	if (host == NULL) {
		fprintf(stderr, "An error occurred while trying to create an ENet server host.\n");
		exit(EXIT_FAILURE);
	}

	enet_host_connect(host, &address, 3, 0);

	return host;
}

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

	ENetHost* host = enet_host_create(&address, 128, 3, 0, 0);
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
		e = enet_host_service(host, &event, 5);
	} while (e <= 0 || (event.type == ENET_EVENT_TYPE_RECEIVE && event.packet->dataLength == 0));

	return event;
}

void cleanupHost(ENetHost* host) {
	enet_host_destroy(host);
}

enet_uint16 getLastCommand(ENetPeer *peer) {
	ENetOutgoingCommand * outgoingCommand = NULL;
	ENetListIterator currentCommand;
	for (currentCommand = enet_list_begin(&peer->outgoingReliableCommands);
	currentCommand != enet_list_end(&peer->outgoingReliableCommands);
	currentCommand = enet_list_next(currentCommand))
	{
		outgoingCommand = (ENetOutgoingCommand *) currentCommand;
	}
	if (outgoingCommand == NULL) {
		return 0;
	}

	return outgoingCommand->reliableSequenceNumber;
}

*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/sasha-s/go-deadlock"
)

func NewConnectHost(laddr string, lport int) (*Host, error) {
	cHost := C.initClient(C.CString(laddr), C.int(lport))
	if cHost == nil {
		return nil, errors.New("an error occured initializing the ENet host in C")
	}

	return &Host{
		cHost: cHost,
		peers: map[*C.ENetPeer]*Peer{},
	}, nil
}

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
	Mutex deadlock.Mutex
}

func (h *Host) Service() <-chan Event {
	events := make(chan Event)
	go func() {
		for {
			serviced := false
			var event C.ENetEvent
			for !serviced {
				if C.enet_host_check_events(h.cHost, &event) <= 0 {
					if C.enet_host_service(h.cHost, &event, 5) <= 0 {
						break
					}
					serviced = true
				}

				events <- h.eventFromCEvent(&event)
			}

			for _, peer := range h.peers {
				peer.Mutex.Lock()

				for _, queued := range peer.Queued {
					flags := ^uint32(PacketFlagNoAllocate) // always allocate (safer with CGO usage below)
					if queued.Channel == 1 || queued.Channel == 2 {
						flags = flags & PacketFlagReliable
					}

					payload := queued.Data
					packet := C.enet_packet_create(
						unsafe.Pointer(&payload[0]),
						C.size_t(len(payload)),
						C.enet_uint32(flags),
					)
					result := C.enet_peer_send(peer.CPeer, C.enet_uint8(queued.Channel), packet)
					if result == -1 {
						queued.Done <- false
						continue
					}
					command := C.getLastCommand(peer.CPeer)
					peer.Pending = append(peer.Pending, PendingPacket{
						Sequence: uint16(command),
						Done:     queued.Done,
					})
				}

				peer.Queued = make([]QueuedPacket, 0)

				peer.Mutex.Unlock()
			}

			for _, peer := range h.peers {
				peer.CheckACKs()
			}
		}
	}()
	return events
}

func (h *Host) Shutdown() {
	C.cleanupHost(h.cHost)
}
