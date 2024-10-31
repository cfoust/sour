package game

import (
	"sort"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
)

type TeamMode interface {
	Teams() map[string]*Team
	ForEachTeam(func(*Team))
	Join(*Player)
	ChangeTeam(*Player, string, bool)
	Leave(*Player)
	HandleFrag(fragger, victim *Player)
}

type teamMode struct {
	s                 Server
	teamsByName       map[string]*Team
	otherTeamsAllowed bool
	keepTeams         bool
}

var _ TeamMode = &teamMode{}

func withTeams(s Server, otherTeamsAllowed, keepTeams bool, teams ...*Team) *teamMode {
	teamsByName := map[string]*Team{}
	for _, team := range teams {
		teamsByName[team.Name] = team
	}
	return &teamMode{
		s:                 s,
		teamsByName:       teamsByName,
		otherTeamsAllowed: otherTeamsAllowed,
		keepTeams:         keepTeams,
	}
}

func (m *teamMode) selectTeam(p *Player) *Team {
	if m.keepTeams {
		for _, t := range m.teamsByName {
			if p.Team.Name == t.Name {
				return t
			}
		}
	}
	return m.selectWeakestTeam()
}

func (m *teamMode) selectWeakestTeam() *Team {
	teams := []*Team{}
	for _, team := range m.teamsByName {
		teams = append(teams, team)
	}

	sort.Sort(BySizeAndScore(teams))
	return teams[0]
}

func (m *teamMode) Join(p *Player) {
	team := m.selectTeam(p)
	team.Add(p)
	m.s.Broadcast(P.SetTeam{int32(p.CN), p.Team.Name, -1})
}

func (*teamMode) Leave(p *Player) {
	p.Team.Remove(p)
}

func (m *teamMode) HandleFrag(fragger, victim *Player) {
	victim.Die()
	if fragger.Team == victim.Team {
		fragger.Frags--
	} else {
		fragger.Frags++
	}
	m.s.Broadcast(P.Died{int32(victim.CN), int32(fragger.CN), fragger.Frags, fragger.Team.Frags})
}

func (m *teamMode) ForEachTeam(do func(t *Team)) {
	for _, team := range m.teamsByName {
		do(team)
	}
}

func (m *teamMode) Teams() map[string]*Team {
	return m.teamsByName
}

func (m *teamMode) ChangeTeam(p *Player, newTeamName string, forced bool) {
	var reason int32 = -1 // = none = silent
	if p.State != playerstate.Spectator {
		if forced {
			reason = 1 // = forced
		} else {
			reason = 0 // = voluntary
		}
	}

	setTeam := func(old, new *Team) {
		if p.State == playerstate.Alive {
			m.HandleFrag(p, p)
		}
		old.Remove(p)
		new.Add(p)
		m.s.Broadcast(P.SetTeam{int32(p.CN), p.Team.Name, reason})
	}

	// try existing teams first
	for name, team := range m.teamsByName {
		if name == newTeamName {
			// todo: check privileges and team balance
			setTeam(p.Team, team)
			return
		}
	}

	if m.otherTeamsAllowed {
		newTeam := NewTeam(newTeamName)
		m.teamsByName[newTeamName] = newTeam
		setTeam(p.Team, newTeam)
	}
}
