package nmc

type ID int32 // network message code

const None ID = -1

const (
	TryJoin ID = iota // = CONNECT
	ServerInfo
	Welcome
	InitializeClient
	Position
	ChatMessage
	Sound
	Leave // = CDIS
	Shoot
	Explode
	Suicide // 10
	Died
	Damage
	HitPush
	ShotEffects
	ExplodeEffects
	TrySpawn
	SpawnState
	ConfirmSpawn
	ForceDeath
	ChangeWeapon // 20
	Taunt
	MapChange
	VoteMap
	TeamInfo
	PickupSpawn
	PickupTry
	PickupAck // = ITEMACC
	Teleport
	JumpPad
	Ping // 30
	Pong
	ClientPing
	TimeLeft // = TIMEUP
	ForceIntermission
	ServerMessage
	PickupList
	PlayerStateList // = RESUME
	EDITMODE
	EDITENT
	EDITF // 40
	EDITT
	EDITM
	FLIP
	COPY
	PASTE
	ROTATE
	REPLACE
	DELCUBE
	REMIP
	EDITVSLOT // 50
	UNDO
	REDO
	NEWMAP
	GETMAP
	SENDMAP
	CLIPBOARD
	EDITVAR
	MasterMode
	Kick
	ClearBans // 60
	CurrentMaster
	Spectator
	SetMaster
	SetTeam
	Bases
	BaseInfo
	BaseScore
	REPAMMO
	BASEREGEN
	ANNOUNCE // 70
	ListDemos
	SendDemoList
	GetDemo
	SendDemo
	DemoPlayback
	RecordDemo
	StopDemo
	ClearDemos
	TouchFlag
	ReturnFlag // 80
	ResetFlag
	InvisibleFlag
	TryDropFlag
	DropFlag
	ScoreFlag
	InitFlags
	TeamChatMessage
	Client
	AuthTry
	AuthKick // 90
	AuthChallenge
	AuthAnswer
	RequestAuth
	PauseGame
	GAMESPEED
	ADDBOT
	DELBOT
	INITAI
	FROMAI
	BOTLIMIT // 100
	BOTBALANCE
	MapCRC
	CHECKMAPS
	ChangeName  // = SWITCHNAME
	ChangeModel // = SWITCHMODEL
	ChangeTeam  // = SWITCHTEAM
	INITTOKENS
	TAKETOKEN
	EXPIRETOKENS
	DROPTOKENS // 110
	DEPOSITTOKENS
	STEALTOKENS
	ServerCommand
	DEMOPACKET
	//NUMMSG
)

// A list of NMCs which can only be sent by a server, never by a client.
var ServerOnlyNMCs = []ID{
	ServerInfo,
	InitializeClient,
	Welcome,
	MapChange,
	ServerMessage,
	Damage,
	HitPush,
	ShotEffects,
	ExplodeEffects,
	Died,
	SpawnState,
	ForceDeath,
	TeamInfo,
	PickupSpawn,
	PickupAck,
	TimeLeft,
	Leave,
	CurrentMaster,
	Pong,
	PlayerStateList,
	BaseScore,
	BaseInfo,
	BASEREGEN,
	ANNOUNCE,
	SendDemoList,
	SendDemo,
	DemoPlayback,
	SENDMAP,
	DropFlag,
	ScoreFlag,
	ReturnFlag,
	ResetFlag,
	InvisibleFlag,
	Client,
	AuthChallenge,
	INITAI,
	EXPIRETOKENS,
	DROPTOKENS,
	STEALTOKENS,
	DEMOPACKET,
}
