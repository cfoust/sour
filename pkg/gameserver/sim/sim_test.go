package sim_test

import (
	"fmt"
	"testing"

	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func newTestServer() *gameserver.Server {
	server := gameserver.New(&gameserver.Config{
		MaxClients:       32,
		MatchLength:      600,
		DefaultGameSpeed: 100,
		DefaultMode:      "ffa",
		DefaultMap:       "complex",
	})
	server.StartGame(server.StartMode(0), "complex") // 0 = FFA
	return server
}

func TestSingleClientConnectAndSpawn(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "TestPlayer")

	if !c.IsAlive() {
		t.Error("client should be alive after spawning")
	}
	if c.CN != 0 {
		t.Errorf("expected CN 0, got %d", c.CN)
	}
	if c.Health <= 0 {
		t.Errorf("expected positive health, got %d", c.Health)
	}
	if c.LifeSequence <= 0 {
		t.Errorf("expected positive life sequence, got %d", c.LifeSequence)
	}
}

func TestMultipleClients(t *testing.T) {
	server := newTestServer()

	const numClients = 8
	clients := make([]*sim.Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("Player%d", i))
	}

	for i, c := range clients {
		if !c.IsAlive() {
			t.Errorf("client %d should be alive", i)
		}
	}

	cns := make(map[int32]bool)
	for i, c := range clients {
		if cns[c.CN] {
			t.Errorf("client %d has duplicate CN %d", i, c.CN)
		}
		cns[c.CN] = true
	}
}

func TestRapidConnectDisconnect(t *testing.T) {
	server := newTestServer()

	for i := 0; i < 20; i++ {
		c := sim.ConnectAndSpawn(server, fmt.Sprintf("Churn%d", i))
		if !c.IsAlive() {
			t.Errorf("client %d should be alive after join", i)
		}
		c.Disconnect(server)
	}
}

func TestPositionUpdatesFlow(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "PosTest")

	for i := 0; i < 10; i++ {
		c.QueuePosition()
		sim.Tick(server, 33, c)
	}

	if !c.IsAlive() {
		t.Error("client should still be alive after sending positions")
	}
	if c.Yaw == 0 {
		t.Error("expected yaw to have changed from position updates")
	}
}

func TestPingResponse(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "PingTest")

	c.QueuePing()
	sim.Tick(server, 33, c)

	// After the tick, the server sent N_PONG and the client responded
	// with N_CLIENTPING — step again to process
	sim.Tick(server, 33, c)

	// Ping should be computed
	if c.Ping < 0 {
		t.Errorf("expected non-negative ping, got %d", c.Ping)
	}
}

func TestConcurrentClientsWithActivity(t *testing.T) {
	server := newTestServer()

	const numClients = 16
	clients := make([]*sim.Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("Stress%d", i))
	}

	// Run several ticks with position updates
	for tick := 0; tick < 10; tick++ {
		for _, c := range clients {
			c.QueuePosition()
		}
		sim.Tick(server, 33, clients...)
	}

	for i, c := range clients {
		if !c.IsAlive() {
			t.Errorf("client %d died unexpectedly", i)
		}
	}

	// Disconnect half
	for i := 0; i < numClients/2; i++ {
		output := clients[i].Disconnect(server)
		// Dispatch disconnect notifications to remaining clients
		for j := numClients / 2; j < numClients; j++ {
			clients[j].HandleOutputs(output)
		}
	}

	sim.Tick(server, 33, clients[numClients/2:]...)

	for i := numClients / 2; i < numClients; i++ {
		if !clients[i].IsAlive() {
			t.Errorf("client %d died after others disconnected", i)
		}
	}
}
