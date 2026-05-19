// Package deadline provides a deterministic game-clock timer type.
package deadline

// Deadline represents a future point in game time, replacing timer.Timer.
// It is a passive data structure — no goroutines, no channels, no mutexes.
// Expiry must be checked explicitly via Expired().
type Deadline struct {
	expires int64 // absolute ms timestamp when this fires
	active  bool
}

// Set schedules the deadline to expire at clock + delayMs.
func (d *Deadline) Set(clock int64, delayMs int64) {
	d.expires = clock + delayMs
	d.active = true
}

// Expired returns true if the deadline is active and has passed.
func (d *Deadline) Expired(clock int64) bool {
	return d.active && clock >= d.expires
}

// TimeLeftMs returns milliseconds remaining, or 0 if expired/inactive.
func (d *Deadline) TimeLeftMs(clock int64) int64 {
	if !d.active {
		return 0
	}
	left := d.expires - clock
	if left < 0 {
		return 0
	}
	return left
}

// Active returns whether the deadline is set and has not been stopped.
func (d *Deadline) Active() bool {
	return d.active
}

// Stop deactivates the deadline.
func (d *Deadline) Stop() {
	d.active = false
	d.expires = 0
}

// Pause converts the absolute deadline to a remaining duration so that
// Resume can re-anchor it to a new clock value.
func (d *Deadline) Pause(clock int64) {
	if d.active {
		remaining := d.expires - clock
		if remaining < 0 {
			remaining = 0
		}
		d.expires = remaining // temporarily store remaining duration
	}
}

// Resume re-anchors a paused deadline to the current clock.
func (d *Deadline) Resume(clock int64) {
	if d.active {
		d.expires = clock + d.expires // convert remaining back to absolute
	}
}
