package tests

import (
	"testing"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/mastermode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestPauseResume(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	// Grant master so the player can pause
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Role = role.Master

	// Pause
	c.Send(1, P.PauseGame{Paused: true, Client: c.CN})
	sim.Tick(server, 33, c)

	if !server.Clock.Paused() {
		t.Error("game should be paused")
	}

	// Resume
	c.Send(1, P.PauseGame{Paused: false, Client: c.CN})
	sim.Tick(server, 33, c)

	if server.Clock.Paused() {
		t.Error("game should be resumed")
	}
}

func TestPause_RequiresPrivilege(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Unprivileged")

	// Without master, pause should fail
	c.Send(1, P.PauseGame{Paused: true, Client: c.CN})
	sim.Tick(server, 33, c)

	if server.Clock.Paused() {
		t.Error("unprivileged player should not be able to pause")
	}

	// Grant master — now pause should work
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Role = role.Master

	c.Send(1, P.PauseGame{Paused: true, Client: c.CN})
	sim.Tick(server, 33, c)

	if !server.Clock.Paused() {
		t.Error("master should be able to pause")
	}
}

func TestMasterMode_SetByMaster(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Master")

	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Role = role.Master

	c.Send(1, P.MasterMode{MasterMode: int32(mastermode.Locked)})
	sim.Tick(server, 33, c)

	if server.MasterMode != mastermode.Locked {
		t.Errorf("expected locked master mode, got %d", server.MasterMode)
	}
}

func TestMasterMode_DeniedWithoutPrivilege(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Nobody")

	c.Send(1, P.MasterMode{MasterMode: int32(mastermode.Locked)})
	sim.Tick(server, 33, c)

	if server.MasterMode == mastermode.Locked {
		t.Error("unprivileged player should not change master mode")
	}
}

func TestKick(t *testing.T) {
	server := newTestServer()
	master := sim.ConnectAndSpawn(server, "Master")
	victim := sim.ConnectAndSpawnWith(server, "Victim", master)

	serverMaster := server.Clients.GetClientByID(master.SessionID)
	serverMaster.Role = role.Master

	master.Send(1, P.Kick{Victim: victim.CN, Reason: "testing"})
	sim.Tick(server, 33, master, victim)

	// Victim should be disconnected
	serverVictim := server.Clients.GetClientByID(victim.SessionID)
	if serverVictim != nil {
		t.Error("victim should be disconnected after kick")
	}
}

func TestKick_DeniedWithoutPrivilege(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Nobody")
	c2 := sim.ConnectAndSpawn(server, "Target")

	c1.Send(1, P.Kick{Victim: c2.CN, Reason: "no"})
	sim.Tick(server, 33, c1, c2)

	serverTarget := server.Clients.GetClientByID(c2.SessionID)
	if serverTarget == nil {
		t.Error("unprivileged player should not be able to kick")
	}
}

func TestMapVote_RequiresMaster(t *testing.T) {
	server := newTestServer()
	// Set MM to veto (required for map voting)
	server.SetPublicServer(mastermode.Veto)

	c := sim.ConnectAndSpawn(server, "Player")

	// Without master, map vote should fail
	c.Send(1, P.MapVote{Map: "dust2", Mode: int32(gamemode.Insta)})
	sim.Tick(server, 33, c)

	if server.Map == "dust2" {
		t.Error("unprivileged player should not force map change")
	}

	// Grant master
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Role = role.Master

	c.Send(1, P.MapVote{Map: "dust2", Mode: int32(gamemode.Insta)})
	sim.Tick(server, 33, c)
	sim.Tick(server, 0, c)

	if server.Map != "dust2" {
		t.Errorf("master should force map change, got %s", server.Map)
	}
	if server.GameMode.ID() != gamemode.Insta {
		t.Errorf("expected insta mode, got %s", server.GameMode.ID())
	}
}

func TestSetMaster_Relinquish(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	// Grant master directly (password-based claiming is not implemented)
	serverClient := server.Clients.GetClientByID(c.SessionID)
	serverClient.Role = role.Master

	if serverClient.Role != role.Master {
		t.Fatal("should have master")
	}

	// Relinquish via N_SETMASTER with Master=0
	c.Send(1, P.SetMaster{Client: c.CN, Master: 0, Password: ""})
	sim.Tick(server, 33, c)

	if serverClient.Role != role.None {
		t.Errorf("expected no role after relinquish, got %d", serverClient.Role)
	}
}

func TestMasterMode_LockedPreventsUnspec(t *testing.T) {
	server := newTestServer()
	master := sim.ConnectAndSpawn(server, "Master")
	player := sim.ConnectAndSpawnWith(server, "Player", master)

	serverMaster := server.Clients.GetClientByID(master.SessionID)
	serverMaster.Role = role.Master

	// Lock the server
	master.Send(1, P.MasterMode{MasterMode: int32(mastermode.Locked)})
	sim.Tick(server, 33, master, player)

	// Force player to spectator
	master.Send(1, sim.SpectatorMsg(player.CN, true))
	sim.Tick(server, 33, master, player)
	sim.Tick(server, 0, master, player)

	// Player tries to unspec themselves — should fail in locked mode
	player.Send(1, sim.SpectatorMsg(player.CN, false))
	sim.Tick(server, 33, master, player)

	serverPlayer := server.Clients.GetClientByID(player.SessionID)
	if serverPlayer.State != 5 { // playerstate.Spectator
		t.Error("locked mode should prevent unprivileged unspec")
	}
}
