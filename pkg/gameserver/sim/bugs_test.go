package sim_test

import (
	"fmt"
	"testing"

	"github.com/cfoust/sour/pkg/gameserver/sim"
)

// ---------------------------------------------------------------------------
// Bug 1: "Invincible" Players — LifeSequence desync
// ---------------------------------------------------------------------------

func TestInvinciblePlayer_ValidHit(t *testing.T) {
	server := newTestServer()
	attacker := sim.ConnectAndSpawn(server, "Attacker")
	victim := sim.ConnectAndSpawn(server, "Victim")

	healthBefore := victim.Health

	attacker.ShootAt(victim)
	sim.Tick(server, 33, attacker, victim)

	if victim.Health >= healthBefore {
		t.Errorf("BUG: victim health did not decrease after being shot "+
			"(before=%d, after=%d)", healthBefore, victim.Health)
	}
}

func TestInvinciblePlayer_OverTime(t *testing.T) {
	server := newTestServer()
	p1 := sim.ConnectAndSpawn(server, "Fighter1")
	p2 := sim.ConnectAndSpawn(server, "Fighter2")

	// Track p2's deaths across all rounds. Each round, p1 shoots p2
	// 5 times (Pistol does 35 damage, so 5 shots = 175 > 100 health).
	// If hits are landing, p2 should die at least once per round.
	deathsBefore := p2.Deaths

	for round := 0; round < 20; round++ {
		// Ensure both are alive
		for !p1.IsAlive() {
			p1.QueueRespawn()
			sim.Tick(server, 100, p1, p2)
			sim.Tick(server, 0, p1, p2)
		}
		for !p2.IsAlive() {
			p2.QueueRespawn()
			sim.Tick(server, 100, p1, p2)
			sim.Tick(server, 0, p1, p2)
		}

		for shot := 0; shot < 5; shot++ {
			p1.ShootAt(p2)
			sim.Tick(server, 33, p1, p2)
		}
	}

	kills := p2.Deaths - deathsBefore
	t.Logf("Results: p2 died %d times over 20 rounds (frags: p1=%d)",
		kills, p1.Frags)

	if kills == 0 {
		t.Error("BUG: zero kills over 20 rounds — hits are not landing")
	}
	if kills < 10 {
		t.Errorf("Only %d kills over 20 rounds — expected at least 10. "+
			"Hits are being silently dropped.", kills)
	}
}

// ---------------------------------------------------------------------------
// Bug 2: Relay packet duplication
// ---------------------------------------------------------------------------

func TestRelayNoDuplication(t *testing.T) {
	server := newTestServer()

	const numClients = 4
	clients := make([]*sim.Client, numClients)
	for i := 0; i < numClients; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("Relay%d", i))
	}

	// Have client 0 send 5 text messages
	for i := 0; i < 5; i++ {
		clients[0].Send(1, sim.TextMsg(fmt.Sprintf("msg%d", i)))
	}

	outputs := sim.Tick(server, 33, clients...)

	// Count how many text messages the non-sender clients received
	totalTexts := 0
	for _, pkt := range outputs {
		if pkt.Session == clients[0].SessionID {
			continue
		}
		for _, msg := range pkt.Messages {
			if msg.Type() == 5 { // N_TEXT
				totalTexts++
			}
		}
	}

	expectedTotal := 5 * (numClients - 1) // 15
	if totalTexts > expectedTotal {
		t.Errorf("BUG: relay sent %d text messages but expected %d (ratio: %.1fx). "+
			"Packet duplication bug present.",
			totalTexts, expectedTotal, float64(totalTexts)/float64(expectedTotal))
	} else {
		t.Logf("OK: received %d text messages (expected %d)", totalTexts, expectedTotal)
	}
}
