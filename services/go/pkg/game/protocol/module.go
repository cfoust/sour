package protocol

import (
	"github.com/cfoust/sour/pkg/game/io"
)

type Message interface {
	Type() MessageCode
}

// N_ADDBOT
type AddBot struct {
	NumBots int
}

func (m AddBot) Type() MessageCode { return N_ADDBOT }

// N_AUTHANS
type AuthAns struct {
	Description string
	Id          int
	Answer      string
}

func (m AuthAns) Type() MessageCode { return N_AUTHANS }

// N_AUTHKICK
type AuthKick struct {
	Description string
	Answer      string
	Victim      int
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
	Balance int
}

func (m BotBalance) Type() MessageCode { return N_BOTBALANCE }

// N_BOTLIMIT
type BotLimit struct {
	Limit int
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
	Demo int
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
	On     int
	Client int
}

func (m DemoPlayback) Type() MessageCode { return N_DEMOPLAYBACK }

type Hit struct {
	Target       int
	Lifesequence int
	// TODO impl this calc
	// hit.dist = getint(p)/DMF;
	Dist int
	Rays int
	// TODO
	// hit.dir[k] = getint(p)/DMF;
	Dir0 int
	Dir1 int
	Dir2 int
}

// N_EXPLODE
type Explode struct {
	Cmillis int
	Gun     int
	Id      int
	Hits    []Hit `type:"count"`
}

func (m Explode) Type() MessageCode { return N_EXPLODE }

// N_FORCEINTERMISSION
type ForceIntermission struct {
}

func (m ForceIntermission) Type() MessageCode { return N_FORCEINTERMISSION }

// N_FROMAI
type FromAI struct {
	Qcn int
}

func (m FromAI) Type() MessageCode { return N_FROMAI }

// N_GAMESPEED
type GameSpeed struct {
	Speed  int
	Client int
}

func (m GameSpeed) Type() MessageCode { return N_GAMESPEED }

// N_GETDEMO
type GetDemo struct {
	Demo int
	Tag  int
}

func (m GetDemo) Type() MessageCode { return N_GETDEMO }

// N_GETMAP
type GetMap struct {
}

func (m GetMap) Type() MessageCode { return N_GETMAP }

// N_ITEMPICKUP
type ItemPickup struct {
	Item int
}

func (m ItemPickup) Type() MessageCode { return N_ITEMPICKUP }

// N_KICK
type Kick struct {
	Victim int
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
	Crc int
}

func (m MapCRC) Type() MessageCode { return N_MAPCRC }

// N_MAPVOTE
type MapVote struct {
	Map  string
	Mode int
}

func (m MapVote) Type() MessageCode { return N_MAPVOTE }

// N_RECORDDEMO
type RecordDemo struct {
	Enabled int
}

func (m RecordDemo) Type() MessageCode { return N_RECORDDEMO }

// N_SENDMAP
type SendMap struct {
	Map []byte
}

func (m SendMap) Type() MessageCode { return N_SENDMAP }

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
	Client   int
	Master   int
	Password string
}

func (m SetMaster) Type() MessageCode { return N_SETMASTER }

// N_SHOOT
type Shoot struct {
	Id    int
	Gun   int
	From0 int
	From1 int
	From2 int
	To0   int
	To1   int
	To2   int
	Hits  []Hit `type:"count"`
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
	Model           int
	Password        string
	AuthDescription string
	AuthName        string
}

func (m Connect) Type() MessageCode { return N_CONNECT }

// N_SERVINFO
type ServerInfo struct {
	Client      int
	Protocol    int
	SessionId   int
	HasPassword int
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
	Id        int
	Challenge string
}

func (m AuthChallenge) Type() MessageCode { return N_AUTHCHAL }

// N_PONG
type Pong struct {
	Cmillis int
}

func (m Pong) Type() MessageCode { return N_PONG }

// N_PING
type Ping struct {
	Cmillis int
}

func (m Ping) Type() MessageCode { return N_PING }

// N_POS
type Pos struct {
	Client int
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
	Client int
}

