package game

import (
	"log"
	"time"

	"github.com/cfoust/sour/pkg/game/protocol"

	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/timer"
)

type FlagMode interface {
	NeedsMapInfo() bool
	FlagsInitPacket() protocol.Message
}

type flagMode interface {
	TeamMode
	CanSpawn(*Player) bool
	InitFlags([]*flag) bool
	TouchFlag(*Player, *flag)
	DropFlag(*Player, *flag)
	TeamByFlagTeamID(int32) *Team
}

type flag struct {
	index         int32
	team          *Team
	teamID        int32
	carrier       *Player
	version       int32
	spawnLocation *geom.Vector
	dropLocation  *geom.Vector
	dropTime      time.Time
	pendingReset  *timer.Timer
}

type handlesFlags struct {
	s Server
	flagMode
	flags []*flag
}

var (
	_ FlagMode  = &handlesFlags{}
	_ HasTimers = &handlesFlags{}
)

func handlingFlags(fm flagMode) *handlesFlags {
	return &handlesFlags{
		flagMode: fm,
	}
}

func (m *handlesFlags) NeedsMapInfo() bool {
	log.Println("flag init asked:", len(m.flags))
	return len(m.flags) == 0
}

func (m *handlesFlags) HandlePacket(p *Player, message protocol.Message) bool {
	switch message.Type() {
	case protocol.N_INITFLAGS:
		initFlags := message.(*protocol.ClientInitFlags)
		m.initFlags(initFlags)

	case protocol.N_TAKEFLAG:
		takeFlag := message.(*protocol.TakeFlag)
		m.touchFlag(p, takeFlag)

	case protocol.N_TRYDROPFLAG:
		m.dropAllFlags(p)

	default:
		return false
	}

	return true
}

func (m *handlesFlags) initFlags(message *protocol.ClientInitFlags) {
	flags := []*flag{}

	for i, clientFlag := range message.Flags {
		teamID := clientFlag.Team

		flags = append(flags, &flag{
			index:  int32(i),
			team:   m.TeamByFlagTeamID(int32(teamID)),
			teamID: int32(teamID),
			spawnLocation: geom.NewVector(
				clientFlag.Position.X,
				clientFlag.Position.Y,
				clientFlag.Position.Z,
			),
		})
	}

	if len(m.flags) != 0 {
		log.Println("got initflags packet, but flags are already initialized")
		return
	}

	ok := m.InitFlags(flags)
	if ok {
		m.flags = flags
	}
}

func (m *handlesFlags) touchFlag(p *Player, message *protocol.TakeFlag) {
	if p.State != playerstate.Alive {
		return
	}

	i := message.Flag

	if i < 0 || len(m.flags) <= int(i) {
		log.Printf("flag index %d from takeflag packet out of range [0..%d]", i, len(m.flags))
		return
	}
	f := m.flags[i]
	if f.carrier != nil {
		return
	}

	if f.version != int32(message.Version) {
		return
	}

	m.TouchFlag(p, f)
}

func (m *handlesFlags) dropAllFlags(p *Player) {
	for _, f := range m.flags {
		if f != nil && f.carrier == p {
			m.DropFlag(p, f)
		}
	}
}

func (m *handlesFlags) FlagsInitPacket() protocol.Message {
	message := protocol.ServerInitFlags{
		Scores: [2]protocol.TeamScore{
			{m.flags[0].team.Score},
			{m.flags[1].team.Score},
		},
	}

	for _, f := range m.flags {
		if f == nil || f.team == nil {
			continue
		}

		var carrierCN int32 = -1
		if f.carrier != nil {
			carrierCN = int32(f.carrier.CN)
		}

		flagState := protocol.FlagState{
			Version:   f.version,
			Spawn:     0,
			Owner:     int32(carrierCN),
			Invisible: false,
		}

		if f.carrier == nil {
			dropped := !f.dropTime.IsZero()
			flagState.Dropped = dropped
			if dropped {
				v := f.dropLocation
				flagState.Position.X = v.X()
				flagState.Position.Y = v.Y()
				flagState.Position.Z = v.Z()
			}
		}

		message.Flags = append(message.Flags, flagState)
	}

	return message
}

func (m *handlesFlags) HandleFrag(actor, victim *Player) {
	m.dropAllFlags(victim)
	m.flagMode.HandleFrag(actor, victim)
}

func (m *handlesFlags) Pause() {
	for _, f := range m.flags {
		if f == nil || f.pendingReset == nil || f.pendingReset.TimeLeft() == 0 {
			continue
		}
		f.pendingReset.Pause()
	}
}

func (m *handlesFlags) Resume() {
	for _, f := range m.flags {
		if f == nil || f.pendingReset == nil || f.pendingReset.TimeLeft() == 0 {
			continue
		}
		f.pendingReset.Start()
	}
}

func (m *handlesFlags) Leave(p *Player) {
	m.dropAllFlags(p)
	m.flagMode.Leave(p)
}

func (m *handlesFlags) CleanUp() {
	for _, f := range m.flags {
		if f == nil || f.pendingReset == nil {
			continue
		}
		f.pendingReset.Stop()
	}
}
