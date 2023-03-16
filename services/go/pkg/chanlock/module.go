package chanlock

import (
	"context"
	"time"
	//"reflect"

	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

// Utility for diagnosing channel lock.
type Chanlock struct {
	log      zerolog.Logger
	lastMark string
	ticker   *time.Ticker
	mutex    deadlock.RWMutex
}

const (
	TIMEOUT_DURATION      = 15 * time.Second
	HEALTH_CHECK_DURATION = 1 * time.Second
)

func New(logger zerolog.Logger) *Chanlock {
	return &Chanlock{
		log:    logger,
		ticker: time.NewTicker(HEALTH_CHECK_DURATION),
	}
}

func (c *Chanlock) Mark(name string) {
	c.mutex.Lock()
	c.lastMark = name
	c.mutex.Unlock()
}

func (c *Chanlock) Poll(ctx context.Context) <-chan time.Time {
	out := make(chan time.Time)

	go func() {
		for {
			select {
			case t := <-c.ticker.C:
				timeout := time.NewTimer(TIMEOUT_DURATION)
				ok := make(chan bool)
				go func() {
					select {
					case <-ctx.Done():
						return
					case <-ok:
						return
					case <-timeout.C:
						c.mutex.RLock()
						mark := c.lastMark
						c.mutex.RUnlock()

						c.log.Error().Msgf("event loop no longer healthy")

						if mark != "" {
							c.log.Error().Msgf("last mark: %s", mark)
						}
					}
				}()
				out <- t
				ok <- true
				c.Mark("")
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}