func (m PauseGame) Type() MessageCode { return N_PAUSEGAME }

// N_TIMEUP
type TimeUp struct {
	Value int
}

func (m TimeUp) Type() MessageCode { return N_TIMEUP }

// N_ANNOUNCE
type Announce struct {
	Announcement int
}

func (m Announce) Type() MessageCode { return N_ANNOUNCE }

// N_MASTERMODE
type MasterMode struct {
	MasterMode int
}

func (m MasterMode) Type() MessageCode { return N_MASTERMODE }

// N_CDIS
type ClientDisconnected struct {
	Client int
}

func (m ClientDisconnected) Type() MessageCode { return N_CDIS }

// N_JUMPPAD
type JumpPad struct {
	Client  int
	JumpPad int
}

func (m JumpPad) Type() MessageCode { return N_JUMPPAD }

// N_TELEPORT
type Teleport struct {
	Client      int
	Source      int
	Destination int
}

func (m Teleport) Type() MessageCode { return N_TELEPORT }

// N_SPECTATOR
type Spectator struct {
	Client int
	Value  int
}

func (m Spectator) Type() MessageCode { return N_SPECTATOR }

// N_SETTEAM
type SetTeam struct {
	Client int
	Team   string
	Reason int
}

func (m SetTeam) Type() MessageCode { return N_SETTEAM }

// N_CURRENTMASTER
type CurrentMaster struct {
	Mastermode int
	Clients    []struct {
		Client    int
		Privilege int
	} `type:"term" cmp:"gez"`
}

func (m CurrentMaster) Type() MessageCode { return N_CURRENTMASTER }

// N_MAPCHANGE
type MapChange struct {
	Name     string
	Mode     int
	HasItems int
}

func (m MapChange) Type() MessageCode { return N_MAPCHANGE }

// N_TEAMINFO
type TeamInfo struct {
	Teams []struct {
		Team  string
		Frags int
	} `type:"term" cmp:"len"`
}

func (m TeamInfo) Type() MessageCode { return N_TEAMINFO }

// N_INITCLIENT
type InitClient struct {
	Client      int
	Name        string
	Team        string
	Playermodel int
}

func (m InitClient) Type() MessageCode { return N_INITCLIENT }

// N_SPAWNSTATE
type SpawnState struct {
	Client       int
	LifeSequence int
	Health       int
	MaxHealth    int
	Armour       int
	Armourtype   int
	Gunselect    int
	Ammo         []struct {
		Amount int
	} `type:"count" const:"6"`
}

func (m SpawnState) Type() MessageCode { return N_SPAWNSTATE }

type ClientState struct {
	Id           int
	State        int
	Frags        int
	Flags        int
	Deaths       int
	Quadmillis   int
	Lifesequence int
	Health       int
	Maxhealth    int
	Armour       int
	Armourtype   int
	Gunselect    int
	Ammo         []struct {
		Amount int
	} `type:"count" const:"6"`
}

// N_RESUME
type Resume struct {
	Clients []ClientState `type:"term" cmp:"gez"`
}

func (m Resume) Type() MessageCode { return N_RESUME }

// N_INITFLAGS
type InitFlags struct {
	Teamscores []struct {
		Score int
	} `type:"count" const:"2"`

	Flags []struct {
		Version   int
		Spawn     int
		Owner     int `type:"cond" cmp:"lz"`
		Invisible int
		Dropped   int `type:"cond" cmp:"nz"`
		Dx        int
		Dy        int
		Dz        int
	} `type:"count"`
}

func (m InitFlags) Type() MessageCode { return N_INITFLAGS }

// N_DROPFLAG
type DropFlag struct {
	Client  int
	Flag    int
	Version int
	Dx      int
	Dy      int
	Dz      int
}

func (m DropFlag) Type() MessageCode { return N_DROPFLAG }

// N_SCOREFLAG
type ScoreFlag struct {
	Client       int
	Relayflag    int
	Relayversion int
	Goalflag     int
	Goalversion  int
	Goalspawn    int
	Team         int
	Score        int
	Oflags       int
}

