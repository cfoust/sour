package sim_test

import (
	"fmt"
	"testing"

	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestMultiClient_PositionRelay(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Player1")
	c2 := sim.ConnectAndSpawn(server, "Player2")

	c1.QueuePosition()
	outputs := sim.Tick(server, 33, c1, c2)

	receivedPos := false
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID && pkt.Channel == 0 {
			receivedPos = true
			break
		}
	}
	if !receivedPos {
		t.Error("c2 should receive c1's position update via relay")
	}
}

func TestMultiClient_ChatRelay(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Chatter")
	c2 := sim.ConnectAndSpawn(server, "Listener")

	c1.Send(1, sim.TextMsg("hello world"))
	outputs := sim.Tick(server, 33, c1, c2)

	receivedChat := false
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID && pkt.Channel == 1 {
			for _, msg := range pkt.Messages {
				if msg.Type() == 5 { // N_TEXT
					receivedChat = true
				}
			}
		}
	}
	if !receivedChat {
		t.Error("c2 should receive c1's chat message via relay")
	}
}

func TestMultiClient_TeamChat(t *testing.T) {
	server := newServerWithMode(gamemode.Teamplay, "complex")

	// Create 4 players to ensure at least 2 are on the same team
	clients := make([]*sim.Client, 4)
	for i := 0; i < 4; i++ {
		clients[i] = sim.ConnectAndSpawn(server, fmt.Sprintf("Player%d", i))
	}

	// Find two on the same team and one on a different team
	var sender, teammate, enemy *sim.Client
	for i, c := range clients {
		for j, o := range clients {
			if i != j && c.Team == o.Team {
				sender = c
				teammate = o
				// Find someone on a different team
				for _, e := range clients {
					if e.Team != c.Team {
						enemy = e
						break
					}
				}
				break
			}
		}
		if sender != nil {
			break
		}
	}

	if sender == nil || teammate == nil || enemy == nil {
		t.Skip("couldn't find the right team composition")
	}

	sender.Send(1, sim.SayTeamMsg("team only"))
	outputs := sim.Tick(server, 33, clients...)

	teammateGot := false
	enemyGot := false
	for _, pkt := range outputs {
		for _, msg := range pkt.Messages {
			if _, ok := msg.(sim.SayTeamType); ok {
				if pkt.Session == teammate.SessionID {
					teammateGot = true
				}
				if pkt.Session == enemy.SessionID {
					enemyGot = true
				}
			}
		}
	}
	if !teammateGot {
		t.Error("teammate should receive team chat")
	}
	if enemyGot {
		t.Error("enemy should NOT receive team chat")
	}
}

func TestDisconnect_OthersNotified(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Stayer")
	c2 := sim.ConnectAndSpawnWith(server, "Leaver", c1)

	if c1.KnownPlayers == nil || c1.KnownPlayers[c2.CN] != "Leaver" {
		t.Fatal("c1 should know about c2 before disconnect")
	}

	output := c2.Disconnect(server)
	c1.HandleOutputs(output)

	if _, exists := c1.KnownPlayers[c2.CN]; exists {
		t.Error("c1 should not know about c2 after disconnect")
	}
}

func TestDisconnect_CNReuse(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "First")
	cn1 := c1.CN
	c1.Disconnect(server)

	c2 := sim.ConnectAndSpawn(server, "Second")
	if c2.CN != cn1 {
		t.Errorf("expected CN %d to be reused, got %d", cn1, c2.CN)
	}
}
