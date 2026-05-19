package sim

import (
	"github.com/cfoust/sour/pkg/gameserver"
)

// Tick collects all clients' outboxes, steps the server, and dispatches
// the output back to the clients. This is the core test helper.
func Tick(server *gameserver.Server, dtMs int64, clients ...*Client) []gameserver.ServerPacket {
	var inputs []gameserver.InputPacket
	for _, c := range clients {
		inputs = append(inputs, c.DrainOutbox()...)
	}
	outputs := server.Step(dtMs, inputs)
	for _, c := range clients {
		c.HandleOutputs(outputs)
	}
	return outputs
}

// ConnectAndSpawn creates a client, connects it, and steps until it's alive.
// Panics if the client can't spawn within a few steps.
func ConnectAndSpawn(server *gameserver.Server, name string) *Client {
	c, _ := New(server, name)

	// Step to process the N_CONNECT from the client's outbox
	Tick(server, 0, c)

	// The server should have sent Welcome + SpawnState, and the client
	// should have responded with SpawnRequest. Step again to process it.
	Tick(server, 0, c)

	// One more step in case the spawn confirmation needs processing
	if !c.IsAlive() {
		Tick(server, 0, c)
	}

	if !c.IsAlive() {
		panic("client " + name + " failed to spawn")
	}

	return c
}

// ConnectAndSpawnWith creates a client and steps it with the given existing
// clients so they all see each other's join/init messages.
func ConnectAndSpawnWith(server *gameserver.Server, name string, others ...*Client) *Client {
	c, connectOutput := New(server, name)
	for _, o := range others {
		o.HandleOutputs(connectOutput)
	}

	all := append([]*Client{c}, others...)
	Tick(server, 0, all...)
	Tick(server, 0, all...)
	if !c.IsAlive() {
		Tick(server, 0, all...)
	}

	if !c.IsAlive() {
		panic("client " + name + " failed to spawn")
	}

	return c
}
