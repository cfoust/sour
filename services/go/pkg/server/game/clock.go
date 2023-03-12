package game

import (
	"fmt"
	"log"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/timer"
)

type Clock interface {
	Start()
	Pause(*Player)
	Paused() bool
	Resume(*Player)
	Stop()
	Ended() bool
	TimeLeft() time.Duration
	SetTimeLeft(time.Duration)
	Leave(*Player)
	CleanUp()
}

type casualClock struct {
	s          Server
	t          *timer.Timer
	modeTimers HasTimers
}

var _ Clock = &casualClock{}

func NewCasualClock(s Server, m HasTimers) *casualClock {
	return &casualClock{
		s:          s,
		t:          timer.AfterFunc(s.GameDuration(), s.Intermission),
		modeTimers: m,
	}
}

func (c *casualClock) Start() {
	go c.t.Start()
	c.s.Broadcast(P.TimeUp{
		Remaining: int32(c.t.TimeLeft() / time.Second),
	})
}

func (c *casualClock) Pause(p *Player) {
	if c.t.Paused() {
		return
	}
	var cn int32 = -1
	if p != nil {
		cn = int32(p.CN)
	}
	c.s.Broadcast(P.PauseGame{
		Paused: true,
		Client: cn,
	})
	c.t.Pause()
	c.modeTimers.Pause()
}

func (c *casualClock) Paused() bool {
	return c.t.Paused()
}

func (c *casualClock) Resume(p *Player) {
	if !c.t.Paused() {
		return
	}
	var cn int32 = -1
	if p != nil {
		cn = int32(p.CN)
	}
	c.s.Broadcast(P.PauseGame{
		Paused: false,
		Client: cn,
	})
	c.t.Start()
	c.modeTimers.Resume()
}

func (c *casualClock) Leave(*Player) {}

func (c *casualClock) Stop() {
	c.s.Broadcast(P.TimeUp{0})
	c.t.Stop()
}

func (c *casualClock) Ended() bool {
	return c.t.TimeLeft() <= 0
}

func (c *casualClock) TimeLeft() time.Duration {
	return c.t.TimeLeft()
}

func (c *casualClock) SetTimeLeft(d time.Duration) {
	if !c.t.SetTimeLeft(d) {
		log.Println("game timer had already expired")
	}
	c.s.Broadcast(P.TimeUp{int32(d / time.Second)})
}

func (c *casualClock) CleanUp() {
	c.t.Stop()
	c.modeTimers.CleanUp()
}

type Competitive interface {
	Clock
	Spawned(*Player)
}

type competitiveClock struct {
	*casualClock
	pendingResumeActions []*time.Timer
	mapLoadPending       map[*Player]struct{}
}

var (
	_ Clock       = &competitiveClock{}
	_ Competitive = &competitiveClock{}
)

func NewCompetitiveClock(s Server, m HasTimers) *competitiveClock {
	return &competitiveClock{
		casualClock:    NewCasualClock(s, m),
		mapLoadPending: map[*Player]struct{}{},
	}
}

func (c *competitiveClock) Start() {
	c.casualClock.Start()
	c.s.ForEachPlayer(func(p *Player) {
		if p.State != playerstate.Spectator {
			c.mapLoadPending[p] = struct{}{}
		}
	})
	if len(c.mapLoadPending) > 0 {
		c.s.Message("waiting for all players to load the map")
		c.Pause(nil)
	}
}

func (c *competitiveClock) Spawned(p *Player) {
	delete(c.mapLoadPending, p)
	if len(c.mapLoadPending) == 0 {
		c.s.Message("all players spawned, starting game")
		c.Resume(nil)
	}
}

func (c *competitiveClock) Pause(p *Player) {
	if !c.t.Paused() {
		c.casualClock.Pause(p)
	} else if len(c.pendingResumeActions) > 0 {
		// a resume is pending, cancel it
		c.Resume(p)
	}
}

func (c *competitiveClock) Resume(p *Player) {
	if len(c.pendingResumeActions) > 0 {
		for _, action := range c.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		c.pendingResumeActions = nil
		c.s.Message("resuming aborted")
		return
	}

	if p != nil {
		c.s.Message(fmt.Sprintf("%s wants to resume the game", c.s.UniqueName(p)))
	}
	c.s.Message("resuming game in 3 seconds")
	c.pendingResumeActions = []*time.Timer{
		time.AfterFunc(1*time.Second, func() { c.s.Message("resuming game in 2 seconds") }),
		time.AfterFunc(2*time.Second, func() { c.s.Message("resuming game in 1 second") }),
		time.AfterFunc(3*time.Second, func() {
			c.casualClock.Resume(p)
			c.pendingResumeActions = nil
		}),
	}
}

func (c *competitiveClock) Leave(p *Player) {
	if p.State != playerstate.Spectator && !c.Ended() {
		c.s.Message("a player left the game")
		c.Pause(nil)
	}
}

func (c *competitiveClock) CleanUp() {
	if len(c.pendingResumeActions) > 0 {
		for _, action := range c.pendingResumeActions {
			if action != nil {
				action.Stop()
			}
		}
		c.pendingResumeActions = nil
	}
	c.casualClock.CleanUp()
}

func (c *competitiveClock) ToCasual() *casualClock { return c.casualClock }
