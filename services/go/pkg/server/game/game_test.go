package game

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/cfoust/sour/pkg/server/protocol/nmc"
)

var (
	_ Server = &mockServer{}
	//_ Player = &mockPlayer{}
)

type mockServer struct{}

func (s *mockServer) GameDuration() time.Duration { return 10 * time.Minute }

func (s *mockServer) Broadcast(nmc.ID, ...interface{}) {}

func (s *mockServer) Intermission() {}

func (s *mockServer) ForEachPlayer(func(*Player)) {}

func (s *mockServer) UniqueName(p *Player) string { return fmt.Sprintf("%v", p) }

func (s *mockServer) NumberOfPlayers() int { return 5 }

func TestCompetitiveMode(t *testing.T) {
	s := &mockServer{}

	timing := NewCompetitiveClock(s)

	var mode Mode = NewEfficCTF(s, true, timing)

	log.Printf("%T", mode)

	teamed, ok := mode.(TeamMode)
	if !ok {
		t.Error("effic ctf is not a team mode")
		return
	}

	_, ok = mode.(Clock)
	if !ok {
		t.Error("effic ctf is not a timed mode")
		return
	}

	p1, p2 := NewPlayer(1), NewPlayer(2)

	teamed.Join(&p1)

	if countPlayers(teamed) != 1 {
		t.Error("after one player joined, player count is not 1")
	}

	teamed.Join(&p2)

	if countPlayers(teamed) != 2 {
		t.Error("after two players joined, player count is not 2")
	}
}

func countPlayers(tm TeamMode) (sum int) {
	tm.ForEachTeam(func(t *Team) { sum += len(t.Players) })
	return
}
