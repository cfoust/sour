package game

import (
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
)

type Mode interface {
	HasTimers
	ID() gamemode.ID
	NeedsMapInfo() bool
	Leave(*Player)
	CanSpawn(clock int64, p *Player) bool
	Spawn(*PlayerState) // sets armour, ammo, and health
	HandleFrag(clock int64, fragger, victim *Player)
}

type HandlesPackets interface {
	HandlePacket(*Player, P.Message) bool
}

// Tickable is implemented by game modes that need periodic updates
// (e.g., pickup respawns, flag resets).
type Tickable interface {
	Tick(clock int64)
}

type noSpawnWait struct{}

func (*noSpawnWait) CanSpawn(clock int64, p *Player) bool { return true }

type fiveSecondsSpawnWait struct{}

func (*fiveSecondsSpawnWait) CanSpawn(clock int64, p *Player) bool {
	return p.LastDeath == 0 || clock-p.LastDeath > 5000
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

func (m *teamlessMode) HandleFrag(clock int64, actor, victim *Player) {
	victim.Die(clock)
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
