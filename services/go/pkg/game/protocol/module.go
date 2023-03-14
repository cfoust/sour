package protocol

import (
	"fmt"

	"github.com/cfoust/sour/pkg/game/io"
)

type Message interface {
	Type() MessageCode
}

type Packet struct {
	Channel  uint8
	Messages []Message
}

var FAILED = fmt.Errorf("failed to unmarshal message")

// N_ADDBOT
type AddBot struct {
	NumBots int32
}

func (m AddBot) Type() MessageCode { return N_ADDBOT }

// N_AUTHANS
type AuthAns struct {
	Description string
	Id          int32
	Answer      string
}

func (m AuthAns) Type() MessageCode { return N_AUTHANS }

// N_AUTHKICK
type AuthKick struct {
	Description string
	Answer      string
	Victim      int32
}

func (m AuthKick) Type() MessageCode { return N_AUTHKICK }

// N_AUTHTRY
type AuthTry struct {
	Description string
	Answer      string
}

func (m AuthTry) Type() MessageCode { return N_AUTHTRY }

// N_BOTBALANCE
type BotBalance struct {
	Balance int32
}

func (m BotBalance) Type() MessageCode { return N_BOTBALANCE }

// N_BOTLIMIT
type BotLimit struct {
	Limit int32
}

func (m BotLimit) Type() MessageCode { return N_BOTLIMIT }

// N_CHECKMAPS
type CheckMaps struct {
}

func (m CheckMaps) Type() MessageCode { return N_CHECKMAPS }

// N_CLEARBANS
type ClearBans struct {
}

func (m ClearBans) Type() MessageCode { return N_CLEARBANS }

// N_CLEARDEMOS
type ClearDemos struct {
	Demo int32
}

func (m ClearDemos) Type() MessageCode { return N_CLEARDEMOS }

// N_DELBOT
type DelBot struct {
}

func (m DelBot) Type() MessageCode { return N_DELBOT }

// N_DEMOPACKET
type DemoPacket struct {
}

func (m DemoPacket) Type() MessageCode { return N_DEMOPACKET }

// N_DEMOPLAYBACK
type DemoPlayback struct {
	On     int32
	Client int32
}

func (m DemoPlayback) Type() MessageCode { return N_DEMOPLAYBACK }

type Hit struct {
	Target       int32
	LifeSequence int32
	Distance     float64
	Rays         int32
	Direction    Vec
}

// N_EXPLODE
type Explode struct {
	Cmillis int32
	Gun     int32
	Id      int32
	Hits    []Hit
}

func (m Explode) Type() MessageCode { return N_EXPLODE }

// N_FORCEINTERMISSION
type ForceIntermission struct {
}

func (m ForceIntermission) Type() MessageCode { return N_FORCEINTERMISSION }

// N_FROMAI
type FromAI struct {
	Qcn int32
}

func (m FromAI) Type() MessageCode { return N_FROMAI }

// N_GAMESPEED
type GameSpeed struct {
	Speed  int32
	Client int32
}

func (m GameSpeed) Type() MessageCode { return N_GAMESPEED }

// N_GETDEMO
type GetDemo struct {
	Demo int32
	Tag  int32
}

func (m GetDemo) Type() MessageCode { return N_GETDEMO }

// N_GETMAP
type GetMap struct {
}

func (m GetMap) Type() MessageCode { return N_GETMAP }

// N_ITEMPICKUP
type ItemPickup struct {
	Item int32
}

func (m ItemPickup) Type() MessageCode { return N_ITEMPICKUP }

// N_KICK
type Kick struct {
	Victim int32
	Reason string
}

func (m Kick) Type() MessageCode { return N_KICK }

// N_LISTDEMOS
type ListDemos struct {
}

func (m ListDemos) Type() MessageCode { return N_LISTDEMOS }

// N_MAPCRC
type MapCRC struct {
	Map string
	Crc int32
}

func (m MapCRC) Type() MessageCode { return N_MAPCRC }

// N_MAPVOTE
type MapVote struct {
	Map  string
	Mode int32
}

func (m MapVote) Type() MessageCode { return N_MAPVOTE }

// N_RECORDDEMO
type RecordDemo struct {
	Enabled int32
}

func (m RecordDemo) Type() MessageCode { return N_RECORDDEMO }

// N_SENDMAP
type SendMap struct {
	Map []byte
}

func (m SendMap) Type() MessageCode { return N_SENDMAP }

