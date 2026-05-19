package sim_test

import "testing"

// raceEnabled is set to true when the race detector is active.
// See norace_test.go for the non-race counterpart.
var raceEnabled = false

func skipIfRace(t *testing.T) {
	if raceEnabled {
		t.Skip("skipping under race detector (scheduling-sensitive)")
	}
}
