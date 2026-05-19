package tests

import (
	"fmt"
	"testing"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestTeam_FriendlyFireDecreasesFrags(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	// Connect enough players so we get teammates
	clients := make([]*sim.Client, 4)
	for i := 0; i < 4; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("P%d", i))
	}

	// Find two on the same team
	var attacker, teammate *sim.Client
	for i, c := range clients {
		for j, o := range clients {
			if i != j && c.Team == o.Team {
				attacker = c
				teammate = o
				break
			}
		}
		if attacker != nil {
			break
		}
	}
	if attacker == nil {
		t.Skip("couldn't find two teammates")
	}

	fragsBefore := attacker.Frags

	// Kill teammate
	for i := 0; i < 5; i++ {
		attacker.ShootAt(teammate)
		sim.Tick(server, 33, clients...)
	}

	if teammate.IsAlive() {
		t.Fatal("teammate should be dead")
	}

	// Teamkill should decrement frags
	if attacker.Frags >= fragsBefore {
		t.Errorf("teamkill should decrement frags (before=%d, after=%d)",
			fragsBefore, attacker.Frags)
	}
}

func TestTeam_EnemyKillIncreasesFrags(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	clients := make([]*sim.Client, 4)
	for i := 0; i < 4; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("P%d", i))
	}

	// Find two on different teams
	var attacker, enemy *sim.Client
	for i, c := range clients {
		for j, o := range clients {
			if i != j && c.Team != o.Team {
				attacker = c
				enemy = o
				break
			}
		}
		if attacker != nil {
			break
		}
	}
	if attacker == nil {
		t.Skip("couldn't find two enemies")
	}

	fragsBefore := attacker.Frags

	for i := 0; i < 5; i++ {
		attacker.ShootAt(enemy)
		sim.Tick(server, 33, clients...)
	}

	if enemy.IsAlive() {
		t.Fatal("enemy should be dead")
	}

	if attacker.Frags <= fragsBefore {
		t.Errorf("enemy kill should increment frags (before=%d, after=%d)",
			fragsBefore, attacker.Frags)
	}
}

func TestTeam_Balancing(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	// Add 6 players — should get 3 per team
	clients := make([]*sim.Client, 6)
	for i := 0; i < 6; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("P%d", i))
	}

	goodCount := 0
	evilCount := 0
	for _, c := range clients {
		switch c.Team {
		case "good":
			goodCount++
		case "evil":
			evilCount++
		}
	}

	// Teams should be balanced (3-3 or at worst 4-2)
	diff := goodCount - evilCount
	if diff < 0 {
		diff = -diff
	}
	if diff > 2 {
		t.Errorf("teams are unbalanced: good=%d evil=%d", goodCount, evilCount)
	}
	t.Logf("Team balance: good=%d evil=%d", goodCount, evilCount)
}

func TestTeam_ChangeTeam(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	originalTeam := c1.Team
	targetTeam := "evil"
	if originalTeam == "evil" {
		targetTeam = "good"
	}

	c1.Send(1, P.SwitchTeam{Team: targetTeam})
	sim.Tick(server, 33, c1, c2)

	serverClient := server.Clients.GetClientByID(c1.SessionID)
	if serverClient.Team.Name == originalTeam {
		t.Errorf("team should have changed from %s", originalTeam)
	}
	if serverClient.Team.Name != targetTeam {
		t.Errorf("expected team %s, got %s", targetTeam, serverClient.Team.Name)
	}
}

func TestTeam_ChangeTeamKillsIfAlive(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	c := sim.ConnectAndSpawn(server, "Player")

	if !c.IsAlive() {
		t.Fatal("should be alive")
	}

	targetTeam := "evil"
	if c.Team == "evil" {
		targetTeam = "good"
	}

	c.Send(1, P.SwitchTeam{Team: targetTeam})
	sim.Tick(server, 33, c)

	// Switching teams while alive triggers a self-frag
	if c.Frags >= 0 {
		t.Errorf("switching teams while alive should decrement frags, got %d",
			c.Frags)
	}
}

