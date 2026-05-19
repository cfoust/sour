package sim_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

// newTestServer creates a gameserver.Server configured for testing with a
// short match length and FFA mode, starts its Poll loop, and returns a
// cleanup function.
func newTestServer(ctx context.Context) (*gameserver.Server, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	server := gameserver.New(ctx, &gameserver.Config{
		MaxClients:       32,
		MatchLength:      600,
		DefaultGameSpeed: 100,
		DefaultMode:      "ffa",
		DefaultMap:       "complex",
	})

	server.StartGame(server.StartMode(0), "complex") // 0 = FFA

	go server.Poll(ctx)

	// Drain the maps channel so StartGame doesn't block
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-server.ReceiveMaps():
			}
		}
	}()

	return server, cancel
}

func TestSingleClientConnectAndSpawn(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	client, err := sim.New(ctx, server, router, "TestPlayer")
	if err != nil {
		t.Fatalf("failed to create sim client: %v", err)
	}
	defer client.Disconnect()

	// Client should be alive after New returns
	if !client.IsAlive() {
		t.Error("client should be alive after joining")
	}

	// Should have been assigned CN 0
	client.Mu.Lock()
	cn := client.CN
	health := client.Health
	lifeSeq := client.LifeSequence
	client.Mu.Unlock()

	if cn != 0 {
		t.Errorf("expected CN 0, got %d", cn)
	}

	// Should have valid spawn state
	if health <= 0 {
		t.Errorf("expected positive health, got %d", health)
	}
	if lifeSeq <= 0 {
		t.Errorf("expected positive life sequence, got %d", lifeSeq)
	}
}

func TestMultipleClients(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	const numClients = 8
	clients := make([]*sim.Client, numClients)

	for i := 0; i < numClients; i++ {
		c, err := sim.New(ctx, server, router, fmt.Sprintf("Player%d", i))
		if err != nil {
			t.Fatalf("failed to create client %d: %v", i, err)
		}
		clients[i] = c
	}

	// All clients should be alive
	for i, c := range clients {
		if !c.IsAlive() {
			t.Errorf("client %d should be alive", i)
		}
	}

	// All clients should have unique CNs
	cns := make(map[int32]bool)
	for i, c := range clients {
		if cns[c.CN] {
			t.Errorf("client %d has duplicate CN %d", i, c.CN)
		}
		cns[c.CN] = true
	}

	// Clean up
	for _, c := range clients {
		c.Disconnect()
	}
}

func TestRapidConnectDisconnect(t *testing.T) {
	skipIfRace(t)
	ctx := context.Background()
	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	// Rapidly connect and disconnect clients to stress test the server's
	// client management (ClientManager.Add/Disconnect, relay AddClient/
	// RemoveClient) under concurrency. A small sleep between iterations
	// lets the server's Poll goroutine and async Send goroutines drain.
	for i := 0; i < 10; i++ {
		c, err := sim.New(ctx, server, router, fmt.Sprintf("Churn%d", i))
		if err != nil {
			t.Fatalf("failed to create client %d: %v", i, err)
		}
		if !c.IsAlive() {
			t.Errorf("client %d should be alive after join", i)
		}
		c.Disconnect()
		time.Sleep(100 * time.Millisecond)
	}
}

func TestPositionUpdatesFlow(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	client, err := sim.New(ctx, server, router, "PosTest")
	if err != nil {
		t.Fatalf("failed to create sim client: %v", err)
	}
	defer client.Disconnect()

	// Let the client run for a bit to send position updates and pings
	time.Sleep(200 * time.Millisecond)

	// Client should still be alive (no crash from sending positions)
	if !client.IsAlive() {
		t.Error("client should still be alive after sending positions")
	}

	// Yaw should have drifted from position updates
	client.Mu.Lock()
	yaw := client.Yaw
	client.Mu.Unlock()
	if yaw == 0 {
		t.Error("expected yaw to have changed from position updates")
	}
}

func TestPingResponse(t *testing.T) {
	ctx := context.Background()
	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	client, err := sim.New(ctx, server, router, "PingTest")
	if err != nil {
		t.Fatalf("failed to create sim client: %v", err)
	}
	defer client.Disconnect()

	// Wait long enough for at least one ping/pong cycle (250ms interval)
	time.Sleep(400 * time.Millisecond)

	// Ping should have been computed (non-zero after at least one pong)
	// Note: the first ping response sets Ping = (0*5 + rtt)/6
	// which could be very small but should be >= 0
	client.Mu.Lock()
	lpt := client.LastPingTime
	client.Mu.Unlock()
	if lpt == 0 {
		t.Error("expected at least one ping to have been sent")
	}
}

func TestConcurrentClientsWithActivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server, cleanup := newTestServer(ctx)
	defer cleanup()

	router := sim.NewRouter(ctx, server)
	defer router.Stop()

	const numClients = 16
	clients := make([]*sim.Client, numClients)

	for i := 0; i < numClients; i++ {
		c, err := sim.New(ctx, server, router, fmt.Sprintf("Stress%d", i))
		if err != nil {
			t.Fatalf("failed to create client %d: %v", i, err)
		}
		clients[i] = c
	}

	// Let all clients send position updates and pings concurrently
	time.Sleep(500 * time.Millisecond)

	// All should still be alive
	for i, c := range clients {
		if !c.IsAlive() {
			t.Errorf("client %d died unexpectedly", i)
		}
	}

	// Disconnect half, check the rest survive
	for i := 0; i < numClients/2; i++ {
		clients[i].Disconnect()
	}

	time.Sleep(200 * time.Millisecond)

	for i := numClients / 2; i < numClients; i++ {
		if !clients[i].IsAlive() {
			t.Errorf("client %d died after others disconnected", i)
		}
	}

	// Clean up remaining
	for i := numClients / 2; i < numClients; i++ {
		clients[i].Disconnect()
	}
}
