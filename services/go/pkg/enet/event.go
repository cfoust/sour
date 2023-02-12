package enet

/*
#cgo LDFLAGS: -lenet
#include <enet/enet.h>
*/
import "C"

type EventType uint

const (
	EventTypeConnect    = C.ENET_EVENT_TYPE_CONNECT
	EventTypeDisconnect = C.ENET_EVENT_TYPE_DISCONNECT
	EventTypeReceive    = C.ENET_EVENT_TYPE_RECEIVE
)

type Event struct {
	Type      EventType
	Peer      *Peer
	ChannelID uint8
	Data      uint32
	Packet    *Packet
}

func (h *Host) eventFromCEvent(cEvent *C.ENetEvent) Event {
	e := Event{
		Type:      EventType(cEvent._type),
		Peer:      h.peerFromCPeer(cEvent.peer, h),
		ChannelID: uint8(cEvent.channelID),
		Data:      uint32(cEvent.data),
	}

	if e.Type == EventTypeReceive {
		e.Packet = packetFromCPacket(cEvent.packet)
		C.enet_packet_destroy(cEvent.packet)
	}

	return e

}
