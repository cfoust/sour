package pausableticker

import (
	"sync"
	"time"
)

type Ticker struct {
	C <-chan time.Time // The channel on which the ticks are delivered.

	sync.Mutex
	pause  chan bool
	paused bool
	stop   chan struct{}
	ticker *time.Ticker
}

func New(d time.Duration) *Ticker {
	c := make(chan time.Time)
	pause := make(chan bool)
	stop := make(chan struct{})
	ticker := time.NewTicker(d)

	t := &Ticker{
		C:      c,
		pause:  pause,
		stop:   stop,
		ticker: ticker,
	}

	go t.run(c)

	return t
}

func (t *Ticker) run(c chan<- time.Time) {
	defer close(t.stop)

	for {
		select {
		case c <- <-t.ticker.C:
		case shouldPause := <-t.pause:
			if shouldPause {
				t.paused = true
				for shouldPause {
					select {
					case shouldPause = <-t.pause:
					case <-t.stop:
						return
					}
				}
				t.paused = false
			}
		case <-t.stop:
			return
		}
	}
}

func (t *Ticker) Pause() {
	t.Lock()
	defer t.Unlock()

	if t.pause != nil {
		t.pause <- true
	}
}

func (t *Ticker) Paused() bool {
	return t.paused
}

func (t *Ticker) Resume() {
	t.Lock()
	defer t.Unlock()

	if t.pause != nil {
		t.pause <- false
	}
}

func (t *Ticker) Stop() {
	t.Lock()
	defer t.Unlock()

	if t.stop != nil {
		close(t.pause)
		t.pause = nil
		t.stop <- struct{}{}
		<-t.stop
		t.stop = nil
		go t.ticker.Stop()
	}
}

func (t *Ticker) Stopped() bool {
	return t.stop == nil
}
