package sim_test

import (
	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
)

func newTestServer() *gameserver.Server {
	return newServerWithMode(gamemode.FFA, "complex")
}

func newServerWithMode(mode gamemode.ID, mapName string) *gameserver.Server {
	server := gameserver.New(&gameserver.Config{
		MaxClients:       32,
		MatchLength:      600,
		DefaultGameSpeed: 100,
		DefaultMode:      "ffa",
		DefaultMap:       mapName,
		Maps:             []string{"complex", "dust2", "turbine"},
	})
	server.StartGame(server.StartMode(mode), mapName)
	return server
}
