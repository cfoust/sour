package server

import (
	"fmt"

	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
)

func (s *Server) StartMode(id gamemode.ID) game.Mode {
	switch id {
	case gamemode.FFA:
		return game.NewFFA(s)
	case gamemode.CoopEdit:
		return game.NewCoopEdit(s)
	case gamemode.Insta:
		return game.NewInsta(s)
	case gamemode.InstaTeam:
		return game.NewInstaTeam(s, s.KeepTeams)
	case gamemode.Effic:
		return game.NewEffic(s)
	case gamemode.Teamplay:
		return game.NewTeamplay(s, s.KeepTeams)
	case gamemode.EfficTeam:
		return game.NewEfficTeam(s, s.KeepTeams)
	case gamemode.Tactics:
		return game.NewTactics(s)
	case gamemode.TacticsTeam:
		return game.NewTacticsTeam(s, s.KeepTeams)
	case gamemode.CTF:
		return game.NewCTF(s, s.KeepTeams)
	case gamemode.InstaCTF:
		return game.NewInstaCTF(s, s.KeepTeams)
	case gamemode.EfficCTF:
		return game.NewEfficCTF(s, s.KeepTeams)
	default:
		panic(fmt.Sprintf("unhandled gamemode ID %d", id))
	}
}
