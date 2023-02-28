package game

import (
	"log"
	"time"

	"github.com/sauerbraten/timer"

	"github.com/cfoust/sour/pkg/server/geom"
	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
)

type FlagMode interface {
	NeedsMapInfo() bool
	FlagsInitPacket() []interface{}
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

func (m *handlesFlags) HandlePacket(p *Player, packetType nmc.ID, pkt *protocol.Packet) bool {
	switch packetType {
	case nmc.InitFlags:
		m.initFlags(pkt)

	case nmc.TouchFlag:
		m.touchFlag(p, pkt)

	case nmc.TryDropFlag:
		m.dropAllFlags(p)

	default:
		return false
	}

	return true
}

func (m *handlesFlags) initFlags(pkt *protocol.Packet) {
	numFlags, ok := pkt.GetInt()
	if !ok {
		log.Println("could not read number of flags from initflags packet (packet too short):", pkt)
		return
	}

	flags := []*flag{}
	for i := int32(0); i < numFlags; i++ {
		teamID, ok := pkt.GetInt()
		if !ok {
			log.Println("could not read flag team from initflags packet (packet too short):", pkt)
			return
		}

		spawnLocation, ok := pkt.GetVector()
		if !ok {
			log.Println("could not read flag spawn location from initflags packet (packet too short):", pkt)
			return
		}
		spawnLocation = spawnLocation.Mul(1 / geom.DMF)

		flags = append(flags, &flag{
			index:         i,
			team:          m.TeamByFlagTeamID(teamID),
			teamID:        teamID,
			spawnLocation: spawnLocation,
		})
	}

	if len(m.flags) != 0 {
		log.Println("got initflags packet, but flags are already initialized")
		return
	}

	ok = m.InitFlags(flags)
	if ok {
		m.flags = flags
	}
}

func (m *handlesFlags) touchFlag(p *Player, pkt *protocol.Packet) {
	if p.State != playerstate.Alive {
		return
	}

	i, ok := pkt.GetInt()
	if !ok {
		log.Println("could not read flag index from takeflag packet (packet too short):", pkt)
		return
	}
	if i < 0 || len(m.flags) <= int(i) {
		log.Printf("flag index %d from takeflag packet out of range [0..%d]", i, len(m.flags))
		return
	}
	f := m.flags[i]
	if f.carrier != nil {
		return
	}

	version, ok := pkt.GetInt()
	if !ok {
		log.Println("could not read flag version from takeflag packet (packet too short):", pkt)
		return
	}
	if f.version != version {
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

func (m *handlesFlags) FlagsInitPacket() []interface{} {
	q := []interface{}{}

	for _, f := range m.flags {
		if f.team == nil {
			log.Printf("flag with index '%d' has no team!", f.index)
			continue
		}
		q = append(q, f.team.Score)
	}

	q = append(q, len(m.flags))
	for _, f := range m.flags {
		if f == nil || f.team == nil {
			continue
		}

		var carrierCN int32 = -1
		if f.carrier != nil {
			carrierCN = int32(f.carrier.CN)
		}
		q = append(q, f.version, 0, carrierCN, 0)
		if f.carrier == nil {
			dropped := !f.dropTime.IsZero()
			q = append(q, dropped)
			if dropped {
				q = append(q, f.dropLocation.Mul(geom.DMF))
			}
		}
	}

	return q
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
