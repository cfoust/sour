package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>
#include <stdio.h>

int isCommandPresent(ENetPeer *peer, enet_uint16 seq) {
	ENetOutgoingCommand * outgoingCommand = NULL;
	ENetListIterator currentCommand;
	for (currentCommand = enet_list_begin(&peer->outgoingReliableCommands);
	currentCommand != enet_list_end(&peer->outgoingReliableCommands);
	currentCommand = enet_list_next(currentCommand))
	{
		outgoingCommand = (ENetOutgoingCommand *) currentCommand;
		if (outgoingCommand->reliableSequenceNumber == seq) {
			return 1;
		}
	}

	return 0;
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
	"net"
	"sync"
	"unsafe"
)

type PeerState uint

type PendingPacket struct {
	Sequence uint16
	Done     chan bool
}

type Peer struct {
	Address *net.UDPAddr
	State   PeerState
	CPeer   *C.ENetPeer

	// Messages for which we have yet to receive ACKs
	Pending []PendingPacket
	Mutex   sync.Mutex
}

func (h *Host) peerFromCPeer(cPeer *C.ENetPeer) *Peer {
	if cPeer == nil {
		return nil
	}

	// peer exists already
	if p, ok := h.peers[cPeer]; ok {
		return p
	}

	ipBytes := uint32(cPeer.address.host)
	ip := net.IPv4(byte((ipBytes<<24)>>24), byte((ipBytes<<16)>>24), byte((ipBytes<<8)>>24), byte(ipBytes>>24))

	p := &Peer{
		Pending: make([]PendingPacket, 0),
		Address: &net.UDPAddr{
			IP:   ip,
			Port: int(cPeer.address.port),
		},
		State: PeerState(cPeer.state),
		CPeer: cPeer,
	}

	h.peers[cPeer] = p

	return p
}

func (h *Host) Disconnect(p *Peer, reason ID) {
	C.enet_peer_disconnect(p.CPeer, C.enet_uint32(reason))
	delete(h.peers, p.CPeer)
}

func (p *Peer) CheckACKs() {
	p.Mutex.Lock()
	newPending := make([]PendingPacket, 0)
	for _, pending := range p.Pending {
		if C.isCommandPresent(p.CPeer, C.ushort(pending.Sequence)) == 1 {
			newPending = append(newPending, pending)
			continue
		}

		pending.Done <- true
	}
	p.Pending = newPending
	p.Mutex.Unlock()
}

func (p *Peer) Send(channel uint8, payload []byte) <-chan bool {
	flags := ^uint32(PacketFlagNoAllocate) // always allocate (safer with CGO usage below)
	if channel == 1 || channel == 2 {
		flags = flags & PacketFlagReliable
	}

	packet := C.enet_packet_create(unsafe.Pointer(&payload[0]), C.size_t(len(payload)), C.enet_uint32(flags))
	C.enet_peer_send(p.CPeer, C.enet_uint8(channel), packet)
	command := C.getLastCommand(p.CPeer)

	done := make(chan bool, 1)
	p.Mutex.Lock()
	p.Pending = append(p.Pending, PendingPacket{
		Sequence: uint16(command),
		Done: done,
	})
	p.Mutex.Unlock()
	return done
}