func (s SendMap) Marshal(p *io.Packet) error {
	*p = append(*p, s.Map...)
	return nil
}

func (s *SendMap) Unmarshal(p *io.Packet) error {
	s.Map = *p
	*p = (*p)[0:0]
	return nil
}

// N_SERVCMD
type ServCMD struct {
	Command string
}

func (m ServCMD) Type() MessageCode { return N_SERVCMD }

// N_SETMASTER
type SetMaster struct {
	Client   int32
	Master   int32
	Password string
}

func (m SetMaster) Type() MessageCode { return N_SETMASTER }

// N_SHOOT
type Shoot struct {
	Id   int32
	Gun  int32
	From Vec
	To   Vec
	Hits []Hit
}

func (m Shoot) Type() MessageCode { return N_SHOOT }

// N_STOPDEMO
type StopDemo struct {
}

func (m StopDemo) Type() MessageCode { return N_STOPDEMO }

// N_SUICIDE
type Suicide struct {
}

func (m Suicide) Type() MessageCode { return N_SUICIDE }

// N_SWITCHTEAM
type SwitchTeam struct {
	Team string
}

func (m SwitchTeam) Type() MessageCode { return N_SWITCHTEAM }

// N_TRYDROPFLAG
type TryDropFlag struct {
}

func (m TryDropFlag) Type() MessageCode { return N_TRYDROPFLAG }

// N_CONNECT
type Connect struct {
	Name            string
	Model           int32
	Password        string
	AuthDescription string
	AuthName        string
}

func (m Connect) Type() MessageCode { return N_CONNECT }

// N_SERVINFO
type ServerInfo struct {
	Client      int32
	Protocol    int32
	SessionId   int32
	HasPassword bool
	Description string
	Domain      string
}

func (m ServerInfo) Type() MessageCode { return N_SERVINFO }

// N_WELCOME
type Welcome struct {
}

func (m Welcome) Type() MessageCode { return N_WELCOME }

// N_AUTHCHAL
type AuthChallenge struct {
	Desc      string
	Id        int32
	Challenge string
}

func (m AuthChallenge) Type() MessageCode { return N_AUTHCHAL }

// N_PONG
type Pong struct {
	Cmillis int32
}

func (m Pong) Type() MessageCode { return N_PONG }

// N_PING
type Ping struct {
	Cmillis int32
}

func (m Ping) Type() MessageCode { return N_PING }

// N_POS
type Pos struct {
	Client int32
	State  PhysicsState
}

func (m Pos) Type() MessageCode { return N_POS }

// N_SERVMSG
type ServerMessage struct {
	Text string
}

func (m ServerMessage) Type() MessageCode { return N_SERVMSG }

// N_PAUSEGAME
type PauseGame struct {
	Paused bool
	Client int32
}

func (m PauseGame) Type() MessageCode { return N_PAUSEGAME }

// N_TIMEUP
type TimeUp struct {
	Remaining int32
}

func (m TimeUp) Type() MessageCode { return N_TIMEUP }

// N_ANNOUNCE
type Announce struct {
	Announcement int32
}

func (m Announce) Type() MessageCode { return N_ANNOUNCE }

// N_MASTERMODE
type MasterMode struct {
	MasterMode int32
}

func (m MasterMode) Type() MessageCode { return N_MASTERMODE }

// N_CDIS
type ClientDisconnected struct {
	Client int32
}

func (m ClientDisconnected) Type() MessageCode { return N_CDIS }

// N_JUMPPAD
type JumpPad struct {
	Client  int32
	JumpPad int32
}

func (m JumpPad) Type() MessageCode { return N_JUMPPAD }

// N_TELEPORT
type Teleport struct {
	Client      int32
	Source      int32
	Destination int32
}

func (m Teleport) Type() MessageCode { return N_TELEPORT }

// N_SPECTATOR
type Spectator struct {
	Client     int32
	Spectating bool
}

func (m Spectator) Type() MessageCode { return N_SPECTATOR }

// N_SETTEAM
type SetTeam struct {
	Client int32
	Team   string
	Reason int32
}

func (m SetTeam) Type() MessageCode { return N_SETTEAM }

// N_CURRENTMASTER
type ClientPrivilege struct {
	Client    int32
	Privilege int32
}
type CurrentMaster struct {
	MasterMode int32
	Clients    []ClientPrivilege `type:"term"`
}

