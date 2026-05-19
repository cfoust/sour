package tests

import (
	"testing"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/sim"
)

func TestSwitchName(t *testing.T) {
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Original")
	c2 := sim.ConnectAndSpawnWith(server, "Observer", c1)

	c1.Send(1, P.SwitchName{Name: "NewName"})
	sim.Tick(server, 33, c1, c2)

	serverClient := server.Clients.GetClientByID(c1.SessionID)
	if serverClient.Name != "NewName" {
		t.Errorf("expected name NewName, got %s", serverClient.Name)
	}
}

func TestSwitchName_EmptyRejected(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	c.Send(1, P.SwitchName{Name: ""})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Name != "Player" {
		t.Errorf("empty name should be rejected, got %q", serverClient.Name)
	}
}

func TestSwitchName_TooLongRejected(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	longName := "ThisNameIsWayTooLongForSauerbraten"
	c.Send(1, P.SwitchName{Name: longName})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Name == longName {
		t.Error("name > 24 chars should be rejected")
	}
}

func TestSwitchModel(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	c.Send(1, P.SwitchModel{Model: 5})
	sim.Tick(server, 33, c)

	serverClient := server.Clients.GetClientByID(c.SessionID)
	if serverClient.Model != 5 {
		t.Errorf("expected model 5, got %d", serverClient.Model)
	}
}

func TestServerOnly_Rejected(t *testing.T) {
	// Clients should not be able to send server-only messages
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	numClientsBefore := server.Clients.GetNumClients()

	// Try to send a server-only message (N_DIED)
	c.Send(1, P.Died{Client: 0, Killer: 0, KillerFrags: 999, VictimFrags: 0})
	sim.Tick(server, 33, c)

	// The client should be disconnected for invalid message
	numClientsAfter := server.Clients.GetNumClients()
	if numClientsAfter >= numClientsBefore {
		t.Error("sending server-only message should disconnect the client")
	}
}

func TestInvalidChannel_Ignored(t *testing.T) {
	server := newTestServer()
	c := sim.ConnectAndSpawn(server, "Player")

	// Channel 0 before join is rejected, but after join should work for N_POS
	c.QueuePosition()
	sim.Tick(server, 33, c)

	if !c.IsAlive() {
		t.Error("valid position on channel 0 should not disconnect")
	}
}

func TestMultipleMessagesPerPacket(t *testing.T) {
	// A single packet can contain multiple messages
	server := newTestServer()
	c1 := sim.ConnectAndSpawn(server, "Sender")
	c2 := sim.ConnectAndSpawn(server, "Receiver")

	// Send multiple chat messages in one packet
	c1.Send(1,
		P.Text{Text: "msg1"},
		P.Text{Text: "msg2"},
		P.Text{Text: "msg3"},
	)
	outputs := sim.Tick(server, 33, c1, c2)

	// c2 should receive all 3 via relay
	textCount := 0
	for _, pkt := range outputs {
		if pkt.Session == c2.SessionID {
			for _, msg := range pkt.Messages {
				if _, ok := msg.(P.Text); ok {
					textCount++
				}
			}
		}
	}
	if textCount != 3 {
		t.Errorf("expected 3 text messages relayed, got %d", textCount)
	}
}
