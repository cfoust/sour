package sim

import (
	"math/rand"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/weapon"
)

// Message constructors for tests

func TextMsg(text string) P.Message         { return P.Text{Text: text} }
func SuicideMsg() P.Message                 { return P.Suicide{} }
func SayTeamMsg(text string) P.Message      { return P.SayTeam{Text: text} }
func SpectatorMsg(cn int32, on bool) P.Message {
	return P.Spectator{Client: cn, Spectating: on}
}

// SayTeamType is exported so tests can type-assert on relayed team messages.
type SayTeamType = P.SayTeam

// InitFlags sends flag positions for CTF modes. Two flags are required:
// team 1 (good) and team 2 (evil).
func (c *Client) InitFlags(goodX, goodY, goodZ, evilX, evilY, evilZ float64) {
	c.Send(1, P.ClientInitFlags{
		Flags: []P.ClientFlagState{
			{Team: 1, Position: P.Vec{X: goodX, Y: goodY, Z: goodZ}},
			{Team: 2, Position: P.Vec{X: evilX, Y: evilY, Z: evilZ}},
		},
	})
}

// TakeFlag sends a flag pickup attempt.
func (c *Client) TakeFlag(flag int32, version int32) {
	c.Send(1, P.ClientTakeFlag{Flag: flag, Version: version})
}

// DropFlag sends a flag drop.
func (c *Client) DropFlag() {
	c.Send(1, P.TryDropFlag{})
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
