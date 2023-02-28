package game

import (
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
)

type deathmatch struct {
	*teamlessMode
	noSpawnWait
}

func newDeathmatch(m *teamlessMode) *deathmatch {
	return &deathmatch{
		teamlessMode: m,
	}
}

type FFA struct {
	*deathmatch
	ffaSpawnState
	*handlesPickups
}

// assert interface implementations at compile time
var (
	_ Mode       = &FFA{}
	_ PickupMode = &FFA{}
)

func NewFFA(s Server) *FFA {
	return &FFA{
		deathmatch:     newDeathmatch(withoutTeams(s)),
		handlesPickups: handlingPickups(s),
	}
}

func (*FFA) ID() gamemode.ID { return gamemode.FFA }

type Effic struct {
	*deathmatch
	efficSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var _ Mode = &Effic{}

func NewEffic(s Server) *Effic {
	return &Effic{
		deathmatch: newDeathmatch(withoutTeams(s)),
	}
}

func (*Effic) ID() gamemode.ID { return gamemode.Effic }

type Insta struct {
	*deathmatch
	instaSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var _ Mode = &Insta{}

func NewInsta(s Server) *Insta {
	return &Insta{
		deathmatch: newDeathmatch(withoutTeams(s)),
	}
}

func (*Insta) ID() gamemode.ID { return gamemode.Insta }

type Tactics struct {
	*deathmatch
	tacticsSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var _ Mode = &Tactics{}

func NewTactics(s Server) *Tactics {
	return &Tactics{
		deathmatch: newDeathmatch(withoutTeams(s)),
	}
}

func (*Tactics) ID() gamemode.ID { return gamemode.Tactics }

type teamDeatchmatchMode struct {
	*teamMode
	noSpawnWait
}

func newTeamDeathmatchMode(s Server, keepTeams bool) *teamDeatchmatchMode {
	return &teamDeatchmatchMode{
		teamMode: withTeams(s, true, keepTeams, NewTeam("good"), NewTeam("evil")),
	}
}

type Teamplay struct {
	*teamDeatchmatchMode
	ffaSpawnState
	*handlesPickups
}

// assert interface implementations at compile time
var (
	_ Mode       = &Teamplay{}
	_ TeamMode   = &Teamplay{}
	_ PickupMode = &Teamplay{}
)

func NewTeamplay(s Server, keepTeams bool) *Teamplay {
	return &Teamplay{
		teamDeatchmatchMode: newTeamDeathmatchMode(s, keepTeams),
		handlesPickups:      handlingPickups(s),
	}
}

func (*Teamplay) ID() gamemode.ID { return gamemode.Teamplay }

type EfficTeam struct {
	*teamDeatchmatchMode
	efficSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var (
	_ Mode     = &EfficTeam{}
	_ TeamMode = &EfficTeam{}
)

func NewEfficTeam(s Server, keepTeams bool) *EfficTeam {
	return &EfficTeam{
		teamDeatchmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*EfficTeam) ID() gamemode.ID { return gamemode.EfficTeam }

type InstaTeam struct {
	*teamDeatchmatchMode
	instaSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var (
	_ Mode     = &InstaTeam{}
	_ TeamMode = &InstaTeam{}
)

func NewInstaTeam(s Server, keepTeams bool) *InstaTeam {
	return &InstaTeam{
		teamDeatchmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*InstaTeam) ID() gamemode.ID { return gamemode.InstaTeam }

type TacticsTeam struct {
	*teamDeatchmatchMode
	tacticsSpawnState
	noMapInfo
	noTimers
}

// assert interface implementations at compile time
var (
	_ Mode     = &TacticsTeam{}
	_ TeamMode = &TacticsTeam{}
)

func NewTacticsTeam(s Server, keepTeams bool) *TacticsTeam {
	return &TacticsTeam{
		teamDeatchmatchMode: newTeamDeathmatchMode(s, keepTeams),
	}
}

func (*TacticsTeam) ID() gamemode.ID { return gamemode.TacticsTeam }
