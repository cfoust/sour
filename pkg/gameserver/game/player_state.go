package game

import (
	"github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/deadline"
	"github.com/cfoust/sour/pkg/gameserver/protocol/armour"
	"github.com/cfoust/sour/pkg/gameserver/protocol/entity"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
)

type PlayerState struct {
	State playerstate.ID

	// The user's state before entering edit mode
	EditState playerstate.ID

	// fields that reset at spawn
	LastSpawnAttempt int64 // ms timestamp, -1 = none
	QuadDeadline    deadline.Deadline
	LastShot        int64 // ms timestamp
	GunReloadEnd    int64 // ms timestamp
	// reset at spawn to value depending on mode
	Health         int32
	Armour         int32
	ArmourType     armour.ID
	SelectedWeapon weapon.Weapon
	Ammo           map[weapon.ID]int32 // weapon → ammo

	// reset at map change
	LifeSequence    int32
	LastDeath       int64 // ms timestamp, 0 = none
	MaxHealth       int32
	Frags           int32
	Deaths          int32
	Teamkills       int32
	DamagePotential int32
	Damage          int32
	Flags           int32
}

func NewPlayerState() PlayerState {
	ps := PlayerState{
		LastSpawnAttempt: -1,
	}
	ps.Reset()
	return ps
}

func (ps *PlayerState) ToWire() protocol.EntityState {
	spawnState := protocol.EntityState{
		LifeSequence: ps.LifeSequence,
		Health:       ps.Health,
		MaxHealth:    ps.MaxHealth,
		Armour:       ps.Armour,
		Armourtype:   int32(ps.ArmourType),
		Gunselect:    int32(ps.SelectedWeapon.ID),
	}

	for _, id := range weapon.WeaponsWithAmmo {
		spawnState.Ammo[id-1] = protocol.AmmoState{
			Amount: ps.Ammo[id],
		}
	}

	return spawnState
}

func (ps *PlayerState) Spawn(clock int64) {
	ps.LifeSequence = (ps.LifeSequence + 1) % 128

	ps.LastSpawnAttempt = clock
	ps.QuadDeadline.Stop()
	ps.LastShot = 0
	ps.GunReloadEnd = 0
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

func (ps *PlayerState) CanPickup(clock int64, p *timedPickup) bool {
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
		return ps.QuadDeadline.TimeLeftMs(clock) < int64(p.MaxAmount)
	default:
		return ps.Ammo[weapon.ID(p.Typ-7)] < p.MaxAmount
	}
}

func (ps *PlayerState) Pickup(clock int64, p *timedPickup) {
	min := func(a, b int32) int32 {
		if a < b {
			return a
		}
		return b
	}
	switch p.Typ {
	case entity.PickupBoost:
		ps.MaxHealth = min(ps.MaxHealth+p.Amount, p.MaxAmount)
		ps.Health = min(ps.Health+(2*p.Amount), ps.MaxHealth)
	case entity.PickupHealth:
		ps.Health = min(ps.Health+p.Amount, ps.MaxHealth)
	case entity.PickupGreenArmour:
		ps.ArmourType = armour.Green
		ps.Armour = min(ps.Armour+p.Amount, p.MaxAmount)
	case entity.PickupYellowArmor:
		ps.ArmourType = armour.Yellow
		ps.Armour = min(ps.Armour+p.Amount, p.MaxAmount)
	case entity.PickupQuadDamage:
		timeLeft := ps.QuadDeadline.TimeLeftMs(clock)
		newTimeLeft := int64(min(int32(timeLeft)+p.Amount, p.MaxAmount))
		ps.QuadDeadline.Set(clock, newTimeLeft)
	default:
		ps.Ammo[weapon.ID(p.Typ-7)] = min(ps.Ammo[weapon.ID(p.Typ-7)]+p.Amount, p.MaxAmount)
	}
}

func (ps *PlayerState) Die(clock int64) {
	if ps.State != playerstate.Alive {
		return
	}
	ps.State = playerstate.Dead
	ps.Deaths++
	ps.LastDeath = clock
	ps.QuadDeadline.Stop()
}

// Resets a client's game state.
func (ps *PlayerState) Reset() {
	if ps.State != playerstate.Spectator {
		ps.State = playerstate.Dead
	}

	ps.LifeSequence = 0
	ps.LastDeath = 0
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
