package gameserver

type Config struct {
	MaxClients       int
	MatchLength      int
	DefaultGameSpeed int
	DefaultMode      string
	DefaultMap       string
	Maps             []string
}
