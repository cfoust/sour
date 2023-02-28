package game

import (
	"fmt"
	"log"
	"time"

	"github.com/sauerbraten/timer"
	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/entity"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
)

type PickupMode interface {
	HandlesPackets
	NeedsMapInfo() bool
	PickupsInitPacket() []interface{}
}

type noMapInfo struct{}

func (*noMapInfo) NeedsMapInfo() bool { return false }

type timedPickup struct {
	id int32
	entity.Pickup
	pendingSpawn *timer.Timer
}

type handlesPickups struct {
	s       Server
	pickups map[int32]*timedPickup
}

var _ PickupMode = &handlesPickups{}

func handlingPickups(s Server) *handlesPickups {
	return &handlesPickups{
		s:       s,
		pickups: map[int32]*timedPickup{},
	}
}

func (m *handlesPickups) spawnDelayed(p *timedPickup) {
	delayDependingOnNumPlayers := func() time.Duration {
		numPlayers := m.s.NumberOfPlayers()
		if numPlayers < 3 {
			return 4
		}
		if numPlayers > 4 {
			return 2
		}
		return 3
	}

	var delay time.Duration
	switch p.Typ {
	case entity.PickupShotgun,
		entity.PickupMinigun,
		entity.PickupRocketLauncher,
		entity.PickupRifle,
		entity.PickupGrenadeLauncher,
		entity.PickupPistol:
		delay = 4 * delayDependingOnNumPlayers()
	case entity.PickupHealth:
		delay = 5 * delayDependingOnNumPlayers()
	case entity.PickupGreenArmour:
		delay = 20
	case entity.PickupYellowArmor:
		delay = 30
	case entity.PickupBoost:
		delay = 60
	case entity.PickupQuadDamage:
		delay = 70
	default:
		panic(fmt.Sprintf("unhandled entity type %d pickup.delay", p.Typ))
	}
	p.pendingSpawn = timer.AfterFunc(delay*time.Second, func() {
		m.s.Broadcast(nmc.PickupSpawn, p.id)
	})
	go p.pendingSpawn.Start()
}

func (m *handlesPickups) NeedsMapInfo() bool {
	return len(m.pickups) == 0
}

func (m *handlesPickups) HandlePacket(p *Player, packetType nmc.ID, pkt *protocol.Packet) bool {
	switch packetType {
	case nmc.PickupList:
		if len(m.pickups) > 0 || p.State == playerstate.Spectator {
			for n, ok := pkt.GetInt(); ok && n >= 0 && len(*pkt) > 0; n, ok = pkt.GetInt() {
				// read and discard
			}
			break
		}
		m.initPickups(pkt)

	case nmc.PickupTry:
		entityID, ok := pkt.GetInt()
		if !ok {
			log.Println("could not read entity ID from entity pickup packet:", p)
			break
		}
		if len(m.pickups) == 0 || p.State != playerstate.Alive {
			break
		}
		pu, ok := m.pickups[entityID]
		if !ok {
			log.Printf("player tried to pick up unknown ent with ID %d", entityID)
			break
		}
		if pu.pendingSpawn.TimeLeft() > 0 {
			log.Printf("player tried to pick up %d, but it hasn't spawned", entityID)
			// pick up either didn't spawn yet or another player got it first
			break
		}
		if !p.CanPickup(pu) {
			log.Printf("player can't pick up %v", pu)
			break
		}
		m.spawnDelayed(pu)
		m.s.Broadcast(nmc.PickupAck, entityID, p.CN)
		p.Pickup(pu)

	default:
		log.Println("received unrelated packet", packetType, pkt)
		return false
	}

	return true
}

func (m *handlesPickups) initPickups(pkt *protocol.Packet) {
	const maxPickups = 10_000

	for len(*pkt) > 0 {
		id, ok := pkt.GetInt()
		if !ok {
			log.Println("couldn't read pickup ID from itemlist packet")
			return
		}
		if id < 0 || id > maxPickups {
			return
		}

		_typ, ok := pkt.GetInt()
		if !ok {
			log.Println("couldn't read pickup type from itemlist packet")
			return
		}
		typ := entity.ID(_typ)
		if typ < entity.PickupShotgun || typ > entity.PickupQuadDamage {
			log.Println("pickup type from itemlist packet outside of range [Shotgun..Quad]")
			return
		}

		p := &timedPickup{
			id:     id,
			Pickup: entity.Pickups[typ],
		}
		switch typ {
		case entity.PickupGreenArmour,
			entity.PickupYellowArmor,
			entity.PickupBoost,
			entity.PickupQuadDamage:
			m.spawnDelayed(p)
		default:
			p.pendingSpawn = timer.NewTimer(0) // 0 time left -> treated as spawned
		}

		m.pickups[id] = p
	}
}

func (m *handlesPickups) PickupsInitPacket() []interface{} {
	q := []interface{}{}
	for id, p := range m.pickups {
		if p.pendingSpawn.TimeLeft() == 0 {
			q = append(q, id, p.Typ)
		}
	}
	return append(q, -1)
}

func (m *handlesPickups) Pause() {
	for _, p := range m.pickups {
		p.pendingSpawn.Pause()
	}
}

func (m *handlesPickups) Resume() {
	for _, p := range m.pickups {
		p.pendingSpawn.Start()
	}
}

func (m *handlesPickups) CleanUp() {
	for id, p := range m.pickups {
		if p.pendingSpawn != nil {
			p.pendingSpawn.Stop()
		}
		delete(m.pickups, id)
	}
}
