package game

import (
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
)

type Mode interface {
	HasTimers
	ID() gamemode.ID
	NeedsMapInfo() bool
	Leave(*Player)
	CanSpawn(*Player) bool
	Spawn(*PlayerState) // sets armour, ammo, and health
	HandleFrag(fragger, victim *Player)
}

type HandlesPackets interface {
	HandlePacket(*Player, P.Message) bool
}

type noSpawnWait struct{}

func (*noSpawnWait) CanSpawn(*Player) bool { return true }

type fiveSecondsSpawnWait struct{}

func (*fiveSecondsSpawnWait) CanSpawn(p *Player) bool {
	return p.LastDeath.IsZero() || time.Since(p.LastDeath) > 5*time.Second
}

// simple frag handling
type teamlessMode struct {
	s Server
}

func withoutTeams(s Server) *teamlessMode {
	return &teamlessMode{
		s: s,
	}
}

func (m *teamlessMode) HandleFrag(actor, victim *Player) {
	victim.Die()
	if actor == victim {
		actor.Frags--
	} else {
		actor.Frags++
	}
	m.s.Broadcast(P.Died{
		Client:      int32(victim.CN),
		Killer:      int32(actor.CN),
		KillerFrags: actor.Frags,
		VictimFrags: actor.Team.Frags,
	})
}

func (m *teamlessMode) Leave(*Player) {}

type HasTimers interface {
	Pause()
	Resume()
	Leave(*Player)
	CleanUp()
}

type noTimers struct{}

func (*noTimers) Pause() {}

func (*noTimers) Resume() {}

func (*noTimers) Leave(*Player) {}

func (*noTimers) CleanUp() {}
