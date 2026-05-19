package sim

import (
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

// updateLoop sends periodic position updates and pings, matching the C++
// client's c2sinfo function (client.cpp:1313).
//
// Position updates: every 33ms (~30fps), matching the C++ client's throttle
// (client.cpp:1316: if(totalmillis - lastupdate < 33 && !force) return;)
//
// Ping: every 250ms (client.cpp:1304: if(totalmillis-lastping>250))
func updateLoop(c *Client) {
	defer c.wg.Done()

	posTicker := time.NewTicker(33 * time.Millisecond)
	defer posTicker.Stop()

	pingTicker := time.NewTicker(250 * time.Millisecond)
	defer pingTicker.Stop()

	// Respawn ticker: when dead, try to respawn periodically.
	// The C++ client sends N_TRYSPAWN when the player presses a key.
	// We simulate this by trying every 100ms after death.
	respawnTicker := time.NewTicker(100 * time.Millisecond)
	defer respawnTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-posTicker.C:
			sendPosition(c)

		case <-pingTicker.C:
			sendPing(c)

		case <-respawnTicker.C:
			tryRespawn(c)
		}
	}
}

// sendPosition sends an N_POS packet matching the C++ client's sendposition
// function (client.cpp:1193-1253).
func sendPosition(c *Client) {
	c.Mu.Lock()
	if c.State != playerstate.Alive {
		c.Mu.Unlock()
		return
	}

	// Slowly drift position to simulate movement
	c.Yaw += 1.0
	if c.Yaw >= 360 {
		c.Yaw -= 360
	}

	pos := P.Pos{
		Client: c.CN,
		State: P.PhysicsState{
			State:        0, // PHYS_FLOAT
			LifeSequence: c.LifeSequence,
			Yaw:          c.Yaw,
			Roll:         0,
			Pitch:        c.Pitch,
			Move:         0,
			Strafe:       0,
			O: P.Vec{
				X: c.X,
				Y: c.Y,
				Z: c.Z,
			},
			Velocity: P.Vec{},
			Falling:  P.Vec{},
		},
	}
	c.Mu.Unlock()

	c.Send(0, pos)
}

// sendPing sends an N_PING matching C++ client's sendmessages
// (client.cpp:1304-1309).
func sendPing(c *Client) {
	millis := c.Totalmillis()
	c.Mu.Lock()
	c.LastPingTime = millis
	c.Mu.Unlock()
	c.Send(1, P.Ping{Cmillis: millis})
}

// tryRespawn sends N_TRYSPAWN if we're dead, simulating a player pressing
// fire to respawn.
func tryRespawn(c *Client) {
	c.Mu.Lock()
	dead := c.State == playerstate.Dead
	c.Mu.Unlock()
	if !dead {
		return
	}
	c.Send(1, P.TrySpawn{})
}
