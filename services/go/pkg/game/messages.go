package game

// N_ADDBOT
type AddBot struct {
	NumBots int
}

// N_AUTHANS
type AuthAns struct {
	Description string
	Id          int
	Answer      string
}

// N_AUTHKICK
type AuthKick struct {
	Description string
	Answer      string
	Victim      int
}

// N_AUTHTRY
type AuthTry struct {
	Description string
	Answer      string
}

// N_BOTBALANCE
type BotBalance struct {
	Balance int
}

// N_BOTLIMIT
type BotLimit struct {
	Limit int
}

// N_CHECKMAPS
type CheckMaps struct {
}

// N_CLEARBANS
type ClearBans struct {
}

// N_CLEARDEMOS
type ClearDemos struct {
	Demo int
}

// N_DELBOT
type DelBot struct {
}

// N_DEMOPACKET
type DemoPacket struct {
}

// N_DEMOPLAYBACK
type DemoPlayback struct {
	On     int
	Client int
}

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

// N_FORCEINTERMISSION
type ForceIntermission struct {
}

// N_FROMAI
type FromAI struct {
	Qcn int
}

// N_GAMESPEED
type GameSpeed struct {
	Speed  int
	Client int
}

// N_GETDEMO
type GetDemo struct {
	Demo int
	Tag  int
}

// N_GETMAP
type GetMap struct {
}

// N_ITEMPICKUP
type ItemPickup struct {
	Item int
}

// N_KICK
type Kick struct {
	Victim int
	Reason string
}

// N_LISTDEMOS
type ListDemos struct {
}

// N_MAPCRC
type MapCRC struct {
	Map string
	Crc int
}

// N_MAPVOTE
type MapVote struct {
	Map  string
	Mode int
}

// N_RECORDDEMO
type RecordDemo struct {
	Enabled int
}

// N_SENDMAP
type SendMap struct {
	Map []byte
}

func (s *SendMap) Unmarshal(p *Packet) error {
	s.Map = *p
	*p = (*p)[0:0]
	return nil
}

// N_SERVCMD
type ServCMD struct {
	Command string
}

// N_SETMASTER
type SetMaster struct {
	Client   int
	Master   int
	Password string
}

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

// N_STOPDEMO
type StopDemo struct {
}

// N_SUICIDE
type Suicide struct {
}

// N_SWITCHTEAM
type SwitchTeam struct {
	Team string
}

// N_TRYDROPFLAG
type TryDropFlag struct {
}

// N_CONNECT
type Connect struct {
	Name            string
	Model           int
	Password        string
	AuthDescription string
	AuthName        string
}

// N_SERVINFO
type ServerInfo struct {
	Client      int
	Protocol    int
	SessionId   int
	HasPassword int
	Description string
	Domain      string
}

// N_WELCOME
type Welcome struct {
}

// N_AUTHCHAL
type AuthChallenge struct {
	Desc      string
	Id        int
	Challenge string
}

// N_PONG
type Pong struct {
	Cmillis int
}

// N_PING
type Ping struct {
	Cmillis int
}

// N_POS
type Pos struct {
	Client int
	State  PhysicsState
}

// N_SERVMSG
type ServerMessage struct {
	Text string
}

// N_PAUSEGAME
type PauseGame struct {
	Paused bool
	Client int
}

// N_TIMEUP
type TimeUp struct {
	Value int
}

// N_ANNOUNCE
type Announce struct {
	Type int
}

// N_MASTERMODE
type MasterMode struct {
	MasterMode int
}

// N_CDIS
type ClientDisconnected struct {
	Client int
}

// N_JUMPPAD
type JumpPad struct {
	Client  int
	JumpPad int
}

// N_TELEPORT
type Teleport struct {
	Client      int
	Source      int
	Destination int
}

// N_SPECTATOR
type Spectator struct {
	Client int
	Value  int
}

// N_SETTEAM
type SetTeam struct {
	Client int
	Team   string
	Reason int
}

// N_CURRENTMASTER
type CurrentMaster struct {
	Mastermode int
	Clients    []struct {
		Client    int
		Privilege int
	} `type:"term" cmp:"gez"`
}

