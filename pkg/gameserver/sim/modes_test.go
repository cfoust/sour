package sim_test

import (
	"testing"

	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestGameMode_FFA(t *testing.T) {
	server := newServerWithMode(gamemode.FFA, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.FFA {
		t.Errorf("expected FFA mode, got %s", c.GameMode)
	}
	if c.Map != "complex" {
		t.Errorf("expected map complex, got %s", c.Map)
	}
	if c.GunSelect != 6 { // pistol
		t.Errorf("expected pistol (6), got gun %d", c.GunSelect)
	}
	if c.Armour != 25 {
		t.Errorf("expected 25 armour in FFA, got %d", c.Armour)
	}
}

func TestGameMode_Insta(t *testing.T) {
	server := newServerWithMode(gamemode.Insta, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.Insta {
		t.Errorf("expected Insta mode, got %s", c.GameMode)
	}
	if c.Health != 1 || c.MaxHealth != 1 {
		t.Errorf("expected 1/1 health in insta, got %d/%d", c.Health, c.MaxHealth)
	}
	if c.Armour != 0 {
		t.Errorf("expected 0 armour in insta, got %d", c.Armour)
	}
}

func TestGameMode_Effic(t *testing.T) {
	server := newServerWithMode(gamemode.Effic, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.Effic {
		t.Errorf("expected Effic mode, got %s", c.GameMode)
	}
	if c.Armour != 100 {
		t.Errorf("expected 100 armour in effic, got %d", c.Armour)
	}
}

func TestGameMode_Tactics(t *testing.T) {
	server := newServerWithMode(gamemode.Tactics, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.Tactics {
		t.Errorf("expected Tactics mode, got %s", c.GameMode)
	}
	if c.Armour != 100 {
		t.Errorf("expected 100 armour in tactics, got %d", c.Armour)
	}
}

func TestGameMode_Teamplay(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")
	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	if c1.Team == "" || c1.Team == "none" {
		t.Error("player 1 should have a real team")
	}
	if c2.Team == "" || c2.Team == "none" {
		t.Error("player 2 should have a real team")
	}
}

func TestGameMode_InstaTeam(t *testing.T) {
	server := newServerWithMode(gamemode.InstaTeam, "complex")
	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	if c1.GameMode != gamemode.InstaTeam {
		t.Errorf("expected InstaTeam, got %s", c1.GameMode)
	}
	if c1.Health != 1 {
		t.Errorf("expected 1 health in insta team, got %d", c1.Health)
	}
	if c1.Team == "" || c2.Team == "" {
		t.Error("both players should have teams")
	}
}

func TestGameMode_EfficTeam(t *testing.T) {
	server := newServerWithMode(gamemode.EfficTeam, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.EfficTeam {
		t.Errorf("expected EfficTeam, got %s", c.GameMode)
	}
	if c.Armour != 100 {
		t.Errorf("expected 100 armour in effic team, got %d", c.Armour)
	}
}

func TestGameMode_CTF(t *testing.T) {
	server := newServerWithMode(gamemode.CTF, "complex")
	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawnWith(server, "Player2", c1)

	if c1.GameMode != gamemode.CTF {
		t.Errorf("expected CTF mode, got %s", c1.GameMode)
	}
	if c1.Team == "" || c1.Team == "none" {
		t.Errorf("player 1 should have a team in CTF, got %q", c1.Team)
	}
	if c2.Team == "" || c2.Team == "none" {
		t.Errorf("player 2 should have a team in CTF, got %q", c2.Team)
	}

	// Initialize flags — the first non-spectator client sends flag positions
	c1.InitFlags(100, 100, 512, 900, 900, 512)
	sim.Tick(server, 0, c1, c2)

	// Both players should be alive and playable
	if !c1.IsAlive() || !c2.IsAlive() {
		t.Error("both players should be alive in CTF")
	}
}

func TestGameMode_InstaCTF(t *testing.T) {
	server := newServerWithMode(gamemode.InstaCTF, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.InstaCTF {
		t.Errorf("expected InstaCTF, got %s", c.GameMode)
	}
	if c.Health != 1 {
		t.Errorf("expected 1 health in insta CTF, got %d", c.Health)
	}
}

func TestGameMode_EfficCTF(t *testing.T) {
	server := newServerWithMode(gamemode.EfficCTF, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	if c.GameMode != gamemode.EfficCTF {
		t.Errorf("expected EfficCTF, got %s", c.GameMode)
	}
	if c.Armour != 100 {
		t.Errorf("expected 100 armour in effic CTF, got %d", c.Armour)
	}
}
