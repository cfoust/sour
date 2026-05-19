package sim

import (
	"math/rand"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
)

// Shoot fires the client's selected weapon at a target position.
// This mirrors the C++ client's addmsg(N_SHOOT, ...) path.
func (c *Client) Shoot(targetX, targetY, targetZ float64) {
	c.Mu.Lock()
	if c.State != playerstate.Alive {
		c.Mu.Unlock()
		return
	}
	msg := P.Shoot{
		Id:  rand.Int31(),
		Gun: c.GunSelect,
		From: P.Vec{
			X: c.X,
			Y: c.Y,
			Z: c.Z,
		},
		To: P.Vec{
			X: targetX,
			Y: targetY,
			Z: targetZ,
		},
		Hits: nil,
	}
	c.Mu.Unlock()
	c.Send(1, msg)
}

// ShootAt fires at another Client, including a proper hit record with
// matching lifeSequence for server-side validation.
func (c *Client) ShootAt(target *Client) {
	c.Mu.Lock()
	if c.State != playerstate.Alive {
		c.Mu.Unlock()
		return
	}
	selfGun := c.GunSelect
	selfX, selfY, selfZ := c.X, c.Y, c.Z
	c.Mu.Unlock()

	target.Mu.Lock()
	if target.State != playerstate.Alive {
		target.Mu.Unlock()
		return
	}
	tCN := target.CN
	tLifeSeq := target.LifeSequence
	tX, tY, tZ := target.X, target.Y, target.Z
	target.Mu.Unlock()

	wpn := weapon.ByID(weapon.ID(selfGun))

	c.Send(1, P.Shoot{
		Id:  rand.Int31(),
		Gun: selfGun,
		From: P.Vec{
			X: selfX,
			Y: selfY,
			Z: selfZ,
		},
		To: P.Vec{
			X: tX,
			Y: tY,
			Z: tZ,
		},
		Hits: []P.Hit{
			{
				Target:       tCN,
				LifeSequence: tLifeSeq,
				Distance:     10.0,
				Rays:         wpn.Rays,
				Direction: P.Vec{
					X: tX - selfX,
					Y: tY - selfY,
					Z: tZ - selfZ,
				},
			},
		},
	})
}
