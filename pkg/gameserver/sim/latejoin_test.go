package sim_test

import (
	"fmt"
	"testing"

	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestLateJoin_SeesExistingPlayers(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c1 := sim.ConnectAndSpawn(server, "Early1")
	c2 := sim.ConnectAndSpawn(server, "Early2")

	for i := 0; i < 5; i++ {
		sim.Tick(server, 33, c1, c2)
	}

	c3 := sim.ConnectAndSpawnWith(server, "LateJoiner", c1, c2)

	if c3.KnownPlayers == nil {
		t.Fatal("late joiner has no known players")
	}
	if c3.KnownPlayers[c1.CN] != "Early1" {
		t.Errorf("late joiner doesn't know about Early1 (CN %d)", c1.CN)
	}
	if c3.KnownPlayers[c2.CN] != "Early2" {
		t.Errorf("late joiner doesn't know about Early2 (CN %d)", c2.CN)
	}
}

func TestLateJoin_SeesExistingFrags(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	attacker := sim.ConnectAndSpawn(server, "Killer")
	victim := sim.ConnectAndSpawn(server, "Victim")

	for round := 0; round < 3; round++ {
		for !attacker.IsAlive() {
			attacker.QueueRespawn()
			sim.Tick(server, 100, attacker, victim)
			sim.Tick(server, 0, attacker, victim)
		}
		for !victim.IsAlive() {
			victim.QueueRespawn()
			sim.Tick(server, 100, attacker, victim)
			sim.Tick(server, 0, attacker, victim)
		}
		for i := 0; i < 5; i++ {
			attacker.ShootAt(victim)
			sim.Tick(server, 33, attacker, victim)
		}
	}

	if attacker.Frags == 0 {
		t.Fatal("attacker should have frags")
	}

	lateJoiner := sim.ConnectAndSpawnWith(server, "LateJoiner", attacker, victim)

	if lateJoiner.ResumeFrags == nil {
		t.Fatal("late joiner has no resume data")
	}
	resumedFrags, ok := lateJoiner.ResumeFrags[attacker.CN]
	if !ok {
		t.Fatal("late joiner doesn't have attacker in resume data")
	}
	if resumedFrags == 0 {
		t.Error("late joiner should see attacker's frags > 0")
	}
}

func TestLateJoin_CorrectMapAndMode(t *testing.T) {
	server := newServerWithMode(gamemode.Insta, "dust2")
	c1 := sim.ConnectAndSpawn(server, "Early")

	for i := 0; i < 10; i++ {
		sim.Tick(server, 33, c1)
	}

	c2 := sim.ConnectAndSpawnWith(server, "Late", c1)

	if c2.Map != "dust2" {
		t.Errorf("late joiner expected map dust2, got %s", c2.Map)
	}
	if c2.GameMode != gamemode.Insta {
		t.Errorf("late joiner expected Insta, got %s", c2.GameMode)
	}
	if c2.Health != 1 {
		t.Errorf("late joiner expected 1 health in insta, got %d", c2.Health)
	}
}

func TestLateJoin_TeamAssignment(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")
	for i := 0; i < 4; i++ {
		sim.ConnectAndSpawn(server, fmt.Sprintf("Player%d", i))
	}

	late := sim.ConnectAndSpawn(server, "LateTeam")
	if late.Team == "" || late.Team == "none" {
		t.Error("late joiner should have a team in teamplay")
	}
}

func TestLateJoin_AfterMapChange(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c1 := sim.ConnectAndSpawn(server, "Early")

	server.ChangeMap(int32(gamemode.Effic), "turbine")
	sim.Tick(server, 0, c1)
	sim.Tick(server, 0, c1)

	c2 := sim.ConnectAndSpawnWith(server, "Late", c1)

	if c2.Map != "turbine" {
		t.Errorf("expected map turbine, got %s", c2.Map)
	}
	if c2.GameMode != gamemode.Effic {
		t.Errorf("expected Effic, got %s", c2.GameMode)
	}
	if c2.Armour != 100 {
		t.Errorf("expected 100 armour in effic, got %d", c2.Armour)
	}
}

func TestLateJoin_CTF_SeesFlags(t *testing.T) {
	server := newServerWithMode(gamemode.CTF, "complex")
	c1 := sim.ConnectAndSpawn(server, "Early1")
	c2 := sim.ConnectAndSpawnWith(server, "Early2", c1)

	// First player inits flags
	c1.InitFlags(100, 100, 512, 900, 900, 512)
	sim.Tick(server, 0, c1, c2)

	// Late joiner
	c3 := sim.ConnectAndSpawnWith(server, "Late", c1, c2)

	if c3.GameMode != gamemode.CTF {
		t.Errorf("late joiner expected CTF, got %s", c3.GameMode)
	}
	if !c3.IsAlive() {
		t.Error("late joiner should be alive in CTF")
	}
	if c3.Team == "" || c3.Team == "none" {
		t.Errorf("late joiner should have a team in CTF, got %q", c3.Team)
	}
}