func (m CurrentMaster) Type() MessageCode { return N_CURRENTMASTER }

// N_MAPCHANGE
type MapChange struct {
	Name     string
	Mode     int32
	HasItems bool
}

func (m MapChange) Type() MessageCode { return N_MAPCHANGE }

// N_TEAMINFO
type Team struct {
	Team  string
	Frags int32
}
type TeamInfo struct {
	Teams []Team `type:"term"`
}

func (m TeamInfo) Type() MessageCode { return N_TEAMINFO }

// N_INITCLIENT
type InitClient struct {
	Client      int32
	Name        string
	Team        string
	Playermodel int32
}

func (m InitClient) Type() MessageCode { return N_INITCLIENT }

// N_SPAWNSTATE
type AmmoState struct {
	Amount int32
}
type EntityState struct {
	LifeSequence int32
	Health       int32
	MaxHealth    int32
	Armour       int32
	Armourtype   int32
	Gunselect    int32
	Ammo         [6]AmmoState
}
type SpawnState struct {
	Client int32
	EntityState
}

func (m SpawnState) Type() MessageCode { return N_SPAWNSTATE }

type ClientState struct {
	Id         int32
	State      int32
	Frags      int32
	Flags      int32
	Deaths     int32
	Quadmillis int32
	EntityState
}

// N_RESUME
type Resume struct {
	Clients []ClientState `type:"term"`
}

func (m Resume) Type() MessageCode { return N_RESUME }

// N_INITFLAGS
type TeamScore struct {
	Score int32
}

type FlagState struct {
	Version   int32
	Spawn     int32
	Owner     int32
	Invisible bool
	Dropped   bool
	Position  Vec
}

func (f FlagState) Marshal(p *io.Packet) error {
	err := p.Put(
		f.Version,
		f.Spawn,
		f.Owner,
		f.Invisible,
	)
	if err != nil {
		return err
	}

	if f.Owner >= 0 {
		return nil
	}

	err = p.Put(
		f.Dropped,
	)
	if err != nil {
		return err
	}

	if !f.Dropped {
		return nil
	}

	err = p.Put(
		f.Position,
	)
	if err != nil {
		return err
	}

	// TODO support m_hold

	return nil
}

func (f *FlagState) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&f.Version,
		&f.Spawn,
		&f.Owner,
		&f.Invisible,
	)
	if err != nil {
		return err
	}

	if f.Owner >= 0 {
		return nil
	}

	err = p.Get(
		&f.Dropped,
	)
	if err != nil {
		return err
	}

	if !f.Dropped {
		return nil
	}

	err = p.Get(
		&f.Position,
	)
	if err != nil {
		return err
	}

	// TODO support m_hold

	return nil
}

type ServerInitFlags struct {
	Scores [2]TeamScore
	Flags  []FlagState
}

func (m ServerInitFlags) Type() MessageCode { return N_INITFLAGS }

type ClientFlagState struct {
	Team     int32
	Position Vec
}

type ClientInitFlags struct {
	Flags []ClientFlagState
}

func (m ClientInitFlags) Type() MessageCode { return N_INITFLAGS }

// N_DROPFLAG
type DropFlag struct {
	Client   int32
	Flag     int32
	Version  int32
	Position Vec
}

func (m DropFlag) Type() MessageCode { return N_DROPFLAG }

// N_SCOREFLAG
type ScoreFlag struct {
	Client       int32
	Relayflag    int32
	Relayversion int32
	Goalflag     int32
	Goalversion  int32
	Goalspawn    int32
	Team         int32
	Score        int32
	Oflags       int32
}

func (m ScoreFlag) Type() MessageCode { return N_SCOREFLAG }

// N_RETURNFLAG
type ReturnFlag struct {
	Client  int32
	Flag    int32
	Version int32
}

func (m ReturnFlag) Type() MessageCode { return N_RETURNFLAG }

// N_TAKEFLAG
type TakeFlag struct {
	Client  int32
	Flag    int32
	Version int32
}

func (m TakeFlag) Type() MessageCode { return N_TAKEFLAG }

// N_RESETFLAG
type ResetFlag struct {
	Flag    int32
	Version int32
	Spawn   int32
	Team    int32
	Score   int32
}

func (m ResetFlag) Type() MessageCode { return N_RESETFLAG }

// N_INVISFLAG
type InvisFlag struct {
	Flag      int32
	Invisible int32
}

func (m InvisFlag) Type() MessageCode { return N_INVISFLAG }

