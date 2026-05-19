package sim_test

import (
	"testing"

	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestMapChange(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	c1.ShootAt(c2)
	sim.Tick(server, 33, c1, c2)

	if c2.Health >= 100 {
		t.Fatal("expected damage before map change")
	}

	server.ChangeMap(int32(gamemode.Insta), "dust2")
	sim.Tick(server, 0, c1, c2)
	sim.Tick(server, 0, c1, c2)

	if c1.Map != "dust2" {
		t.Errorf("c1 expected map dust2, got %s", c1.Map)
	}
	if c1.GameMode != gamemode.Insta {
		t.Errorf("c1 expected Insta mode, got %s", c1.GameMode)
	}
	if c2.Map != "dust2" {
		t.Errorf("c2 expected map dust2, got %s", c2.Map)
	}
	if !c1.IsAlive() || !c2.IsAlive() {
		t.Error("both clients should be alive after map change")
	}
	if c1.Health != 1 {
		t.Errorf("c1 expected 1 health after insta map change, got %d", c1.Health)
	}
}

func TestMapChange_ResetsFrags(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	attacker := sim.ConnectAndSpawn(server, "Attacker")
	victim := sim.ConnectAndSpawn(server, "Victim")

	for i := 0; i < 5; i++ {
		attacker.ShootAt(victim)
		sim.Tick(server, 33, attacker, victim)
	}

	if attacker.Frags == 0 {
		t.Fatal("expected frags before map change")
	}

	server.ChangeMap(int32(gamemode.FFA), "dust2")
	sim.Tick(server, 0, attacker, victim)
	sim.Tick(server, 0, attacker, victim)

	serverAttacker := server.Clients.GetClientByID(attacker.SessionID)
	if serverAttacker.Frags != 0 {
		t.Errorf("expected server-side frags to be 0 after map change, got %d",
			serverAttacker.Frags)
	}
}

func TestMapChange_ModeSwitch(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	server.ChangeMap(int32(gamemode.Teamplay), "dust2")
	sim.Tick(server, 0, c)
	sim.Tick(server, 0, c)

	if c.GameMode != gamemode.Teamplay {
		t.Errorf("expected Teamplay after mode switch, got %s", c.GameMode)
	}
	if c.Team == "" || c.Team == "none" {
		t.Error("expected team assignment after switching to teamplay")
	}

	server.ChangeMap(int32(gamemode.FFA), "complex")
	sim.Tick(server, 0, c)
	sim.Tick(server, 0, c)

	if c.GameMode != gamemode.FFA {
		t.Errorf("expected FFA after switching back, got %s", c.GameMode)
	}
}

func TestMapChange_ToFromCTF(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	// Switch to CTF
	server.ChangeMap(int32(gamemode.CTF), "dust2")
	sim.Tick(server, 0, c)
	sim.Tick(server, 0, c)

	if c.GameMode != gamemode.CTF {
		t.Errorf("expected CTF, got %s", c.GameMode)
	}
	if !c.IsAlive() {
		t.Error("should be alive after switching to CTF")
	}

	// Switch back to FFA
	server.ChangeMap(int32(gamemode.FFA), "complex")
	sim.Tick(server, 0, c)
	sim.Tick(server, 0, c)

	if c.GameMode != gamemode.FFA {
		t.Errorf("expected FFA, got %s", c.GameMode)
	}
}
