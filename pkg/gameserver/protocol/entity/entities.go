package entity

type ID int32

const (
	NotUsed ID = iota
	LIGHT
	MAPMODEL
	PLAYERSTART
	ENVMAP
	PARTICLES
	MAPSOUND
	SPOTLIGHT
	PickupShotgun // 8
	PickupMinigun
	PickupRocketLauncher
	PickupRifle
	PickupGrenadeLauncher
	PickupPistol
	PickupHealth
	PickupBoost
	PickupGreenArmour
	PickupYellowArmor
	PickupQuadDamage
	Teleport
	Teledest
	Monster
	CARROT
	JUMPPAD
	BASE
	RESPAWNPOINT
	BOX
	BARREL
	PLATFORM
	ELEVATOR
	FLAG
	MAXENTTYPES
)
