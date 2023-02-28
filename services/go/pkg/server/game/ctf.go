package game

import (
	"log"

	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
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

func (m *CTF) HandlePacket(p *Player, packetType nmc.ID, pkt *protocol.Packet) bool {
	switch packetType {
	case nmc.InitFlags,
		nmc.TouchFlag,
		nmc.TryDropFlag:
		return m.ctfMode.HandlePacket(p, packetType, pkt)
	case nmc.PickupList,
		nmc.PickupTry:
		return m.handlesPickups.HandlePacket(p, packetType, pkt)
	default:
		log.Println("received unrelated packet", packetType, pkt)
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
