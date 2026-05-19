// Package sim provides a simulated Sauerbraten client for deterministic
// testing of the gameserver. No goroutines, no channels, no mutexes.
package sim

import (
	"math/rand"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

var nextSessionID uint32 = 1000

// Client simulates a Sauerbraten client's protocol-level behavior.
// It is pure data — the test drives it by calling methods and stepping
// the server.
type Client struct {
	CN        int32
	SessionID uint32
	Name      string

	State        playerstate.ID
	LifeSequence int32
	GunSelect    int32
	Frags        int32
	Deaths       int32
	Health       int32
	MaxHealth    int32
	Armour       int32

	X, Y, Z float64
	Yaw     float64
	Pitch   float64

	Ping  int32
	Alive bool

	outbox []gameserver.ServerPacket
	millis int32
}

// New creates a sim client and processes the server's Connect output
// (ServerInfo). The client queues an N_CONNECT response in its outbox.
func New(server *gameserver.Server, name string) (*Client, []gameserver.ServerPacket) {
	nextSessionID++
	sessionID := nextSessionID

	c := &Client{
		Name:      name,
		SessionID: sessionID,
		X:         512 + rand.Float64()*64,
		Y:         512 + rand.Float64()*64,
		Z:         530,
	}

	connectOutput := server.Connect(sessionID)
	c.HandleOutputs(connectOutput)

	return c, connectOutput
}

// Send queues a packet to be sent to the server.
func (c *Client) Send(channel uint8, messages ...P.Message) {
	c.outbox = append(c.outbox, gameserver.ServerPacket{
		Session:  c.SessionID,
		Channel:  channel,
		Messages: messages,
	})
}

// DrainOutbox returns all queued packets and clears the outbox.
func (c *Client) DrainOutbox() []gameserver.InputPacket {
	result := c.outbox
	c.outbox = nil
	return result
}

// IsAlive returns whether the client is currently alive.
func (c *Client) IsAlive() bool {
	return c.State == playerstate.Alive
}

// HandleOutputs processes server output packets destined for this client.
func (c *Client) HandleOutputs(outputs []gameserver.ServerPacket) {
	for _, pkt := range outputs {
		if pkt.Session == c.SessionID {
			c.HandleMessages(pkt.Messages)
		}
	}
}

// Disconnect removes the client from the server and returns output packets.
func (c *Client) Disconnect(server *gameserver.Server) []gameserver.ServerPacket {
	return server.Leave(c.SessionID)
}

// QueuePosition queues an N_POS packet.
func (c *Client) QueuePosition() {
	if c.State != playerstate.Alive {
		return
	}
	c.Yaw += 1.0
	if c.Yaw >= 360 {
		c.Yaw -= 360
	}
	c.Send(0, P.Pos{
		Client: c.CN,
		State: P.PhysicsState{
			State:        0,
			LifeSequence: c.LifeSequence,
			Yaw:          c.Yaw,
			Pitch:        c.Pitch,
			O:            P.Vec{X: c.X, Y: c.Y, Z: c.Z},
		},
	})
}

// QueuePing queues an N_PING packet.
func (c *Client) QueuePing() {
	c.millis += 33
	c.Send(1, P.Ping{Cmillis: c.millis})
}

// QueueRespawn queues an N_TRYSPAWN packet.
func (c *Client) QueueRespawn() {
	if c.State != playerstate.Dead {
		return
	}
	c.Send(1, P.TrySpawn{})
}
