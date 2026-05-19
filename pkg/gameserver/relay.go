package gameserver

import (
	"github.com/cfoust/sour/pkg/game/protocol"
)

// SyncRelay buffers position and client packet data and flushes it to all
// other clients during Step(). It replaces the old goroutine-based
// relay/relay.go which had a deadlock bug (mutex held during send) and a
// packet duplication bug (double-append in flush).
type SyncRelay struct {
	positions  map[uint32][]protocol.Message // cn -> latest position
	packets    map[uint32][]protocol.Message // cn -> queued client packets
	sessionIDs map[uint32]uint32             // cn -> sessionID
}

func NewSyncRelay() SyncRelay {
	return SyncRelay{
		positions:  make(map[uint32][]protocol.Message),
		packets:    make(map[uint32][]protocol.Message),
		sessionIDs: make(map[uint32]uint32),
	}
}

func (r *SyncRelay) AddClient(cn uint32, sessionID uint32) {
	r.sessionIDs[cn] = sessionID
}

func (r *SyncRelay) RemoveClient(cn uint32) {
	delete(r.positions, cn)
	delete(r.packets, cn)
	delete(r.sessionIDs, cn)
}

// SetPosition stores the latest position for a client. Only the most recent
// position is kept — older ones are overwritten.
func (r *SyncRelay) SetPosition(cn uint32, msgs ...protocol.Message) {
	r.positions[cn] = msgs
}

// AddPacket queues client packets (chat, gun select, taunt, etc.) for relay.
func (r *SyncRelay) AddPacket(cn uint32, msgs ...protocol.Message) {
	r.packets[cn] = append(r.packets[cn], msgs...)
}

// Flush sends all buffered positions and packets to all other clients.
// Each client receives everyone else's data but not their own.
func (r *SyncRelay) Flush(out *OutputBuffer) {
	if len(r.sessionIDs) < 2 {
		// Clear buffers even if we don't send
		for cn := range r.positions {
			delete(r.positions, cn)
		}
		for cn := range r.packets {
			delete(r.packets, cn)
		}
		return
	}

	// Flush positions on channel 0
	for senderCN, pos := range r.positions {
		if len(pos) == 0 {
			continue
		}
		for receiverCN, receiverSession := range r.sessionIDs {
			if receiverCN == senderCN {
				continue
			}
			out.SendChan(receiverSession, 0, pos...)
		}
	}

	// Flush client packets on channel 1 with ClientPacket header
	for senderCN, pkts := range r.packets {
		if len(pkts) == 0 {
			continue
		}
		// Prepend ClientPacket header (mirrors N_CLIENT wrapping)
		wrapped := make([]protocol.Message, 0, 1+len(pkts))
		wrapped = append(wrapped, protocol.ClientPacket{
			Client: int32(senderCN),
		})
		wrapped = append(wrapped, pkts...)

		for receiverCN, receiverSession := range r.sessionIDs {
			if receiverCN == senderCN {
				continue
			}
			out.SendChan(receiverSession, 1, wrapped...)
		}
	}

	// Clear buffers
	for cn := range r.positions {
		delete(r.positions, cn)
	}
	for cn := range r.packets {
		delete(r.packets, cn)
	}
}

// FlushPositionAndSend flushes a client's buffered position data and then
// sends an additional message (like N_TELEPORT or N_JUMPPAD) to all other
// clients on channel 0.
func (r *SyncRelay) FlushPositionAndSend(cn uint32, p protocol.Message, out *OutputBuffer) {
	// First send any buffered position data
	if pos, ok := r.positions[cn]; ok && len(pos) > 0 {
		for receiverCN, receiverSession := range r.sessionIDs {
			if receiverCN == cn {
				continue
			}
			out.SendChan(receiverSession, 0, pos...)
		}
		delete(r.positions, cn)
	}

	// Then send the additional message
	for receiverCN, receiverSession := range r.sessionIDs {
		if receiverCN == cn {
			continue
		}
		out.SendChan(receiverSession, 0, p)
	}
}
