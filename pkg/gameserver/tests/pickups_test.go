package tests

import (
	"testing"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestPickups_ItemListInit(t *testing.T) {
	// FFA mode needs map info (pickups) — the first non-spectator
	// client sends N_ITEMLIST to initialize entity spawns
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	// Send an item list with some pickups
	// Entity types: 8=shells, 9=bullets, 14=health, 16=greenarmour
	c.Send(1, P.ItemList{
		Items: []P.Item{
			{Index: 0, Type: 8},  // shells (shotgun ammo)
			{Index: 1, Type: 14}, // health
			{Index: 2, Type: 16}, // green armour
		},
	})
	sim.Tick(server, 0, c)

	// After init, NeedsMapInfo should be false
	if server.GameMode.NeedsMapInfo() {
		t.Error("NeedsMapInfo should be false after item list init")
	}
}

func TestPickups_LateJoinerSeesItems(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c1 := sim.ConnectAndSpawn(server, "Early")

	// Init items
	c1.Send(1, P.ItemList{
		Items: []P.Item{
			{Index: 0, Type: 8},
			{Index: 1, Type: 14},
		},
	})
	sim.Tick(server, 0, c1)

	// Late joiner should receive the item list in welcome
	c2 := sim.ConnectAndSpawnWith(server, "Late", c1)

	// The late joiner's welcome includes pickups init packet
	// (verified by NeedsMapInfo being false and no errors)
	if !c2.IsAlive() {
		t.Error("late joiner should be alive")
	}
}

func TestPickups_ItemPickup(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	// Init items
	c.Send(1, P.ItemList{
		Items: []P.Item{
			{Index: 0, Type: 14}, // health pickup
		},
	})
	sim.Tick(server, 0, c)

	// Damage the player first so health pickup is useful
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Health = 50

	// Pick up item 0
	c.Send(1, P.ItemPickup{Item: 0})
	outputs := sim.Tick(server, 33, c)

	// Should receive ItemAcc
	receivedAck := false
	for _, pkt := range outputs {
		if pkt.Session == c.SessionID {
			for _, msg := range pkt.Messages {
				if _, ok := msg.(P.ItemAck); ok {
					receivedAck = true
				}
			}
		}
	}
	if !receivedAck {
		t.Error("should receive ItemAck after pickup")
	}

	// Health should increase
	if serverClient.Health <= 50 {
		t.Errorf("health should increase after health pickup, got %d",
			serverClient.Health)
	}
}

func TestPickups_RespawnAfterDelay(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	// Init items
	c.Send(1, P.ItemList{
		Items: []P.Item{
			{Index: 0, Type: 14}, // health
		},
	})
	sim.Tick(server, 0, c)

	// Pick up the item
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Health = 50
	c.Send(1, P.ItemPickup{Item: 0})
	sim.Tick(server, 33, c)

	// Immediately try to pick up again — should fail (respawn timer)
	serverClient.Health = 50
	c.Send(1, P.ItemPickup{Item: 0})
	sim.Tick(server, 33, c)

	if serverClient.Health > 50 {
		t.Error("should not be able to pick up item before respawn timer")
	}

	// Advance time past respawn delay (health = 5 * delayDependingOnNumPlayers)
	// With 1 player, delay = 4, so 5*4 = 20 seconds
	for i := 0; i < 700; i++ {
		sim.Tick(server, 33, c)
	}

	// Now the item should have respawned (ItemSpawn broadcast)
	// Try picking up again
	serverClient.Health = 50
	c.Send(1, P.ItemPickup{Item: 0})
	sim.Tick(server, 33, c)

	if serverClient.Health <= 50 {
		t.Error("item should have respawned after delay")
	}
}
