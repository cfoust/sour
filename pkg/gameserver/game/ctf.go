package game

import (
	"log"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
)

type CTFMode = FlagMode

type ctfMode = handlesFlags

func newCTFMode(s Server, keepTeams bool) *ctfMode {
	good, evil := NewTeam("good"), NewTeam("evil")
	return handlingFlags(
		newCTF(
			s,
			withTeams(s, false, keepTeams, good, evil),
			good,
			evil,
		),
	)
}

type CTF struct {
	ctfSpawnState
	*ctfMode
	*handlesPickups
}

// assert interface implementations at compile time
var (
	_ Mode       = &CTF{}
	_ HasTimers  = &CTF{}
	_ TeamMode   = &CTF{}
	_ FlagMode   = &CTF{}
	_ PickupMode = &CTF{}
)

func NewCTF(s Server, keepTeams bool) *CTF {
	return &CTF{
		ctfMode:        newCTFMode(s, keepTeams),
		handlesPickups: handlingPickups(s),
	}
}

func (m *CTF) NeedsMapInfo() bool {
	return m.handlesPickups.NeedsMapInfo() || m.ctfMode.NeedsMapInfo()
}

func (m *CTF) HandlePacket(p *Player, message P.Message) bool {
	switch message.Type() {

	case P.N_INITFLAGS, P.N_TAKEFLAG, P.N_TRYDROPFLAG:
		return m.ctfMode.HandlePacket(p, message)

	case P.N_ITEMLIST, P.N_ITEMPICKUP:
		return m.handlesPickups.HandlePacket(p, message)
	default:
		log.Println("received unrelated packet", message)
		return false
	}
}

func (m *CTF) Pause() {
	m.ctfMode.Pause()
	m.handlesPickups.Pause()
}

func (m *CTF) Resume() {
	m.ctfMode.Resume()
	m.handlesPickups.Resume()
}

func (m *CTF) CleanUp() {
	m.ctfMode.CleanUp()
	m.handlesPickups.CleanUp()
}

func (*CTF) ID() gamemode.ID { return gamemode.CTF }

type EfficCTF struct {
	efficSpawnState
	*ctfMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &EfficCTF{}
	_ HasTimers = &EfficCTF{}
	_ TeamMode  = &EfficCTF{}
	_ FlagMode  = &EfficCTF{}
)

func NewEfficCTF(s Server, keepTeams bool) *EfficCTF {
	return &EfficCTF{
		ctfMode: newCTFMode(s, keepTeams),
	}
}

func (*EfficCTF) ID() gamemode.ID { return gamemode.EfficCTF }

type InstaCTF struct {
	instaSpawnState
	*ctfMode
}

// assert interface implementations at compile time
var (
	_ Mode      = &InstaCTF{}
	_ HasTimers = &InstaCTF{}
	_ TeamMode  = &InstaCTF{}
	_ FlagMode  = &InstaCTF{}
)

func NewInstaCTF(s Server, keepTeams bool) *InstaCTF {
	return &InstaCTF{
		ctfMode: newCTFMode(s, keepTeams),
	}
}

func (*InstaCTF) ID() gamemode.ID { return gamemode.InstaCTF }
