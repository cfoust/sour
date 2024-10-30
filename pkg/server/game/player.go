package game

import (
	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/protocol/weapon"
)

type Player struct {
	CN       uint32
	Name     string
	Team     *Team
	Model    int32
	Position *geom.Vector
	PlayerState
}

func NewPlayer(cn uint32) Player {
	return Player{
		CN:          cn,
		Team:        NoTeam,
		PlayerState: NewPlayerState(),
	}
}

func (p *Player) ApplyDamage(attacker *Player, damage int32, weapon weapon.ID, direction *geom.Vector) {
	p.PlayerState.applyDamage(damage)
	if attacker != p && attacker.Team != p.Team {
		attacker.Damage += damage
	}

	// TODO quad?
}

func (p *Player) Reset() {
	// keep the CN, so low CNs can be reused
	p.Name = ""
	p.Team = NoTeam
	p.Model = -1
	p.Position = nil
	p.PlayerState.Reset()
}
