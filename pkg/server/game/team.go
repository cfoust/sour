package game

import (
	"math/rand"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type Team struct {
	Name    string
	Frags   int32
	Score   int32
	Players map[*Player]struct{}
}

func NewTeam(name string) *Team {
	return &Team{
		Name:    name,
		Players: map[*Player]struct{}{},
	}
}

var NoTeam = &Team{Name: "none"}

// sorts teams ascending by size, then score
type BySizeAndScore []*Team

func (teams BySizeAndScore) Len() int {
	return len(teams)
}

func (teams BySizeAndScore) Swap(i, j int) {
	teams[i], teams[j] = teams[j], teams[i]
}

func (teams BySizeAndScore) Less(i, j int) bool {
	if len(teams[i].Players) != len(teams[j].Players) {
		return len(teams[i].Players) < len(teams[j].Players)
	}
	if teams[i].Score != teams[j].Score {
		return teams[i].Score < teams[j].Score
	}
	if teams[i].Frags != teams[j].Frags {
		return teams[i].Frags < teams[j].Frags
	}
	return rng.Intn(2) == 0
}

func (t *Team) Add(p *Player) {
	t.Players[p] = struct{}{}
	p.Team = t
}

func (t *Team) Remove(p *Player) {
	p.Team = NoTeam
	delete(t.Players, p)
}
