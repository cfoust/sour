package game

// N_SERVINFO
type ServerInfo struct {
	Client           int
	Protocol_version int
	Sessionid        int
	Haspwd           int
	Description      string
	Domain           string
}

// N_WELCOME
type Welcome struct {
}

// N_AUTHCHAL
type AuthChallenge struct {
	Desc      string
	Auth_id   int
	Challenge string
}

// N_PONG
type Pong struct {
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
	Value int
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
	Aiclientnum int
	Jumppad     int
}

// N_TELEPORT
type Teleport struct {
	Aiclientnum int
	Teledest    int
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
	Clients []struct {
		State        int
		Frags        int
		Flags        int
		Quadmillis   int
		Client       int
		Lifesequence int
		Health       int
		Maxhealth    int
		Armour       int
		Armourtype   int
		Gunselect    int
		Ammo         []struct {
			Amount int
		} `type:"count" const:"6"`
	} `type:"term" cmp:"gez"`
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

// N_CLIPBOARD
type Clipboard struct {
	Client    int
	UnpackLen int
	Data      []byte `type:"count"`
}

// N_EDITF
type Editf struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
	Dir        int
	Mode       int
}

// N_EDITT
type Editt struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
	Tex        int
	Allfaces   int
}

// N_EDITM
type Editm struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
	Mat        int
	Filter     int
}

// N_FLIP
type Flip struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
}

// N_COPY
type Copy struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
}

// N_PASTE
type Paste struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
}

// N_ROTATE
type Rotate struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
	Dir        int
}

// N_REPLACE
type Replace struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
	Tex        int
	Newtex     int
	Insel      int
}

// N_DELCUBE
type Delcube struct {
	Sel_ox     int
	Sel_oy     int
	Sel_oz     int
	Sel_sx     int
	Sel_sy     int
	Sel_sz     int
	Sel_grid   int
	Sel_orient int
	Sel_cx     int
	Sel_cxs    int
	Sel_cy     int
	Sel_cys    int
	Sel_corner int
}

// N_REMIP
type Remip struct {
}

// N_EDITENT
type EditEnt struct {
	Entid int
	X     int
	Y     int
	Z     int
	Type  int
	Attr1 int
	Attr2 int
	Attr3 int
	Attr4 int
	Attr5 int
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
	Client       int
	Killer       int
	Frags        int
	Victim_frags int
}

// N_FORCEDEATH
type ForceDeath struct {
	Client int
}

// N_NEWMAP
type NewMap struct {
	Size int
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
}

// N_CLIENT
type ClientInfo struct {
	Client      int
	Nummessages int
}

const GUN_COUNT = GUN_PISTOL - GUN_SG + 1

// N_SPAWN
type Spawn struct {
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
	Gunselect int
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
