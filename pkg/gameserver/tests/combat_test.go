package tests

import (
	"testing"

	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestKill_FragAndDeathTracking(t *testing.T) {
	server := newTestServer()
	attacker := sim.ConnectAndSpawn(server, "Attacker")
	victim := sim.ConnectAndSpawn(server, "Victim")

	for i := 0; i < 5; i++ {
		attacker.ShootAt(victim)
		sim.Tick(server, 33, attacker, victim)
	}

	if attacker.Frags < 1 {
		t.Error("attacker should have at least 1 frag")
	}
	if victim.Deaths < 1 {
		t.Error("victim should have at least 1 death")
	}
	if victim.IsAlive() {
		t.Error("victim should be dead")
	}
}

func TestRespawn_AfterDeath(t *testing.T) {
	server := newTestServer()
	attacker := sim.ConnectAndSpawn(server, "Attacker")
	victim := sim.ConnectAndSpawn(server, "Victim")

	for i := 0; i < 5; i++ {
		attacker.ShootAt(victim)
		sim.Tick(server, 33, attacker, victim)
	}

	if victim.IsAlive() {
		t.Fatal("victim should be dead before respawn test")
	}

	victim.QueueRespawn()
	sim.Tick(server, 100, attacker, victim)
	sim.Tick(server, 0, attacker, victim)

	if !victim.IsAlive() {
		t.Error("victim should be alive after respawn")
	}
	if victim.Health != 100 {
		t.Errorf("expected full health after respawn, got %d", victim.Health)
	}
}

func TestSuicide(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	fragsBefore := c.Frags

	c.Send(1, sim.SuicideMsg())
	sim.Tick(server, 33, c)

	if c.IsAlive() {
		t.Error("player should be dead after suicide")
	}
	if c.Frags >= fragsBefore {
		t.Error("suicide should decrement frags")
	}
}

func TestIntermission_TriggeredByTimeUp(t *testing.T) {
	server := gameserver.New(&gameserver.Config{
		MaxClients:       32,
		MatchLength:      1, // 1 second match
		DefaultGameSpeed: 100,
		DefaultMode:      "ffa",
		DefaultMap:       "complex",
		Maps:             []string{"dust2"},
	})
	server.StartGame(server.StartMode(gamemode.FFA), "complex")

	c := sim.ConnectAndSpawn(server, "Player")

	// Advance past match length + intermission delay (10s)
	for i := 0; i < 400; i++ {
		sim.Tick(server, 33, c)
	}

	// The map should have changed (intermission fires at ~1s, map change at ~11s)
	t.Logf("Map after intermission: %s", c.Map)
}

func TestSpectator_Toggle(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	if !c.IsAlive() {
		t.Fatal("should be alive before spectating")
	}

	c.Send(1, sim.SpectatorMsg(c.CN, true))
	sim.Tick(server, 33, c)
	sim.Tick(server, 0, c)

	if c.State != 5 { // playerstate.Spectator
		t.Errorf("expected spectator state (5), got %d", c.State)
	}

	c.Send(1, sim.SpectatorMsg(c.CN, false))
	sim.Tick(server, 33, c)

	if c.State == 5 {
		t.Error("should not be spectating after unspec")
	}
}
