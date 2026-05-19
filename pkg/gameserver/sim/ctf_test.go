package sim_test

import (
	"testing"

	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

// ctfSetup creates a CTF server with flags initialized and 2 players on
// opposing teams. Returns the server, the good-team player, the evil-team
// player, and the observer (the player who received ServerInitFlags and
// can be used for flag state assertions — always the second joiner).
func ctfSetup(t *testing.T) (server *gameserver.Server, good, evil, observer *sim.Client) {
	t.Helper()
	server = newServerWithMode(gamemode.CTF, "complex")

	c1 := sim.ConnectAndSpawn(server, "Player1")

	// First player initializes flags
	c1.InitFlags(100, 100, 512, 900, 900, 512)
	sim.Tick(server, 0, c1)

	// Second player joins and receives flag state in welcome
	c2 := sim.ConnectAndSpawnWith(server, "Player2", c1)

	good, evil = c1, c2
	if c1.Team == "evil" {
		good, evil = c2, c1
	}

	// c2 always received ServerInitFlags in their welcome
	observer = c2

	// Send positions so the server knows where players are (needed for
	// flag drop location when a flag carrier dies)
	good.QueuePosition()
	evil.QueuePosition()
	sim.Tick(server, 0, good, evil)

	return
}

func TestCTF_FlagInit(t *testing.T) {
	_, _, _, obs := ctfSetup(t)

	if len(obs.Flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(obs.Flags))
	}
	for i, f := range obs.Flags {
		if f.Owner != -1 {
			t.Errorf("flag %d should have no owner, got CN %d", i, f.Owner)
		}
		if f.Dropped {
			t.Errorf("flag %d should not be dropped", i)
		}
	}
}

func TestCTF_TakeEnemyFlag(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	// Good player takes evil flag (flag 1 = team 2 = evil)
	evilFlagIdx := int32(1)
	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != good.CN {
		t.Errorf("expected good player (CN %d) to carry evil flag, owner is CN %d",
			good.CN, obs.Flags[evilFlagIdx].Owner)
	}
}

func TestCTF_DropFlag(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	evilFlagIdx := int32(1)
	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != good.CN {
		t.Fatal("good player should be carrying the flag")
	}

	good.DropFlag()
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != -1 {
		t.Error("flag should have no owner after drop")
	}
	if !obs.Flags[evilFlagIdx].Dropped {
		t.Error("flag should be marked as dropped")
	}
}

func TestCTF_ReturnDroppedFlag(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	evilFlagIdx := int32(1)

	// Good takes evil flag, then drops it
	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)
	good.DropFlag()
	sim.Tick(server, 33, good, evil)

	if !obs.Flags[evilFlagIdx].Dropped {
		t.Fatal("evil flag should be dropped")
	}

	// Evil player touches their own dropped flag to return it
	droppedVersion := obs.Flags[evilFlagIdx].Version
	evil.TakeFlag(evilFlagIdx, droppedVersion)
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != -1 {
		t.Error("evil flag should be returned (no owner)")
	}
	if obs.Flags[evilFlagIdx].Dropped {
		t.Error("evil flag should not be dropped after return")
	}
}

func TestCTF_CaptureFlag(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	evilFlagIdx := int32(1)
	goodFlagIdx := int32(0)

	// Good player takes the evil flag
	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != good.CN {
		t.Fatal("good player should be carrying enemy flag")
	}

	// Good player touches their OWN flag at base to score
	ownVersion := obs.Flags[goodFlagIdx].Version
	good.TakeFlag(goodFlagIdx, ownVersion)
	sim.Tick(server, 33, good, evil)

	// Good team (team 1, index 0) score should be 1
	if obs.TeamScore[0] != 1 {
		t.Errorf("good team score should be 1 after capture, got %d",
			obs.TeamScore[0])
	}

	// Enemy flag should be returned after capture
	if obs.Flags[evilFlagIdx].Owner != -1 {
		t.Error("enemy flag should be returned after capture")
	}
	if obs.Flags[evilFlagIdx].Dropped {
		t.Error("enemy flag should not be dropped after capture")
	}
}

func TestCTF_FlagDropOnDeath(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	evilFlagIdx := int32(1)

	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)

	if obs.Flags[evilFlagIdx].Owner != good.CN {
		t.Fatal("good player should be carrying the flag")
	}

	// Evil player kills good player — flag should drop
	for i := 0; i < 10; i++ {
		evil.ShootAt(good)
		sim.Tick(server, 33, good, evil)
	}

	if good.IsAlive() {
		t.Fatal("good player should be dead")
	}

	if obs.Flags[evilFlagIdx].Owner != -1 {
		t.Error("flag should have no carrier after flag holder dies")
	}
	if !obs.Flags[evilFlagIdx].Dropped {
		t.Error("flag should be marked as dropped after flag holder dies")
	}
}

func TestCTF_FlagResetAfterTimeout(t *testing.T) {
	server, good, evil, obs := ctfSetup(t)

	evilFlagIdx := int32(1)

	version := obs.Flags[evilFlagIdx].Version
	good.TakeFlag(evilFlagIdx, version)
	sim.Tick(server, 33, good, evil)
	good.DropFlag()
	sim.Tick(server, 33, good, evil)

	if !obs.Flags[evilFlagIdx].Dropped {
		t.Fatal("flag should be dropped")
	}

	// Advance past the 10-second reset deadline
	for i := 0; i < 350; i++ {
		sim.Tick(server, 33, good, evil)
	}

	if obs.Flags[evilFlagIdx].Dropped {
		t.Error("flag should not be dropped after 10s reset timeout")
	}
	if obs.Flags[evilFlagIdx].Owner != -1 {
		t.Error("flag should have no owner after reset")
	}
}

func TestCTF_RespawnDelay(t *testing.T) {
	server, good, evil, _ := ctfSetup(t)

	// Kill good player
	for i := 0; i < 10; i++ {
		evil.ShootAt(good)
		sim.Tick(server, 33, good, evil)
	}

	if good.IsAlive() {
		t.Fatal("good should be dead")
	}

	// Try to respawn immediately — should fail (5 second delay in CTF)
	good.QueueRespawn()
	sim.Tick(server, 33, good, evil)
	sim.Tick(server, 0, good, evil)

	if good.IsAlive() {
		t.Error("should not respawn before 5 second delay in CTF")
	}

	// Advance past the 5-second delay
	for i := 0; i < 160; i++ {
		sim.Tick(server, 33, good, evil)
	}

	good.QueueRespawn()
	sim.Tick(server, 33, good, evil)
	sim.Tick(server, 0, good, evil)

	if !good.IsAlive() {
		t.Error("should respawn after 5 second delay in CTF")
	}
}