// N_BASES
type BaseState struct {
	AmmoType  int32
	Owner     string
	Enemy     string
	Converted int32
	AmmoCount int32
}
type Bases struct {
	Bases []BaseState
}

func (m Bases) Type() MessageCode { return N_BASES }

// N_BASEINFO
type BaseInfo struct {
	Base      int32
	Owner     string
	Enemy     string
	Converted int32
	AmmoCount int32
}

func (m BaseInfo) Type() MessageCode { return N_BASEINFO }

// N_BASESCORE
type BaseScore struct {
	Base  int32
	Team  string
	Total int32
}

func (m BaseScore) Type() MessageCode { return N_BASESCORE }

// N_REPAMMO
type ReplenishAmmo struct {
	Client   int32
	Ammotype int32
}

func (m ReplenishAmmo) Type() MessageCode { return N_REPAMMO }

// N_TRYSPAWN
type TrySpawn struct {
}

func (m TrySpawn) Type() MessageCode { return N_TRYSPAWN }

// N_BASEREGEN
type BaseRegen struct {
	Client   int32
	Health   int32
	Armour   int32
	Ammotype int32
	Ammo     int32
}

func (m BaseRegen) Type() MessageCode { return N_BASEREGEN }

// N_INITTOKENS
type TokenState struct {
	Token int32
	Team  int32
	Yaw   int32
	X     int32
	Y     int32
	Z     int32
}
type ClientTokenState struct {
	Client int32
	Count  int32
}
type InitTokens struct {
	TeamScores   [2]TeamScore
	Tokens       []TokenState
	ClientTokens []ClientTokenState `type:"term"`
}

func (m InitTokens) Type() MessageCode { return N_INITTOKENS }

// N_TAKETOKEN
type TakeToken struct {
	Client int32
	Token  int32
	Total  int32
}

func (m TakeToken) Type() MessageCode { return N_TAKETOKEN }

// N_EXPIRETOKENS
type ExpireTokens struct {
	Tokens []struct {
		Token int32
	} `type:"term"`
}

func (m ExpireTokens) Type() MessageCode { return N_EXPIRETOKENS }

// N_DROPTOKENS
type DropTokens struct {
	Client int32
	Dropx  int32
	Dropy  int32
	Dropz  int32
	Tokens []struct {
		Token int32
		Team  int32
		Yaw   int32
	} `type:"term"`
}

func (m DropTokens) Type() MessageCode { return N_DROPTOKENS }

// N_STEALTOKENS
type StealTokens struct {
	Client    int32
	Team      int32
	Basenum   int32
	Enemyteam int32
	Score     int32
	Dropx     int32
	Dropy     int32
	Dropz     int32
	Tokens    []struct {
		Token int32
		Team  int32
		Yaw   int32
	} `type:"term"`
}

func (m StealTokens) Type() MessageCode { return N_STEALTOKENS }

// N_DEPOSITTOKENS
type DepositTokens struct {
	Client    int32
	Base      int32
	Deposited int32
	Team      int32
	Score     int32
	Flags     int32
}

func (m DepositTokens) Type() MessageCode { return N_DEPOSITTOKENS }

// N_ITEMLIST
type Item struct {
	Index int32
	Type  int32
}
type ItemList struct {
	Items []Item `type:"term"`
}

func (m ItemList) Type() MessageCode { return N_ITEMLIST }

// N_ITEMSPAWN
type ItemSpawn struct {
	Index int32
}

func (m ItemSpawn) Type() MessageCode { return N_ITEMSPAWN }

// N_ITEMACC
type ItemAck struct {
	Index  int32
	Client int32
}

func (m ItemAck) Type() MessageCode { return N_ITEMACC }

// N_HITPUSH
type HitPush struct {
	Client int32
	Gun    int32
	Damage int32
	From   Vec
}

func (m HitPush) Type() MessageCode { return N_HITPUSH }

// N_SHOTFX
type ShotFX struct {
	Client int32
	Gun    int32
	Id     int32
	From   Vec
	To     Vec
}

func (m ShotFX) Type() MessageCode { return N_SHOTFX }

// N_EXPLODEFX
type ExplodeFX struct {
	Client int32
	Gun    int32
	Id     int32
}

func (m ExplodeFX) Type() MessageCode { return N_EXPLODEFX }

// N_DAMAGE
type Damage struct {
	Client    int32
	Aggressor int32
	Damage    int32
	Armour    int32
	Health    int32
}

