package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>

*/
import "C"

import (
	"log"
	"net"
	"unsafe"
)

type PeerState uint

type Peer struct {
	Address *net.UDPAddr
	State   PeerState
	cPeer   *C.ENetPeer
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
		Address: &net.UDPAddr{
			IP:   ip,
			Port: int(cPeer.address.port),
		},
		State: PeerState(cPeer.state),
		cPeer: cPeer,
	}

	h.peers[cPeer] = p

	return p
}

func (h *Host) Disconnect(p *Peer, reason ID) {
	C.enet_peer_disconnect(p.cPeer, C.enet_uint32(reason))
	delete(h.peers, p.cPeer)
}

func (p *Peer) Send(channel uint8, payload []byte) {
	if len(payload) == 0 {
		return
	}

	flags := ^uint32(PacketFlagNoAllocate) // always allocate (safer with CGO usage below)
	if channel == 1 {
		flags = flags & PacketFlagReliable
	}

	log.Println("sending", payload, "to", p.Address.String())

	packet := C.enet_packet_create(unsafe.Pointer(&payload[0]), C.size_t(len(payload)), C.enet_uint32(flags))
	C.enet_peer_send(p.cPeer, C.enet_uint8(channel), packet)
}