func (m ScoreFlag) Type() MessageCode { return N_SCOREFLAG }

// N_RETURNFLAG
type ReturnFlag struct {
	Client  int
	Flag    int
	Version int
}

func (m ReturnFlag) Type() MessageCode { return N_RETURNFLAG }

// N_TAKEFLAG
type TakeFlag struct {
	Client  int
	Flag    int
	Version int
}

func (m TakeFlag) Type() MessageCode { return N_TAKEFLAG }

// N_RESETFLAG
type ResetFlag struct {
	Flag    int
	Version int
	Spawn   int
	Team    int
	Score   int
}

func (m ResetFlag) Type() MessageCode { return N_RESETFLAG }

// N_INVISFLAG
type InvisFlag struct {
	Flag      int
	Invisible int
}

func (m InvisFlag) Type() MessageCode { return N_INVISFLAG }

// N_BASES
type Bases struct {
	Bases []struct {
		AmmoType  int
		Owner     string
		Enemy     string
		Converted int
		AmmoCount int
	} `type:"count"`
}

func (m Bases) Type() MessageCode { return N_BASES }

// N_BASEINFO
type BaseInfo struct {
	Base      int
	Owner     string
	Enemy     string
	Converted int
	Ammocount int
}

func (m BaseInfo) Type() MessageCode { return N_BASEINFO }

// N_BASESCORE
type BaseScore struct {
	Base  int
	Team  string
	Total int
}

func (m BaseScore) Type() MessageCode { return N_BASESCORE }

// N_REPAMMO
type ReplenishAmmo struct {
	Client   int
	Ammotype int
}

func (m ReplenishAmmo) Type() MessageCode { return N_REPAMMO }

// N_TRYSPAWN
type TrySpawn struct {
}

func (m TrySpawn) Type() MessageCode { return N_TRYSPAWN }

// N_BASEREGEN
type BaseRegen struct {
	Client   int
	Health   int
	Armour   int
	Ammotype int
	Ammo     int
}

func (m BaseRegen) Type() MessageCode { return N_BASEREGEN }

// N_INITTOKENS
type InitTokens struct {
	TeamScores []struct {
		Score int
	} `type:"count" const:"2"`

	Tokens []struct {
		Token int
		Team  int
		Yaw   int
		X     int
		Y     int
		Z     int
	} `type:"count"`

	ClientTokens []struct {
		Client int
		Count  int
	} `type:"term" cmp:"gez"`
}

func (m InitTokens) Type() MessageCode { return N_INITTOKENS }

// N_TAKETOKEN
type TakeToken struct {
	Client int
	Token  int
	Total  int
}

func (m TakeToken) Type() MessageCode { return N_TAKETOKEN }

// N_EXPIRETOKENS
type ExpireTokens struct {
	Tokens []struct {
		Token int
	} `type:"term" cmp:"gez"`
}

func (m ExpireTokens) Type() MessageCode { return N_EXPIRETOKENS }

// N_DROPTOKENS
type DropTokens struct {
	Client int
	Dropx  int
	Dropy  int
	Dropz  int
	Tokens []struct {
		Token int
		Team  int
		Yaw   int
	} `type:"term" cmp:"gez"`
}

func (m DropTokens) Type() MessageCode { return N_DROPTOKENS }

// N_STEALTOKENS
type StealTokens struct {
	Client    int
	Team      int
	Basenum   int
	Enemyteam int
	Score     int
	Dropx     int
	Dropy     int
	Dropz     int
	Tokens    []struct {
		Token int
		Team  int
		Yaw   int
	} `type:"term" cmp:"gez"`
}

func (m StealTokens) Type() MessageCode { return N_STEALTOKENS }

// N_DEPOSITTOKENS
type DepositTokens struct {
	Client    int
	Base      int
	Deposited int
	Team      int
	Score     int
	Flags     int
}

func (m DepositTokens) Type() MessageCode { return N_DEPOSITTOKENS }