// N_MAPCHANGE
type MapChange struct {
	Name     string
	Mode     int
	HasItems int
}

// N_TEAMINFO
type TeamInfo struct {
	Teams []struct {
		Team  string
		Frags int
	} `type:"term" cmp:"len"`
}

// N_INITCLIENT
type InitClient struct {
	Client      int
	Name        string
	Team        string
	Playermodel int
}

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

// N_DROPFLAG
type DropFlag struct {
	Client  int
	Flag    int
	Version int
	Dx      int
	Dy      int
	Dz      int
}

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

// N_RETURNFLAG
type ReturnFlag struct {
	Client  int
	Flag    int
	Version int
}

// N_TAKEFLAG
type TakeFlag struct {
	Client  int
	Flag    int
	Version int
}

// N_RESETFLAG
type ResetFlag struct {
	Flag    int
	Version int
	Spawn   int
	Team    int
	Score   int
}

// N_INVISFLAG
type InvisFlag struct {
	Flag      int
	Invisible int
}

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

// N_BASEINFO
type BaseInfo struct {
	Base      int
	Owner     string
	Enemy     string
	Converted int
	Ammocount int
}

// N_BASESCORE
type BaseScore struct {
	Base  int
	Team  string
	Total int
}

// N_REPAMMO
type ReplenishAmmo struct {
	Client   int
	Ammotype int
}

// N_TRYSPAWN
type TrySpawn struct {
}

// N_BASEREGEN
type BaseRegen struct {
	Client   int
	Health   int
	Armour   int
	Ammotype int
	Ammo     int
}

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

// N_TAKETOKEN
type TakeToken struct {
	Client int
	Token  int
	Total  int
}

// N_EXPIRETOKENS
type ExpireTokens struct {
	Tokens []struct {
		Token int
	} `type:"term" cmp:"gez"`
}

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

// N_DEPOSITTOKENS
type DepositTokens struct {
	Client    int
	Base      int
	Deposited int
	Team      int
	Score     int
	Flags     int
}

// N_ITEMLIST
type ItemList struct {
	Items []struct {
		Index int
		Type  int
	} `type:"term" cmp:"gez"`
}

// N_ITEMSPAWN
type ItemSpawn struct {
	Item_index int
}

// N_ITEMACC
type ItemAck struct {
	Item_index int
	Client     int
}

// N_HITPUSH
type HitPush struct {
	Client int
	Gun    int
	Damage int
	Fx     int
	Fy     int
	Fz     int
}

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

// N_EXPLODEFX
type ExplodeFX struct {
	Client int
	Gun    int
	Id     int
}

// N_DAMAGE
type Damage struct {
	Client    int
	Aggressor int
	Damage    int
	Armour    int
	Health    int
}

// N_DIED
type Died struct {
	Client      int
	Killer      int
	KillerFrags int
	VictimFrags int
}

// N_FORCEDEATH
type ForceDeath struct {
	Client int
}

// N_REQAUTH
type ReqAuth struct {
	Domain string
}

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

// N_SENDDEMOLIST
type SendDemoList struct {
	Demos []struct {
		Info string
	} `type:"count"`
}

// N_SENDDEMO
type SendDemo struct {
	Tag  int
	Data []byte
}

func (s *SendDemo) Unmarshal(p *Packet) error {
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

// N_SPAWN <- from client
type SpawnRequest struct {
	LifeSequence int
	GunSelect    int
}

// N_SOUND
type Sound struct {
	Sound int
}

// N_CLIENTPING
type ClientPing struct {
	Ping int
}

// N_TAUNT
type Taunt struct {
}

// N_GUNSELECT
type GunSelect struct {
	GunSelect int
}

// N_TEXT
type Text struct {
	Text string
}

// N_SAYTEAM
type SayTeam struct {
	Text string
}

// N_SWITCHNAME
type SwitchName struct {
	Name string
}

// N_SWITCHMODEL
type SwitchModel struct {
	Model int
}

// N_EDITMODE
type EditMode struct {
	Value int
}