func (m Damage) Type() MessageCode { return N_DAMAGE }

// N_DIED
type Died struct {
	Client      int32
	Killer      int32
	KillerFrags int32
	VictimFrags int32
}

func (m Died) Type() MessageCode { return N_DIED }

// N_FORCEDEATH
type ForceDeath struct {
	Client int32
}

func (m ForceDeath) Type() MessageCode { return N_FORCEDEATH }

// N_REQAUTH
type ReqAuth struct {
	Domain string
}

func (m ReqAuth) Type() MessageCode { return N_REQAUTH }

// N_INITAI
type InitAI struct {
	Aiclientnum    int32
	Ownerclientnum int32
	Aitype         int32
	Aiskill        int32
	Playermodel    int32
	Name           string
	Team           string
}

func (m InitAI) Type() MessageCode { return N_INITAI }

// N_SENDDEMOLIST
type SendDemoList struct {
	Demos []struct {
		Info string
	}
}

func (m SendDemoList) Type() MessageCode { return N_SENDDEMOLIST }

// N_SENDDEMO
type SendDemo struct {
	Tag  int32
	Data []byte
}

func (m SendDemo) Type() MessageCode { return N_SENDDEMO }

func (s SendDemo) Marshal(p *io.Packet) error {
	err := p.Put(s.Tag)
	if err != nil {
		return err
	}
	*p = append(*p, s.Data...)
	return nil
}

func (s *SendDemo) Unmarshal(p *io.Packet) error {
	err := p.Get(
		&s.Tag,
	)
	if err != nil {
		return err
	}
	s.Data = *p
	*p = (*p)[0:0]
	return nil
}

// N_CLIENT
type ClientPacket struct {
	Client int32
	Length int32
}

func (m ClientPacket) Type() MessageCode { return N_CLIENT }

// N_SPAWN <- from server
type SpawnResponse struct {
	EntityState
}

func (m SpawnResponse) Type() MessageCode { return N_SPAWN }

// N_SPAWN <- from client
type SpawnRequest struct {
	LifeSequence int32
	GunSelect    int32
}

func (m SpawnRequest) Type() MessageCode { return N_SPAWN }

// N_SOUND
type Sound struct {
	Sound int32
}

func (m Sound) Type() MessageCode { return N_SOUND }

// N_CLIENTPING
type ClientPing struct {
	Ping int32
}

func (m ClientPing) Type() MessageCode { return N_CLIENTPING }

// N_TAUNT
type Taunt struct {
}

func (m Taunt) Type() MessageCode { return N_TAUNT }

// N_GUNSELECT
type GunSelect struct {
	GunSelect int32
}

func (m GunSelect) Type() MessageCode { return N_GUNSELECT }

// N_TEXT
type Text struct {
	Text string
}

func (m Text) Type() MessageCode { return N_TEXT }

// N_SAYTEAM
type SayTeam struct {
	Text string
}

func (m SayTeam) Type() MessageCode { return N_SAYTEAM }

// N_SWITCHNAME
type SwitchName struct {
	Name string
}

func (m SwitchName) Type() MessageCode { return N_SWITCHNAME }

// N_SWITCHMODEL
type SwitchModel struct {
	Model int32
}

func (m SwitchModel) Type() MessageCode { return N_SWITCHMODEL }

// N_EDITMODE
type EditMode struct {
	Enabled bool
}

func (m EditMode) Type() MessageCode { return N_EDITMODE }

// This does not represent the messages a client or server is permitted to send
// (= presence here does not imply a message of that type is valid), there are
// just some differences in message structures depending on whether they came
// from the client or the server.
var CLIENT_MESSAGES = make(map[MessageCode]Message)
var SERVER_MESSAGES = make(map[MessageCode]Message)

func registerBoth(message Message) {
	CLIENT_MESSAGES[message.Type()] = message
	SERVER_MESSAGES[message.Type()] = message
}

func registerClient(message Message) {
	CLIENT_MESSAGES[message.Type()] = message
}

func registerServer(message Message) {
	SERVER_MESSAGES[message.Type()] = message
}

