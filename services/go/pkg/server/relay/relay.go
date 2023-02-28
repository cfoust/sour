package relay

import (
	"errors"
	"sync"
	"time"

	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
)

type sendFunc func(channel uint8, payload []byte)

// Relay relays positional data between clients
type Relay struct {
	μ sync.Mutex

	incPositionsNotifs chan uint32              // channel on which clients notify the broker about new packets
	incPositions       map[uint32]<-chan []byte // clients' update channels by topic
	positions          map[uint32][]byte

	incClientPacketsNotifs chan uint32
	incClientPackets       map[uint32]<-chan []byte
	clientPackets          map[uint32][]byte

	send map[uint32]sendFunc
}

func New() *Relay {
	r := &Relay{
		incPositionsNotifs: make(chan uint32),
		incPositions:       map[uint32]<-chan []byte{},
		positions:          map[uint32][]byte{},

		incClientPacketsNotifs: make(chan uint32),
		incClientPackets:       map[uint32]<-chan []byte{},
		clientPackets:          map[uint32][]byte{},

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
				func(uint32, []byte) []byte { return nil },
				0,
			)

			// publish client packets
			r.flush(
				r.clientPackets,
				func(cn uint32, pkt []byte) []byte {
					p := packet.Encode(nmc.Client, cn)
					p.PutUint(uint32(len(pkt)))
					return p
				},
				1,
			)

		case cn := <-r.incPositionsNotifs:
			r.receive(cn, r.incPositions, func(pos []byte) {
				if len(pos) == 0 {
					delete(r.positions, cn)
				} else {
					r.positions[cn] = pos
				}
			})

		case cn := <-r.incClientPacketsNotifs:
			r.receive(cn, r.incClientPackets, func(pkt []byte) {
				r.clientPackets[cn] = append(r.clientPackets[cn], pkt...)
			})
		}
	}
}

func (r *Relay) AddClient(cn uint32, sf sendFunc) (positions *Publisher, packets *Publisher) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if _, ok := r.send[cn]; ok {
		// client is already being serviced
		return nil, nil
	}

	r.send[cn] = sf

	positions, posCh := newPublisher(cn, r.incPositionsNotifs)
	r.incPositions[cn] = posCh

	packets, pktCh := newPublisher(cn, r.incClientPacketsNotifs)
	r.incClientPackets[cn] = pktCh

	return
}

func (r *Relay) RemoveClient(cn uint32) error {
	r.μ.Lock()
	defer r.μ.Unlock()

	if _, ok := r.send[cn]; !ok {
		return errors.New("no such client")
	}

	delete(r.incPositions, cn)
	delete(r.positions, cn)
	delete(r.incClientPackets, cn)
	delete(r.clientPackets, cn)
	delete(r.send, cn)

	return nil
}

func (r *Relay) FlushPositionAndSend(cn uint32, p []byte) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if pos := r.positions[cn]; pos != nil {
		for _cn, send := range r.send {
			if _cn == cn {
				continue
			}
			send(0, pos)
		}
		delete(r.positions, cn)
	}

	for _cn, send := range r.send {
		if _cn == cn {
			continue
		}
		send(0, p)
	}
}

func (r *Relay) receive(cn uint32, from map[uint32]<-chan []byte, process func(upd []byte)) {
	r.μ.Lock()
	defer r.μ.Unlock()

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

func (r *Relay) flush(packets map[uint32][]byte, prefix func(uint32, []byte) []byte, channel uint8) {
	r.μ.Lock()
	defer r.μ.Unlock()

	if len(packets) == 0 || len(r.send) < 2 {
		return
	}

	order := make([]uint32, 0, len(r.send))
	lengths := map[uint32]int{}
	combined := make([]byte, 0, 2*len(packets)*40)

	for cn := range r.send {
		order = append(order, cn)
		pkt := packets[cn]
		if pkt == nil {
			continue
		}
		pkt = append(prefix(cn, pkt), pkt...)
		lengths[cn] = len(pkt)
		combined = append(combined, pkt...)
	}

	if len(combined) == 0 {
		return
	}

	combined = append(combined, combined...)

	offset := 0
	for _, cn := range order {
		l := lengths[cn]
		offset += l
		p := combined[offset : (len(combined)/2)-l+offset]
		r.send[cn](channel, p)
	}

	// clear packets
	for cn := range packets {
		delete(packets, cn)
	}
}
