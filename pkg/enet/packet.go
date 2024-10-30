package enet

/*
#cgo LDFLAGS: -L./enet -lenet
#cgo CFLAGS: -I./enet/include
#include <enet/enet.h>
*/
import "C"

import (
	"unsafe"
)

const (
	PacketFlagNone               = 0
	PacketFlagReliable           = (1 << 0)
	PacketFlagUnsequenced        = (1 << 1)
	PacketFlagNoAllocate         = (1 << 2)
	PacketFlagUnreliableFragment = (1 << 3)
	PacketFlagSent               = (1 << 8)
)

type Packet struct {
	Flags uint32 // bitwise-or of ENetPacketFlag constants
	Data  []byte // allocated data for packet
}

func packetFromCPacket(cPacket *C.ENetPacket) *Packet {
	if cPacket == nil {
		return nil
	}

	return &Packet{
		Flags: uint32(cPacket.flags),
		Data:  C.GoBytes(unsafe.Pointer(cPacket.data), C.int(cPacket.dataLength)),
	}
}
