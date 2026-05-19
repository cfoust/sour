package game

import (
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/deadline"
)

type Clock interface {
	Start(clock int64)
	Tick(clock int64)
	Pause(clock int64, p *Player)
	Paused() bool
	Resume(clock int64, p *Player)
	Stop()
	Ended() bool
	TimeLeft(clock int64) time.Duration
	SetTimeLeft(clock int64, d time.Duration)
	Leave(p *Player)
	CleanUp()
}

type casualClock struct {
	s          Server
	deadline   deadline.Deadline
	paused     bool
	modeTimers HasTimers
}

var _ Clock = &casualClock{}

func NewCasualClock(s Server, m HasTimers) *casualClock {
	return &casualClock{
		s:          s,
		modeTimers: m,
	}
}

func (c *casualClock) Start(clock int64) {
	c.deadline.Set(clock, c.s.GameDuration())
	c.s.Broadcast(P.TimeUp{
		Remaining: int32(c.deadline.TimeLeftMs(clock) / 1000),
	})
}

func (c *casualClock) Tick(clock int64) {
	if c.paused {
		return
	}
	if c.deadline.Expired(clock) {
		c.deadline.Stop()
		c.s.Intermission()
	}
}

func (c *casualClock) Pause(clock int64, p *Player) {
	if c.paused {
		return
	}
	c.paused = true
	var cn int32 = -1
	if p != nil {
		cn = int32(p.CN)
	}
	c.s.Broadcast(P.PauseGame{
		Paused: true,
		Client: cn,
	})
	c.deadline.Pause(clock)
	c.modeTimers.Pause()
}

func (c *casualClock) Paused() bool {
	return c.paused
}

func (c *casualClock) Resume(clock int64, p *Player) {
	if !c.paused {
		return
	}
	c.paused = false
	var cn int32 = -1
	if p != nil {
		cn = int32(p.CN)
	}
	c.s.Broadcast(P.PauseGame{
		Paused: false,
		Client: cn,
	})
	c.deadline.Resume(clock)
	c.modeTimers.Resume()
}

func (c *casualClock) Leave(*Player) {}

func (c *casualClock) Stop() {
	c.s.Broadcast(P.TimeUp{0})
	c.deadline.Stop()
}

func (c *casualClock) Ended() bool {
	return !c.deadline.Active()
}

func (c *casualClock) TimeLeft(clock int64) time.Duration {
	return time.Duration(c.deadline.TimeLeftMs(clock)) * time.Millisecond
}

func (c *casualClock) SetTimeLeft(clock int64, d time.Duration) {
	c.deadline.Set(clock, d.Milliseconds())
	c.s.Broadcast(P.TimeUp{int32(d / time.Second)})
}

func (c *casualClock) CleanUp() {
	c.deadline.Stop()
	c.modeTimers.CleanUp()
}

type endlessClock struct {
	s          Server
	modeTimers HasTimers
}

var _ Clock = &endlessClock{}

func NewEndlessClock(s Server, m HasTimers) *endlessClock {
	return &endlessClock{
		s:          s,
		modeTimers: m,
	}
}

func (c *endlessClock) Start(int64)         {}
func (c *endlessClock) Tick(int64)          {}
func (c *endlessClock) Pause(int64, *Player)  {}
func (c *endlessClock) Paused() bool        { return false }
func (c *endlessClock) Resume(int64, *Player) {}
func (c *endlessClock) Leave(*Player)       {}
func (c *endlessClock) Stop()               {}
func (c *endlessClock) Ended() bool         { return false }
func (c *endlessClock) TimeLeft(int64) time.Duration { return time.Hour }
func (c *endlessClock) SetTimeLeft(int64, time.Duration) {}

func (c *endlessClock) CleanUp() {
	c.modeTimers.CleanUp()
}

type Competitive interface {
	Clock
	Spawned(*Player)
}

type competitiveClock struct {
	*casualClock
	pendingResumeDeadlines [3]deadline.Deadline
	pendingResumeActive    bool
	mapLoadPending         map[*Player]struct{}
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

func (c *competitiveClock) Start(clock int64) {
	c.casualClock.Start(clock)
	c.s.ForEachPlayer(func(p *Player) {
		if p.State != 5 { // playerstate.Spectator
			c.mapLoadPending[p] = struct{}{}
		}
	})
	if len(c.mapLoadPending) > 0 {
		c.s.Message("waiting for all players to load the map")
		c.Pause(clock, nil)
	}
}

func (c *competitiveClock) Tick(clock int64) {
	// Check pending resume countdowns
	if c.pendingResumeActive {
		for i := range c.pendingResumeDeadlines {
			if c.pendingResumeDeadlines[i].Expired(clock) {
				c.pendingResumeDeadlines[i].Stop()
				switch i {
				case 0:
					c.s.Message("resuming game in 2 seconds")
				case 1:
					c.s.Message("resuming game in 1 second")
				case 2:
					c.casualClock.Resume(clock, nil)
					c.pendingResumeActive = false
				}
			}
		}
	}

	c.casualClock.Tick(clock)
}

func (c *competitiveClock) Spawned(p *Player) {
	delete(c.mapLoadPending, p)
	if len(c.mapLoadPending) == 0 {
		c.s.Message("all players spawned, starting game")
		clock := c.s.GameClock()
		c.Resume(clock, nil)
	}
}

func (c *competitiveClock) Pause(clock int64, p *Player) {
	if !c.casualClock.paused {
		c.casualClock.Pause(clock, p)
	} else if c.pendingResumeActive {
		// a resume is pending, cancel it
		c.Resume(clock, p)
	}
}

func (c *competitiveClock) Resume(clock int64, p *Player) {
	if c.pendingResumeActive {
		for i := range c.pendingResumeDeadlines {
			c.pendingResumeDeadlines[i].Stop()
		}
		c.pendingResumeActive = false
		c.s.Message("resuming aborted")
		return
	}

	if p != nil {
		c.s.Message(c.s.UniqueName(p) + " wants to resume the game")
	}
	c.s.Message("resuming game in 3 seconds")
	c.pendingResumeDeadlines[0].Set(clock, 1000)
	c.pendingResumeDeadlines[1].Set(clock, 2000)
	c.pendingResumeDeadlines[2].Set(clock, 3000)
	c.pendingResumeActive = true
}

func (c *competitiveClock) Leave(p *Player) {
	if p.State != 5 && !c.Ended() { // playerstate.Spectator
		c.s.Message("a player left the game")
		c.Pause(c.s.GameClock(), nil)
	}
}

func (c *competitiveClock) CleanUp() {
	for i := range c.pendingResumeDeadlines {
		c.pendingResumeDeadlines[i].Stop()
	}
	c.pendingResumeActive = false
	c.casualClock.CleanUp()
}
