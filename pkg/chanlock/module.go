package chanlock

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/petermattis/goid"
	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

// Opts control how deadlock detection behaves.
// Options are supposed to be set once at a startup (say, when parsing flags).
var Opts = struct {
	// -- almost no runtime penalty, no deadlock detection if Disable == true.
	Disable bool
	// Waiting for a lock for longer than DeadlockTimeout is considered a deadlock.
	// Ignored if DeadlockTimeout <= 0.
	DeadlockTimeout time.Duration
	// The rate at which chanlock's ticker executes.
	HealthCheckRate time.Duration
	// OnPotentialDeadlock is called each time a potential deadlock is detected -- either based on
	// lock order or on lock wait time.
	OnPotentialDeadlock func()
	// Will keep MaxMapSize lock pairs (happens before // happens after) in the map.
	// The map resets once the threshold is reached.
	MaxMapSize int
	// Will dump stacktraces of all goroutines when inconsistent locking is detected.
	PrintAllCurrentGoroutines bool
	mu                        *sync.Mutex // Protects the LogBuf.
	// Will print deadlock info to log buffer.
	LogBuf io.Writer
}{
	DeadlockTimeout: time.Second * 30,
	HealthCheckRate: time.Second * 1,
	OnPotentialDeadlock: func() {
		os.Exit(2)
	},
	MaxMapSize: 1024 * 64,
	mu:         &sync.Mutex{},
	LogBuf:     os.Stderr,
}

// Utility for diagnosing channel lock.
type Chanlock struct {
	log    zerolog.Logger
	ticker *time.Ticker
	mutex  deadlock.RWMutex
}

func New() *Chanlock {
	return &Chanlock{
		ticker: time.NewTicker(Opts.HealthCheckRate),
	}
}

func (c *Chanlock) reportFailure(id int64, stack []uintptr) {
	Opts.mu.Lock()
	fmt.Fprintln(Opts.LogBuf, header)
	fmt.Fprintln(Opts.LogBuf, "Event loop has been unresponsive for more than", Opts.DeadlockTimeout)
	fmt.Fprintf(Opts.LogBuf, "goroutine %v\n", id)
	printStack(Opts.LogBuf, stack)
	stacks := stacks()
	grs := bytes.Split(stacks, []byte("\n\n"))
	for _, g := range grs {
		if goid.ExtractGID(g) == id {
			fmt.Fprintln(Opts.LogBuf, "Here is what goroutine", id, "is doing now")
			Opts.LogBuf.Write(g)
			fmt.Fprintln(Opts.LogBuf)
		}
	}
	if Opts.PrintAllCurrentGoroutines {
		fmt.Fprintln(Opts.LogBuf, "All current goroutines:")
		Opts.LogBuf.Write(stacks)
	}
	fmt.Fprintln(Opts.LogBuf)
	if buf, ok := Opts.LogBuf.(*bufio.Writer); ok {
		buf.Flush()
	}
	Opts.mu.Unlock()
	Opts.OnPotentialDeadlock()
}

func (c *Chanlock) Poll(ctx context.Context) <-chan time.Time {
	out := make(chan time.Time)
	currentID := goid.Get()
	stack := callers(1)

	go func() {
		for {
			select {
			case t := <-c.ticker.C:
				timeout := time.NewTimer(Opts.DeadlockTimeout)
				ok := make(chan bool)
				go func() {
					select {
					case <-ctx.Done():
						return
					case <-ok:
						return
					case <-timeout.C:
						c.reportFailure(currentID, stack)
					}
				}()
				out <- t
				ok <- true
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

const header = "POTENTIAL CHANLOCK:"
