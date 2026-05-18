package relay

import (
	"sort"
	"time"

	"github.com/cfoust/sour/pkg/game/protocol"

	"github.com/sasha-s/go-deadlock"
)

type sendFunc func(channel uint8, payload []protocol.Message)

// Relay relays positional data between clients
type Relay struct {
	mutex deadlock.Mutex

	incPositionsNotifs chan uint32                          // channel on which clients notify the broker about new packets
	incPositions       map[uint32]<-chan []protocol.Message // clients' update channels by topic
	positions          map[uint32][]protocol.Message

	incClientPacketsNotifs chan uint32
	incClientPackets       map[uint32]<-chan []protocol.Message
	clientPackets          map[uint32][]protocol.Message

	send map[uint32]sendFunc
}

func New() *Relay {
	r := &Relay{
		incPositionsNotifs: make(chan uint32),
		incPositions:       map[uint32]<-chan []protocol.Message{},
		positions:          map[uint32][]protocol.Message{},

		incClientPacketsNotifs: make(chan uint32),
		incClientPackets:       map[uint32]<-chan []protocol.Message{},
		clientPackets:          map[uint32][]protocol.Message{},

		send: map[uint32]sendFunc{},
	}

	go r.loop()

	return r
}

func (r *Relay) loop() {
	t := time.Tick(11 * time.Millisecond)
	for {
		select {
		case <-t:
			// publish positions
			r.flush(
				r.positions,
				func(uint32, []protocol.Message) []protocol.Message { return nil },
				0,
			)

			// publish client packets
			r.flush(
				r.clientPackets,
				func(cn uint32, pkt []protocol.Message) []protocol.Message {
					return append([]protocol.Message{protocol.ClientPacket{
						Client: int32(cn),
						// Length is handled by cluster
					}}, pkt...)
				},
				1,
			)

		case cn := <-r.incPositionsNotifs:
			r.receive(cn, r.incPositions, func(pos []protocol.Message) {
				if len(pos) == 0 {
					delete(r.positions, cn)
				} else {
					r.positions[cn] = pos
				}
			})

		case cn := <-r.incClientPacketsNotifs:
			r.receive(cn, r.incClientPackets, func(pkt []protocol.Message) {
				r.clientPackets[cn] = append(r.clientPackets[cn], pkt...)
			})
		}
	}
}

func (r *Relay) AddClient(cn uint32, sf sendFunc) (positions *Publisher, packets *Publisher) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.send[cn]; ok {
		// ZOMBIE CN FIX: Force cleanup of existing CN before adding new client
		// This handles cases where RemoveClient failed due to race conditions,
		// preventing persistent "zombie CNs" that would block future connections
		// with the same CN and cause permanent player invisibility issues.
		r.forceRemoveClient(cn)
	}

	r.send[cn] = sf

	positions, posCh := newPublisher(cn, r.incPositionsNotifs)
	r.incPositions[cn] = posCh

	packets, pktCh := newPublisher(cn, r.incClientPacketsNotifs)
	r.incClientPackets[cn] = pktCh

	return
}

func (r *Relay) RemoveClient(cn uint32) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, ok := r.send[cn]; !ok {
		// ZOMBIE CN FIX: Don't return error for already-removed clients
		// This prevents logging errors during normal double-disconnect scenarios
		return nil
	}

	r.forceRemoveClient(cn)
	return nil
}

// forceRemoveClient performs cleanup without error checking - used to fix zombie CNs
func (r *Relay) forceRemoveClient(cn uint32) {
	// Force cleanup all relay state for this CN
	// Note: We don't need to drain channels as they will be garbage collected
	// when the publisher is closed properly by the client
	delete(r.incPositions, cn)
	delete(r.positions, cn)
	delete(r.incClientPackets, cn)
	delete(r.clientPackets, cn)
	delete(r.send, cn)
}

func (r *Relay) FlushPositionAndSend(cn uint32, p protocol.Message) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Create deterministic order for consistent packet delivery
	order := make([]uint32, 0, len(r.send))
	for _cn := range r.send {
		if _cn != cn {
			order = append(order, _cn)
		}
	}
	sort.Slice(order, func(i, j int) bool {
		return order[i] < order[j]
	})

	if pos := r.positions[cn]; pos != nil {
		for _, _cn := range order {
			r.send[_cn](0, pos)
		}
		delete(r.positions, cn)
	}

	for _, _cn := range order {
		r.send[_cn](0, []protocol.Message{p})
	}
}

func (r *Relay) receive(cn uint32, from map[uint32]<-chan []protocol.Message, process func(upd []protocol.Message)) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	ch, ok := from[cn]
	if !ok {
		// ignore clients that were already removed
		return
	}

	p, ok := <-ch
	if ok {
		process(p)
	}
}

func (r *Relay) flush(packets map[uint32][]protocol.Message, prefix func(uint32, []protocol.Message) []protocol.Message, channel uint8) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(packets) == 0 || len(r.send) < 2 {
		return
	}

	// CRITICAL FIX: Only process clients who actually have packets to avoid slice bound errors.
	// Previously, we iterated over all connected clients (r.send) but only clients with packets
	// were included in the combined array, causing offset misalignment and wrong packet delivery.

	// Only include clients who actually have packets to send
	clientsWithPackets := make([]uint32, 0, len(packets))
	lengths := map[uint32]int{}
	combined := make([]protocol.Message, 0, 2*len(packets)*40)

	// Create deterministic order of clients with packets
	for cn := range packets {
		if packets[cn] != nil {
			clientsWithPackets = append(clientsWithPackets, cn)
		}
	}
	// Sort to ensure consistent ordering across flushes
	sort.Slice(clientsWithPackets, func(i, j int) bool {
		return clientsWithPackets[i] < clientsWithPackets[j]
	})

	// Build combined array with only clients who have packets
	for _, cn := range clientsWithPackets {
		pkt := packets[cn] // We know this is not nil
		pkt = append(prefix(cn, pkt), pkt...)
		lengths[cn] = len(pkt)
		combined = append(combined, pkt...)
	}

	if len(combined) == 0 {
		return
	}

	combined = append(combined, combined...)

	// Send to ALL connected clients, but exclude each client's own packets
	allClients := make([]uint32, 0, len(r.send))
	for cn := range r.send {
		allClients = append(allClients, cn)
	}
	sort.Slice(allClients, func(i, j int) bool {
		return allClients[i] < allClients[j]
	})

	for _, receiverCN := range allClients {
		// Build packet for this receiver by excluding their own packets
		var receiverPackets []protocol.Message
		offset := 0
		for _, senderCN := range clientsWithPackets {
			l := lengths[senderCN]
			if senderCN != receiverCN {
				// Include this sender's packets for this receiver
				senderData := combined[offset : offset+l]
				receiverPackets = append(receiverPackets, senderData...)
			}
			offset += l
		}
		
		if len(receiverPackets) > 0 {
			r.send[receiverCN](channel, receiverPackets)
		}
	}

	// clear packets
	for cn := range packets {
		delete(packets, cn)
	}
}
