package game

import (
	"time"

	"github.com/cfoust/sour/pkg/server/net/packet"
	"github.com/cfoust/sour/pkg/server/protocol/armour"
	"github.com/cfoust/sour/pkg/server/protocol/entity"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/weapon"
	"github.com/cfoust/sour/pkg/server/timer"
)

type PlayerState struct {
	State playerstate.ID

	// fields that reset at spawn
	LastSpawnAttempt time.Time
	QuadTimer        *timer.Timer
	LastShot         time.Time
	GunReloadEnd     time.Time
	// reset at spawn to value depending on mode
	Health         int32
	Armour         int32
	ArmourType     armour.ID
	SelectedWeapon weapon.Weapon
	Ammo           map[weapon.ID]int32 // weapon → ammo

	// reset at map change
	LifeSequence    int32
	LastDeath       time.Time
	MaxHealth       int32
	Frags           int
	Deaths          int
	Teamkills       int
	DamagePotential int32
	Damage          int32
	Flags           int
}

func NewPlayerState() PlayerState {
	ps := PlayerState{}
	ps.Reset()
	return ps
}

func (ps *PlayerState) ToWire() []byte {
	return packet.Encode(
		ps.LifeSequence,
		ps.Health,
		ps.MaxHealth,
		ps.Armour,
		ps.ArmourType,
		ps.SelectedWeapon.ID,
		weapon.FlattenAmmo(ps.Ammo),
	)
}

func (ps *PlayerState) Spawn() {
	ps.LifeSequence = (ps.LifeSequence + 1) % 128

	ps.LastSpawnAttempt = time.Now()
	ps.QuadTimer = nil
	ps.LastShot = time.Time{}
	ps.GunReloadEnd = time.Time{}
}

func (ps *PlayerState) SelectWeapon(id weapon.ID) (weapon.Weapon, bool) {
	if ps.State != playerstate.Alive {
		return weapon.ByID(weapon.Pistol), false
	}
	ps.SelectedWeapon = weapon.ByID(id)
	return ps.SelectedWeapon, true
}

func (ps *PlayerState) applyDamage(damage int32) {
	damageToArmour := damage * armour.Absorption(ps.ArmourType) / 100
	if damageToArmour > ps.Armour {
		damageToArmour = ps.Armour
	}
	ps.Armour -= damageToArmour
	damage -= damageToArmour
	ps.Health -= damage
}

func (ps *PlayerState) CanPickup(p *timedPickup) bool {
	switch p.Typ {
	case entity.PickupBoost:
		return ps.MaxHealth < p.MaxAmount
	case entity.PickupHealth:
		return ps.Health < ps.MaxHealth
	case entity.PickupGreenArmour:
		if ps.ArmourType == armour.Yellow || ps.Armour >= 100 {
			return true
		}
		fallthrough
	case entity.PickupYellowArmor:
		return ps.ArmourType == armour.None || ps.Armour < p.MaxAmount
	case entity.PickupQuadDamage:
		return int32(ps.QuadTimer.TimeLeft()/time.Millisecond) < p.MaxAmount
	default:
		return ps.Ammo[weapon.ID(p.Typ-7)] < p.MaxAmount
	}
}

func (ps *PlayerState) Pickup(p *timedPickup) {
	min := func(a, b int32) int32 {
		if a < b {
			return a
		}
		return b
	}
	switch p.Typ {
	case entity.PickupBoost:
		ps.MaxHealth = min(ps.MaxHealth+p.Amount, p.MaxAmount) // add 50 to max health
		ps.Health = min(ps.Health+(2*p.Amount), ps.MaxHealth)  // add 100 to health
	case entity.PickupHealth:
		ps.Health = min(ps.Health+p.Amount, ps.MaxHealth)
	case entity.PickupGreenArmour:
		ps.ArmourType = armour.Green
		ps.Armour = min(ps.Armour+p.Amount, p.MaxAmount)
	case entity.PickupYellowArmor:
		ps.ArmourType = armour.Yellow
		ps.Armour = min(ps.Armour+p.Amount, p.MaxAmount)
	case entity.PickupQuadDamage:
		timeLeft := ps.QuadTimer.TimeLeft()
		newTimeLeft := time.Duration(min(int32(timeLeft)+p.Amount, p.MaxAmount))
		if ps.QuadTimer != nil {
			ps.QuadTimer.Stop()
		}
		ps.QuadTimer = timer.NewTimer(newTimeLeft)
		go ps.QuadTimer.Start()
	default:
		ps.Ammo[weapon.ID(p.Typ-7)] = min(ps.Ammo[weapon.ID(p.Typ-7)]+p.Amount, p.MaxAmount)
	}
}

func (ps *PlayerState) Die() {
	if ps.State != playerstate.Alive {
		return
	}
	ps.State = playerstate.Dead
	ps.Deaths++
	ps.LastDeath = time.Now()
	if ps.QuadTimer != nil {
		ps.QuadTimer.Stop()
	}
}

// Resets a client's game state.
func (ps *PlayerState) Reset() {
	if ps.State != playerstate.Spectator {
		ps.State = playerstate.Dead
	}

	ps.LifeSequence = 0
	ps.LastDeath = time.Time{}
	ps.MaxHealth = 100
	ps.Frags = 0
	ps.Deaths = 0
	ps.Teamkills = 0
	ps.DamagePotential = 0
	ps.Damage = 0
	ps.Flags = 0
}

// below are Spawn methods scoped on empty structs for embedding into game modes

type ffaSpawnState struct{}

func (*ffaSpawnState) Spawn(ps *PlayerState) {
	ps.ArmourType = armour.Blue
	ps.Armour = 25
	ps.Ammo, ps.SelectedWeapon = weapon.SpawnAmmoFFA()
	ps.Health = ps.MaxHealth
}

type efficSpawnState struct{}

func (*efficSpawnState) Spawn(ps *PlayerState) {
	ps.ArmourType = armour.Green
	ps.Armour = 100
	ps.Ammo, ps.SelectedWeapon = weapon.SpawnAmmoEffic()
	ps.Health = ps.MaxHealth
}

type instaSpawnState struct{}

func (*instaSpawnState) Spawn(ps *PlayerState) {
	ps.ArmourType = armour.None
	ps.Armour = 0
	ps.Ammo, ps.SelectedWeapon = weapon.SpawnAmmoInsta()
	ps.Health, ps.MaxHealth = 1, 1
}

type tacticsSpawnState struct{}

func (*tacticsSpawnState) Spawn(ps *PlayerState) {
	ps.ArmourType = armour.Green
	ps.Armour = 100
	ps.Ammo, ps.SelectedWeapon = weapon.SpawnAmmoTactics()
	ps.Health = ps.MaxHealth
}

type ctfSpawnState struct{}

func (*ctfSpawnState) Spawn(ps *PlayerState) {
	ps.ArmourType = armour.Blue
	ps.Armour = 50
	ps.Ammo, ps.SelectedWeapon = weapon.SpawnAmmoFFA()
	ps.Health = ps.MaxHealth
}
