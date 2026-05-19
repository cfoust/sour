// Package sim provides a simulated Sauerbraten client that faithfully
// replicates the protocol-level state machine from the C++ client
// (game/src/fpsgame/client.cpp). It is designed for integration testing of
// the gameserver package under realistic multi-player conditions.
package sim

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

var nextSessionID uint32 = 1000

// Client faithfully simulates a Sauerbraten client's protocol-level behavior.
// It replicates the state machine from game/src/fpsgame/client.cpp: connection
// handshake, spawn confirmation, position updates, ping keepalive, and
// death/respawn cycling.
//
// Fields are accessed from multiple goroutines (router, updateLoop, test)
// and are protected by Mu.
type Client struct {
	Mu sync.Mutex

	// Server-assigned state (set during handshake)
	CN        int32
	SessionID uint32
	Name      string

	// Protocol state mirroring the C++ client
	State        playerstate.ID
	LifeSequence int32
	GunSelect    int32
	Frags        int32
	Deaths       int32
	Health       int32
	MaxHealth    int32
	Armour       int32

	// Position state
	X, Y, Z float64
	Yaw     float64
	Pitch   float64

	// Ping tracking (mirrors C++ client: lastping, player1->ping)
	LastPingTime int32 // the cmillis we last sent
	Ping         int32 // computed running average

	// Internal
	server   *gameserver.Server
	router   *Router
	incoming chan<- gameserver.ServerPacket // send packets to the server
	cancel   context.CancelFunc
	ctx      context.Context
	wg       sync.WaitGroup

	// joined is closed when the client has fully joined (received Welcome)
	joined chan struct{}
	// alive is closed each time the client becomes alive; recreated on death
	alive     chan struct{}
	aliveMu   sync.Mutex
	connected int32 // atomic: 1 once N_WELCOME received

	// millis tracks a simulated client clock, advancing in real time.
	// Mirrors C++ totalmillis.
	startTime time.Time
}

// Totalmillis returns the simulated client time in milliseconds, matching the
// C++ client's totalmillis variable.
func (c *Client) Totalmillis() int32 {
	return int32(time.Since(c.startTime).Milliseconds())
}

// Send sends a packet to the server on the given channel. The send is
// non-blocking to avoid deadlocks: the router goroutine calls HandleMessage
// which may call Send, while the server's Poll goroutine may simultaneously
// be trying to write to the outgoing channel. This mirrors the real C++
// client, which queues messages in a buffer (client.cpp:1072) and flushes
// them asynchronously in sendmessages().
func (c *Client) Send(channel uint8, messages ...P.Message) {
	go func() {
		select {
		case c.incoming <- gameserver.ServerPacket{
			Session:  c.SessionID,
			Channel:  channel,
			Messages: messages,
		}:
		case <-c.ctx.Done():
		}
	}()
}

// Disconnect cleanly disconnects the simulated client from the server.
func (c *Client) Disconnect() {
	c.cancel()
	if c.router != nil {
		c.router.Remove(c)
	}
	c.server.Leave(c.SessionID)
	c.wg.Wait()
}

// IsAlive returns whether the client is currently alive in game.
func (c *Client) IsAlive() bool {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	return c.State == playerstate.Alive
}

// WaitAlive blocks until the client is alive or ctx is done.
func (c *Client) WaitAlive(ctx context.Context) error {
	c.aliveMu.Lock()
	ch := c.alive
	c.aliveMu.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// New creates a SimClient using the given router for message dispatch.
// This is the primary way to create SimClients in tests. It performs
// the full connection handshake and blocks until the client has joined
// the game (received Welcome + SpawnState and confirmed spawn).
func New(ctx context.Context, server *gameserver.Server, router *Router, name string) (*Client, error) {
	sessionID := atomic.AddUint32(&nextSessionID, 1)

	ctx, cancel := context.WithCancel(ctx)
	c := &Client{
		Name:      name,
		SessionID: sessionID,
		server:    server,
		router:    router,
		incoming:  server.Incoming(),
		cancel:    cancel,
		ctx:       ctx,
		joined:    make(chan struct{}),
		alive:     make(chan struct{}),
		startTime: time.Now(),
		X:         512 + rand.Float64()*64,
		Y:         512 + rand.Float64()*64,
		Z:         530,
	}

	// Register with the router BEFORE connecting to the server, so we
	// don't miss the ServerInfo packet.
	router.Add(c)

	// Register with the server (this sends ServerInfo to us via outgoing)
	_, connectedCh := server.Connect(sessionID)
	if connectedCh == nil {
		cancel()
		router.Remove(c)
		return nil, fmt.Errorf("server rejected connection for session %d", sessionID)
	}

	// Wait for server to process our N_CONNECT (triggered by ServerInfo handler)
	select {
	case <-connectedCh:
	case <-ctx.Done():
		cancel()
		router.Remove(c)
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		cancel()
		router.Remove(c)
		return nil, fmt.Errorf("timed out waiting for server join")
	}

	// Wait for welcome sequence to complete
	select {
	case <-c.joined:
	case <-ctx.Done():
		cancel()
		router.Remove(c)
		return nil, ctx.Err()
	case <-time.After(5 * time.Second):
		cancel()
		router.Remove(c)
		return nil, fmt.Errorf("timed out waiting for welcome")
	}

	// Start periodic updates
	c.wg.Add(1)
	go updateLoop(c)

	return c, nil
}
