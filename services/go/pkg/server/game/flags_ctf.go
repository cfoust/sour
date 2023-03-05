package game

import (
	"log"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/timer"
)

type ctf struct {
	*teamMode
	s           Server
	initialized bool
	good        *Team
	goodFlag    *flag
	evil        *Team
	evilFlag    *flag
}

var _ flagMode = &ctf{}

func newCTF(s Server, m *teamMode, good, evil *Team) *ctf {
	return &ctf{
		s:        s,
		teamMode: m,
		good:     good,
		evil:     evil,
	}
}

func (m *ctf) TeamByFlagTeamID(i int32) *Team {
	switch i {
	case 1:
		return m.good
	case 2:
		return m.evil
	default:
		return nil
	}
}

func (m *ctf) InitFlags(flags []*flag) bool {
	if len(flags) != 2 {
		log.Printf("expected 2 flags in CTF mode, but got %d", len(flags))
		return false
	}

	for _, f := range flags {
		switch f.team {
		case m.good:
			m.goodFlag = f
		case m.evil:
			m.evilFlag = f
		default:
			log.Printf("flag %v can't be matched to either good or evil", f)
			return false
		}
	}

	m.initialized = true

	return true
}

func (m *ctf) TouchFlag(p *Player, f *flag) {
	if p.Team != f.team {
		// player stealing enemy flag
		m.takeFlag(p, f)
	} else if !f.dropTime.IsZero() {
		// player touches her own, dropped flag
		f.pendingReset.Stop()
		m.returnFlag(f)
		m.s.Broadcast(P.ReturnFlag{int(p.CN), int(f.index), int(f.version)})
		return
	} else {
		// player touches her own flag at its base
		enemyFlag := m.evilFlag
		if p.Team == m.evil {
			enemyFlag = m.goodFlag
		}
		if enemyFlag == nil {
			log.Println("enemy flag is nil")
			return
		}
		if enemyFlag.carrier != p {
			return
		}

		m.returnFlag(enemyFlag)
		p.Flags++
		p.Team.Score++
		f.version++
		m.s.Broadcast(P.ScoreFlag{
			int(p.CN),
			int(enemyFlag.index),
			int(enemyFlag.version),
			int(f.index),
			int(f.version),
			0,
			int(f.teamID),
			p.Team.Score,
			p.Flags,
		})
		if p.Team.Score >= 10 {
			m.s.Intermission()
		}
	}
}

func (m *ctf) takeFlag(p *Player, f *flag) {
	// cancel reset
	if f.pendingReset != nil {
		f.pendingReset.Stop()
		f.pendingReset = nil
	}

	f.version++
	m.s.Broadcast(P.TakeFlag{
		int(p.CN),
		int(f.index),
		int(f.version),
	})
	f.carrier = p
}

func (m *ctf) returnFlag(f *flag) {
	f.dropTime = time.Time{}
	f.carrier = nil
	f.version++
}

func (m *ctf) DropFlag(p *Player, f *flag) {
	f.dropLocation = p.Position
	f.dropTime = time.Now()
	f.carrier = nil
	f.version++

	m.s.Broadcast(P.DropFlag{
		int(p.CN),
		int(f.index),
		int(f.version),
		P.Vec{
			f.dropLocation.X(),
			f.dropLocation.Y(),
			f.dropLocation.Z(),
		},
	})

	f.pendingReset = timer.AfterFunc(10*time.Second, func() {
		m.returnFlag(f)
		m.s.Broadcast(P.ResetFlag{
			int(f.index),
			int(f.version),
			0,
			int(f.teamID),
			f.team.Score,
		})
	})
	f.pendingReset.Start()
}

func (m *ctf) CanSpawn(p *Player) bool {
	return p.LastDeath.IsZero() || time.Since(p.LastDeath) > 5*time.Second
}
