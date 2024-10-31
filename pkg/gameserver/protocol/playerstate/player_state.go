package playerstate

type ID uint32

const (
	Alive ID = iota
	Dead
	_ // Spawning, not used on server side
	_ // Lagged, not used on server side
	Editing
	Spectator
)