func init() {
	registerBoth(&AddBot{})
	registerBoth(&Announce{})
	registerBoth(&AuthAns{})
	registerBoth(&AuthChallenge{})
	registerBoth(&AuthKick{})
	registerBoth(&AuthTry{})
	registerBoth(&BaseInfo{})
	registerBoth(&BaseRegen{})
	registerBoth(&BaseScore{})
	registerBoth(&Bases{})
	registerBoth(&BotBalance{})
	registerBoth(&BotLimit{})
	registerBoth(&CheckMaps{})
	registerBoth(&ClearBans{})
	registerBoth(&ClearDemos{})
	registerBoth(&ClientDisconnected{})
	registerBoth(&ClientPacket{})
	registerBoth(&ClientPing{})
	registerBoth(&Connect{})
	registerBoth(&CurrentMaster{})
	registerBoth(&Damage{})
	registerBoth(&DelBot{})
	registerBoth(&DemoPacket{})
	registerBoth(&DemoPlayback{})
	registerBoth(&DepositTokens{})
	registerBoth(&Died{})
	registerBoth(&DropFlag{})
	registerBoth(&DropTokens{})
	registerBoth(&EditMode{})
	registerBoth(&ExpireTokens{})
	registerBoth(&ExplodeFX{})
	registerBoth(&Explode{})
	registerBoth(&ForceDeath{})
	registerBoth(&ForceIntermission{})
	registerBoth(&FromAI{})
	registerBoth(&GameSpeed{})
	registerBoth(&GetDemo{})
	registerBoth(&GetMap{})
	registerBoth(&GunSelect{})
	registerBoth(&HitPush{})
	registerBoth(&InitAI{})
	registerBoth(&InitClient{})
	registerBoth(&InitTokens{})
	registerBoth(&InvisFlag{})
	registerBoth(&ItemAck{})
	registerBoth(&ItemList{})
	registerBoth(&ItemPickup{})
	registerBoth(&ItemSpawn{})
	registerBoth(&JumpPad{})
	registerBoth(&Kick{})
	registerBoth(&ListDemos{})
	registerBoth(&MapCRC{})
	registerBoth(&MapChange{})
	registerBoth(&MapVote{})
	registerBoth(&MasterMode{})
	registerBoth(&PauseGame{})
	registerBoth(&Ping{})
	registerBoth(&Pong{})
	registerBoth(&Pos{})
	registerBoth(&RecordDemo{})
	registerBoth(&ReplenishAmmo{})
	registerBoth(&ReqAuth{})
	registerBoth(&ResetFlag{})
	registerBoth(&Resume{})
	registerBoth(&ReturnFlag{})
	registerBoth(&SayTeam{})
	registerBoth(&ScoreFlag{})
	registerBoth(&SendDemoList{})
	registerBoth(&SendDemo{})
	registerBoth(&SendMap{})
	registerBoth(&ServCMD{})
	registerBoth(&ServerInfo{})
	registerBoth(&ServerMessage{})
	registerBoth(&SetMaster{})
	registerBoth(&SetTeam{})
	registerBoth(&Shoot{})
	registerBoth(&ShotFX{})
	registerBoth(&Sound{})
	registerBoth(&SpawnState{})
	registerBoth(&Spectator{})
	registerBoth(&StealTokens{})
	registerBoth(&StopDemo{})
	registerBoth(&Suicide{})
	registerBoth(&SwitchModel{})
	registerBoth(&SwitchName{})
	registerBoth(&SwitchTeam{})
	registerBoth(&TakeFlag{})
	registerBoth(&TakeToken{})
	registerBoth(&Taunt{})
	registerBoth(&TeamInfo{})
	registerBoth(&Teleport{})
	registerBoth(&Text{})
	registerBoth(&TimeUp{})
	registerBoth(&TryDropFlag{})
	registerBoth(&TrySpawn{})
	registerBoth(&Welcome{})
	registerClient(&SpawnRequest{})
	registerClient(&ClientInitFlags{})
	registerServer(&ServerInitFlags{})
	registerServer(&SpawnResponse{})

	// editing
	registerBoth(&Clipboard{})
	registerBoth(&Copy{})
	registerBoth(&DeleteCube{})
	registerBoth(&EditEntity{})
	registerBoth(&EditFace{})
	registerBoth(&EditMaterial{})
	registerBoth(&EditTexture{})
	registerBoth(&EditVSlot{})
	registerBoth(&Flip{})
	registerBoth(&NewMap{})
	registerBoth(&Paste{})
	registerBoth(&Redo{})
	registerBoth(&Remip{})
	registerBoth(&Replace{})
	registerBoth(&Rotate{})
	registerBoth(&Undo{})
}
