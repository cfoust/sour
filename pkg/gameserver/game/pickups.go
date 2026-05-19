package game

import (
	"fmt"
	"log"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/deadline"
	"github.com/cfoust/sour/pkg/gameserver/protocol/entity"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

type PickupMode interface {
	HandlesPackets
	NeedsMapInfo() bool
	PickupsInitPacket() P.Message
}

type noMapInfo struct{}

func (*noMapInfo) NeedsMapInfo() bool { return false }

type timedPickup struct {
	id int32
	entity.Pickup
	pendingSpawn deadline.Deadline
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
	delayDependingOnNumPlayers := func() int64 {
		numPlayers := m.s.NumberOfPlayers()
		if numPlayers < 3 {
			return 4
		}
		if numPlayers > 4 {
			return 2
		}
		return 3
	}

	var delaySeconds int64
	switch p.Typ {
	case entity.PickupShotgun,
		entity.PickupMinigun,
		entity.PickupRocketLauncher,
		entity.PickupRifle,
		entity.PickupGrenadeLauncher,
		entity.PickupPistol:
		delaySeconds = 4 * delayDependingOnNumPlayers()
	case entity.PickupHealth:
		delaySeconds = 5 * delayDependingOnNumPlayers()
	case entity.PickupGreenArmour:
		delaySeconds = 20
	case entity.PickupYellowArmor:
		delaySeconds = 30
	case entity.PickupBoost:
		delaySeconds = 60
	case entity.PickupQuadDamage:
		delaySeconds = 70
	default:
		panic(fmt.Sprintf("unhandled entity type %d pickup.delay", p.Typ))
	}
	p.pendingSpawn.Set(m.s.GameClock(), delaySeconds*1000)
}

// Tick checks all pickups for expired spawn timers and broadcasts ItemSpawn.
func (m *handlesPickups) Tick(clock int64) {
	for _, p := range m.pickups {
		if p.pendingSpawn.Expired(clock) {
			p.pendingSpawn.Stop()
			m.s.Broadcast(P.ItemSpawn{
				Index: p.id,
			})
		}
	}
}

func (m *handlesPickups) NeedsMapInfo() bool {
	return len(m.pickups) == 0
}

func (m *handlesPickups) HandlePacket(p *Player, message P.Message) bool {
	switch message.Type() {
	case P.N_ITEMLIST:
		itemList := message.(P.ItemList)

		if len(m.pickups) > 0 || p.State == playerstate.Spectator {
			break
		}

		m.initPickups(itemList)

	case P.N_ITEMPICKUP:
		itemPickup := message.(P.ItemPickup)

		if len(m.pickups) == 0 || p.State != playerstate.Alive {
			break
		}

		entityID := itemPickup.Item

		pu, ok := m.pickups[int32(entityID)]
		if !ok {
			log.Printf("player tried to pick up unknown ent with ID %d", entityID)
			break
		}
		clock := m.s.GameClock()
		if pu.pendingSpawn.TimeLeftMs(clock) > 0 {
			log.Printf("player tried to pick up %d, but it hasn't spawned", entityID)
			break
		}
		if !p.CanPickup(clock, pu) {
			break
		}
		m.spawnDelayed(pu)
		m.s.Broadcast(P.ItemAck{entityID, int32(p.CN)})
		p.Pickup(clock, pu)

	default:
		log.Println("received unrelated packet", message.Type())
		return false
	}

	return true
}

func (m *handlesPickups) initPickups(pkt P.ItemList) {
	const maxPickups = 10_000

	for _, item := range pkt.Items {
		typ := entity.ID(item.Type)
		if typ < entity.PickupShotgun || typ > entity.PickupQuadDamage {
			log.Println("pickup type from itemlist packet outside of range [Shotgun..Quad]")
			return
		}

		id := int32(item.Index)
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
			// 0 time left -> treated as spawned (deadline inactive)
		}

		m.pickups[id] = p
	}
}

func (m *handlesPickups) PickupsInitPacket() P.Message {
	clock := m.s.GameClock()
	message := P.ItemList{}
	for id, p := range m.pickups {
		if p.pendingSpawn.TimeLeftMs(clock) == 0 {
			message.Items = append(message.Items, P.Item{id, int32(p.Typ)})
		}
	}
	return message
}

func (m *handlesPickups) Pause() {
	clock := m.s.GameClock()
	for _, p := range m.pickups {
		p.pendingSpawn.Pause(clock)
	}
}

func (m *handlesPickups) Resume() {
	clock := m.s.GameClock()
	for _, p := range m.pickups {
		p.pendingSpawn.Resume(clock)
	}
}

func (m *handlesPickups) CleanUp() {
	for id, p := range m.pickups {
		p.pendingSpawn.Stop()
		delete(m.pickups, id)
	}
}
