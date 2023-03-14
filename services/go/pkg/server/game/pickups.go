package game

import (
	"fmt"
	"log"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/entity"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/timer"
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
		m.s.Broadcast(P.ItemSpawn{
			Index: p.id,
		})
	})
	go p.pendingSpawn.Start()
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
		m.s.Broadcast(P.ItemAck{entityID, int32(p.CN)})
		p.Pickup(pu)

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
			p.pendingSpawn = timer.NewTimer(0) // 0 time left -> treated as spawned
		}

		m.pickups[id] = p
	}
}

func (m *handlesPickups) PickupsInitPacket() P.Message {
	message := P.ItemList{}
	for id, p := range m.pickups {
		if p.pendingSpawn.TimeLeft() == 0 {
			message.Items = append(message.Items, P.Item{id, int32(p.Typ)})
		}
	}
	return message
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
