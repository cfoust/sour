package tests

import (
	"testing"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestGunSelect(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	// FFA starts with pistol (6)
	if c.GunSelect != int32(weapon.Pistol) {
		t.Errorf("expected pistol on spawn, got %d", c.GunSelect)
	}

	// Switch to grenade launcher (only FFA weapon with ammo besides pistol)
	c.Send(1, P.GunSelect{GunSelect: int32(weapon.GrenadeLauncher)})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.SelectedWeapon.ID != weapon.GrenadeLauncher {
		t.Errorf("server should reflect gun switch to GL, got %d",
			serverClient.SelectedWeapon.ID)
	}
}

func TestGunSelect_Relayed(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Switcher")
	c2 := sim.ConnectAndSpawn(server, "Observer")

	c1.Send(1, P.GunSelect{GunSelect: int32(weapon.GrenadeLauncher)})
	outputs := sim.Tick(server, 33, c1, c2)

	// c2 should receive the gun switch via relay
	receivedGunSelect := false
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID {
			for _, msg := range pkt.Messages {
				if gs, ok := msg.(P.GunSelect); ok {
					if gs.GunSelect == int32(weapon.GrenadeLauncher) {
						receivedGunSelect = true
					}
				}
			}
		}
	}
	if !receivedGunSelect {
		t.Error("observer should receive gun select via relay")
	}
}

func TestShoot_DecreasesAmmo(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Shooter")

	serverClient := server.Clients.GetClientByID(c.SessionID)
	ammoBefore := serverClient.Ammo[weapon.Pistol]

	c.Shoot(c.X+10, c.Y, c.Z)
	sim.Tick(server, 33, c)

	ammoAfter := serverClient.Ammo[weapon.Pistol]
	if ammoAfter >= ammoBefore {
		t.Errorf("shooting should decrease ammo (before=%d, after=%d)",
			ammoBefore, ammoAfter)
	}
}

func TestShoot_OutOfAmmo(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Shooter")
	other := sim.ConnectAndSpawn(server, "Target")

	// Drain all pistol ammo (40 rounds)
	for i := 0; i < 45; i++ {
		c.Shoot(c.X+10, c.Y, c.Z)
		sim.Tick(server, 33, c, other)
	}

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Ammo[weapon.Pistol] != 0 {
		t.Errorf("should have 0 ammo after draining, got %d",
			serverClient.Ammo[weapon.Pistol])
	}

	// Shooting with no ammo at a target should do no damage
	healthBefore := other.Health
	c.ShootAt(other)
	sim.Tick(server, 33, c, other)

	if other.Health < healthBefore {
		t.Error("shooting with 0 ammo should not deal damage")
	}
}

func TestSound_Relayed(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Sounder")
	c2 := sim.ConnectAndSpawn(server, "Listener")

	c1.Send(1, P.Sound{Sound: 5})
	outputs := sim.Tick(server, 33, c1, c2)

	receivedSound := false
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID {
			for _, msg := range pkt.Messages {
				if _, ok := msg.(P.Sound); ok {
					receivedSound = true
				}
			}
		}
	}
	if !receivedSound {
		t.Error("sound should be relayed to other players")
	}
}

func TestTaunt_Relayed(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Taunter")
	c2 := sim.ConnectAndSpawn(server, "Observer")

	c1.Send(1, P.Taunt{})
	outputs := sim.Tick(server, 33, c1, c2)

	receivedTaunt := false
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID {
			for _, msg := range pkt.Messages {
				if _, ok := msg.(P.Taunt); ok {
					receivedTaunt = true
				}
			}
		}
	}
	if !receivedTaunt {
		t.Error("taunt should be relayed to other players")
	}
}
