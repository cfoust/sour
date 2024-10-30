package sound

type ID int32

const (
	Jump ID = iota
	Land
	Rifle
	Saw
	Shotgun
	Minigun
	RocketLaunch
	RocketHit
	Weaponload
	PickUpAmmo
	PickUpHealth
	PickUpArmour
	PickUpQuaddamage
	ItemSpawn
	Teleport
	NoAmmo
	QuaddamageOver
	Pain1
	Pain2
	Pain3
	Pain4
	Pain5
	Pain6
	Die1
	Die2
	GrenadeLaunch
	GrenadeExplode
	Splash1 // start singleplayer-only sound
	Splash2
	Grunt1
	Grunt2
	Rumble
	PainO
	PainR
	DeathR
	PainE
	DeathE
	PainS
	DeathS
	PainB
	DeathB
	PainP
	PigGrunt2
	PainH
	DeathH
	PainD
	DeathD
	PigGrunt1
	IceBall
	SlimeBall // end singleplayer-only sounds
	JumpPad
	Pistol
	BaseCaptured // start voice sounds
	BaseLost
	Fight
	HealthBoost
	HealthBoostIn10Seconds
	Quaddamage
	QuaddamageIn10Seconds
	RespawnPointSet // end voice sounds
	FlagPickup
	FlagDrop
	FlagReturn
	FlagScore
	FlagReset
	Burn
	ChainSawAttack
	ChainSawIdle
	Hit
	FlagFail
)
