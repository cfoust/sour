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
		if (outgoingCommand == NULL) {
			continue;
		}
		if (outgoingCommand->reliableSequenceNumber == seq) {
			return 1;
		}
	}

	return 0;
}

*/
import "C"

import (
	"net"

	"github.com/sasha-s/go-deadlock"
)

type PeerState uint

type PendingPacket struct {
	Sequence uint16
	Error    chan error
}

type QueuedPacket struct {
	Channel uint8
	Data    []byte
	Error   chan error
}

type Peer struct {
	Address *net.UDPAddr
	State   PeerState
	CPeer   *C.ENetPeer

	// Messages for which we have yet to receive ACKs
	Pending []PendingPacket
	Queued  []QueuedPacket
	Mutex   deadlock.Mutex
}

func (h *Host) peerFromCPeer(cPeer *C.ENetPeer, host *Host) *Peer {
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
		Queued:  make([]QueuedPacket, 0),
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

		pending.Error <- nil
	}
	p.Pending = newPending
	p.Mutex.Unlock()
}

func (p *Peer) Send(channel uint8, payload []byte) <-chan error {
	done := make(chan error, 1)
	p.Mutex.Lock()
	p.Queued = append(p.Queued, QueuedPacket{
		Channel: channel,
		Data:    payload,
		Error:   done,
	})
	p.Mutex.Unlock()

	return done
}