// N_ITEMLIST
type ItemList struct {
	Items []struct {
		Index int
		Type  int
	} `type:"term" cmp:"gez"`
}

func (m ItemList) Type() MessageCode { return N_ITEMLIST }

// N_ITEMSPAWN
type ItemSpawn struct {
	Item_index int
}

func (m ItemSpawn) Type() MessageCode { return N_ITEMSPAWN }

// N_ITEMACC
type ItemAck struct {
	Item_index int
	Client     int
}

func (m ItemAck) Type() MessageCode { return N_ITEMACC }

// N_HITPUSH
type HitPush struct {
	Client int
	Gun    int
	Damage int
	Fx     int
	Fy     int
	Fz     int
}

func (m HitPush) Type() MessageCode { return N_HITPUSH }

// N_SHOTFX
type ShotFX struct {
	Client int
	Gun    int
	Id     int
	Fx     int
	Fy     int
	Fz     int
	Tx     int
	Ty     int
	Tz     int
}

func (m ShotFX) Type() MessageCode { return N_SHOTFX }

// N_EXPLODEFX
type ExplodeFX struct {
	Client int
	Gun    int
	Id     int
}

func (m ExplodeFX) Type() MessageCode { return N_EXPLODEFX }

// N_DAMAGE
type Damage struct {
	Client    int
	Aggressor int
	Damage    int
	Armour    int
	Health    int
}

func (m Damage) Type() MessageCode { return N_DAMAGE }

// N_DIED
type Died struct {
	Client      int
	Killer      int
	KillerFrags int
	VictimFrags int
}

func (m Died) Type() MessageCode { return N_DIED }

// N_FORCEDEATH
type ForceDeath struct {
	Client int
}

func (m ForceDeath) Type() MessageCode { return N_FORCEDEATH }

// N_REQAUTH
type ReqAuth struct {
	Domain string
}

func (m ReqAuth) Type() MessageCode { return N_REQAUTH }

// N_INITAI
type InitAI struct {
	Aiclientnum    int
	Ownerclientnum int
	Aitype         int
	Aiskill        int
	Playermodel    int
	Name           string
	Team           string
}

func (m InitAI) Type() MessageCode { return N_INITAI }

// N_SENDDEMOLIST
type SendDemoList struct {
	Demos []struct {
		Info string
	} `type:"count"`
}

func (m SendDemoList) Type() MessageCode { return N_SENDDEMOLIST }

// N_SENDDEMO
type SendDemo struct {
	Tag  int
	Data []byte
}

func (m SendDemo) Type() MessageCode { return N_SENDDEMO }

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
type ClientInfo struct {
	Client      int
	Nummessages int
}

func (m ClientInfo) Type() MessageCode { return N_CLIENT }

// N_SPAWN <- from server
type SpawnResponse struct {
	LifeSequence int
	Health       int
	MaxHealth    int
	Armour       int
	ArmourType   int
	GunSelect    int
	Ammo         []struct {
		Amount int
	} `type:"count" const:"6"`
}

func (m SpawnResponse) Type() MessageCode { return N_SPAWN }

// N_SPAWN <- from client
type SpawnRequest struct {
	LifeSequence int
	GunSelect    int
}

func (m SpawnRequest) Type() MessageCode { return N_SPAWN }

// N_SOUND
type Sound struct {
	Sound int
}

func (m Sound) Type() MessageCode { return N_SOUND }

// N_CLIENTPING
type ClientPing struct {
	Ping int
}

func (m ClientPing) Type() MessageCode { return N_CLIENTPING }

// N_TAUNT
type Taunt struct {
}

func (m Taunt) Type() MessageCode { return N_TAUNT }

// N_GUNSELECT
type GunSelect struct {
	GunSelect int
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
	Model int
}

func (m SwitchModel) Type() MessageCode { return N_SWITCHMODEL }

// N_EDITMODE
type EditMode struct {
	Value int
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
	registerBoth(&ClientInfo{})
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
	registerBoth(&InitFlags{})
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
