package game

import (
	"fmt"
	"log"
	"testing"

	"github.com/cfoust/sour/pkg/game/protocol"
)

var _ Server = &mockServer{}

type mockServer struct {
	clock int64
}

func (s *mockServer) GameDuration() int64                    { return 600000 }
func (s *mockServer) GameClock() int64                       { return s.clock }
func (s *mockServer) Broadcast(messages ...protocol.Message) {}
func (s *mockServer) Message(message string)                 {}
func (s *mockServer) Intermission()                          {}
func (s *mockServer) ForEachPlayer(func(*Player))            {}
func (s *mockServer) UniqueName(p *Player) string            { return fmt.Sprintf("%v", p) }
func (s *mockServer) NumberOfPlayers() int                   { return 5 }

func TestCompetitiveMode(t *testing.T) {
	s := &mockServer{}

	var mode Mode = NewEfficCTF(s, true)

	log.Printf("%T", mode)

	teamed, ok := mode.(TeamMode)
	if !ok {
		t.Error("effic ctf is not a team mode")
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