func TestTeam_OtherTeamsAllowed(t *testing.T) {
	// Teamplay has otherTeamsAllowed=true, meaning clients can
	// switch to custom team names beyond "good" and "evil"
	server := newServerWithMode(gamemode.Teamplay, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	c.Send(1, P.SwitchTeam{Team: "custom"})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Team.Name != "custom" {
		t.Errorf("teamplay should allow custom team names, got %q",
			serverClient.Team.Name)
	}
}

func TestTeam_OtherTeamsAllowedInAllTeamModes(t *testing.T) {
	// All team modes in Sauerbraten allow custom team names (you can
	// type /team custom in any team mode). Verify this works in EfficTeam too.
	server := newServerWithMode(gamemode.EfficTeam, "complex")
	c := sim.ConnectAndSpawn(server, "Player")

	c.Send(1, P.SwitchTeam{Team: "custom"})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Team.Name != "custom" {
		t.Errorf("team modes should allow custom team names, got %q",
			serverClient.Team.Name)
	}
}

func TestTeam_KeepTeamsAcrossMapChange(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")
	server.KeepTeams = true

	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	team1Before := c1.Team
	team2Before := c2.Team

	// Change map — teams should persist
	server.ChangeMap(int32(gamemode.Teamplay), "dust2")
	sim.Tick(server, 0, c1, c2)
	sim.Tick(server, 0, c1, c2)

	if c1.Team != team1Before {
		t.Errorf("player 1 team should persist: was %q, now %q",
			team1Before, c1.Team)
	}
	if c2.Team != team2Before {
		t.Errorf("player 2 team should persist: was %q, now %q",
			team2Before, c2.Team)
	}
}

func TestTeam_ShuffleTeamsWithoutKeep(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")
	server.KeepTeams = false

	// Add enough players that shuffling is visible
	clients := make([]*sim.Client, 6)
	for i := 0; i < 6; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("P%d", i))
	}

	// Change map — teams get reassigned by balance
	server.ChangeMap(int32(gamemode.Teamplay), "dust2")
	for i := 0; i < 3; i++ {
		sim.Tick(server, 0, clients...)
	}

	// Just verify everyone still has a team (the actual assignment
	// may or may not change depending on the balance algorithm)
	for i, c := range clients {
		serverClient := server.Clients.GetClientByID(c.SessionID)
		if serverClient.Team.Name != "good" && serverClient.Team.Name != "evil" {
			t.Errorf("player %d should have a valid team after map change, got %q",
				i, serverClient.Team.Name)
		}
	}
}

func TestCTF_ScoreLimitIntermission(t *testing.T) {
	server := newServerWithMode(gamemode.CTF, "complex")
	c1 := sim.ConnectAndSpawn(server, "Scorer")

	c1.InitFlags(100, 100, 512, 900, 900, 512)
	sim.Tick(server, 0, c1)

	c2 := sim.ConnectAndSpawnWith(server, "Other", c1)
	c1.QueuePosition()
	c2.QueuePosition()
	sim.Tick(server, 0, c1, c2)

	// Find which player is on "good" team (team 1, flag 0)
	scorer := c1
	if c1.Team != "good" {
		scorer = c2
	}

	obs := c2
	if len(c2.Flags) == 0 {
		obs = c1
	}

	// Score 10 flags to trigger intermission
	for i := 0; i < 10; i++ {
		evilFlagVersion := obs.Flags[1].Version
		scorer.TakeFlag(1, evilFlagVersion)
		sim.Tick(server, 33, c1, c2)

		goodFlagVersion := obs.Flags[0].Version
		scorer.TakeFlag(0, goodFlagVersion)
		sim.Tick(server, 33, c1, c2)
	}

	// Check score reached 10
	if obs.TeamScore[0] < 10 {
		t.Errorf("good team should have scored 10, got %d", obs.TeamScore[0])
	}
}
