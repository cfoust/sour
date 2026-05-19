package gameserver

type Config struct {
	MaxClients       int
	MatchLength      int // seconds
	DefaultGameSpeed int
	DefaultMode      string
	DefaultMap       string
	Maps             []string
	TickRateMs       int // ms between Step() calls; 0 = default 33
}

func (c *Config) GetTickRateMs() int {
	if c.TickRateMs <= 0 {
		return 33
	}
	return c.TickRateMs
}
