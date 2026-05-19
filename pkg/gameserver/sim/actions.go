package sim

import (
	"math/rand"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
)

// TextMsg creates a P.Text message (exported for tests).
func TextMsg(text string) P.Message {
	return P.Text{Text: text}
}

// Shoot fires at a target position.
func (c *Client) Shoot(targetX, targetY, targetZ float64) {
	if c.State != playerstate.Alive {
		return
	}
	c.Send(1, P.Shoot{
		Id:   rand.Int31(),
		Gun:  c.GunSelect,
		From: P.Vec{X: c.X, Y: c.Y, Z: c.Z},
		To:   P.Vec{X: targetX, Y: targetY, Z: targetZ},
	})
}

// ShootAt fires at another Client with a proper hit record.
func (c *Client) ShootAt(target *Client) {
	if c.State != playerstate.Alive || target.State != playerstate.Alive {
		return
	}
	wpn := weapon.ByID(weapon.ID(c.GunSelect))
	c.Send(1, P.Shoot{
		Id:   rand.Int31(),
		Gun:  c.GunSelect,
		From: P.Vec{X: c.X, Y: c.Y, Z: c.Z},
		To:   P.Vec{X: target.X, Y: target.Y, Z: target.Z},
		Hits: []P.Hit{
			{
				Target:       target.CN,
				LifeSequence: target.LifeSequence,
				Distance:     10.0,
				Rays:         wpn.Rays,
				Direction: P.Vec{
					X: target.X - c.X,
					Y: target.Y - c.Y,
					Z: target.Z - c.Z,
				},
			},
		},
	})
}
