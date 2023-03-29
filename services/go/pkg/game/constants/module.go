package constants

import (
	"github.com/repeale/fp-go/option"
)

const MAP_VERSION = 33

const MAXENTS = 10000

// network quantization scale
const DMF = 16.0  // for world locations
const DNF = 100.0 // for normalized vectors
const DVELF = 1.0 // for playerspeed based velocity vectors

const MAXSTRLEN = 260

// "services/game/src/shared/ents.h" line 91
const DEFAULT_EYE_HEIGHT = 14

const (
	GUN_FIST uint16 = 0
	GUN_SG
	GUN_CG
	GUN_RL
	GUN_RIFLE
	GUN_GL
	GUN_PISTOL
	GUN_FIREBALL
	GUN_ICEBALL
	GUN_SLIMEBALL
	GUN_BITE
	GUN_BARREL
	NUMGUNS
)

const (
	M_TEAM       int = 1 << 0
	M_NOITEMS        = 1 << 1
	M_NOAMMO         = 1 << 2
	M_INSTA          = 1 << 3
	M_EFFICIENCY     = 1 << 4
	M_TACTICS        = 1 << 5
	M_CAPTURE        = 1 << 6
	M_REGEN          = 1 << 7
	M_CTF            = 1 << 8
	M_PROTECT        = 1 << 9
	M_HOLD           = 1 << 10
	M_EDIT           = 1 << 12
	M_DEMO           = 1 << 13
	M_LOCAL          = 1 << 14
	M_LOBBY          = 1 << 15
	M_DMSP           = 1 << 16
	M_CLASSICSP      = 1 << 17
	M_SLOWMO         = 1 << 18
	M_COLLECT        = 1 << 19
)

const (
	MODE_FFA int32 = iota
	MODE_COOP
	MODE_TEAMPLAY
	MODE_INSTA
	MODE_INSTATEAM
	MODE_EFFIC
	MODE_EFFICTEAM
	MODE_TAC
	MODE_TACTEAM
	MODE_CAPTURE
	MODE_REGENCAPTURE
	MODE_CTF
	MODE_INSTACTF
	MODE_PROTECT
	MODE_INSTAPROTECT
	MODE_HOLD
	MODE_INSTAHOLD
	MODE_EFFICCTF
	MODE_EFFICPROTECT
	MODE_EFFICHOLD
	MODE_COLLECT
	MODE_INSTACOLLECT
	MODE_EFFICCOLLECT
)

var MODE_NAMES = []string{
	"ffa", "coop", "teamplay", "insta", "instateam", "effic", "efficteam", "tac", "tacteam", "capture", "regencapture", "ctf", "instactf", "protect", "instaprotect", "hold", "instahold", "efficctf", "efficprotect", "effichold", "collect", "instacollect", "efficcollect",
}

func GetModeNumber(mode string) opt.Option[int] {
	for i, name := range MODE_NAMES {
		if name == mode {
			return opt.Some(i)
		}
	}

	return opt.None[int]()
}

type EntityType byte

const (
	EntityTypeEmpty        EntityType = iota // ET_EMPTY
	EntityTypeLight                          // ET_LIGHT attr1 = radius, attr2 = intensity
	EntityTypeMapModel                       // ET_MAPMODEL attr1 = angle, attr2 = idx
	EntityTypePlayerStart                    // ET_PLAYERSTART attr1 = angle, attr2 = team
	EntityTypeEnvMap                         // ET_ENVMAP attr1 = radius, attr2 = size, attr3 = blur
	EntityTypeParticles                      // ET_PARTICLES
	EntityTypeSound                          // ET_SOUND
	EntityTypeSpotlight                      // ET_SPOTLIGHT attr1=angle
	EntityTypeShells                         // I_SHELLS
	EntityTypeBullets                        // I_BULLETS
	EntityTypeRockets                        // I_ROCKETS
	EntityTypeRounds                         // I_ROUNDS
	EntityTypeGrenades                       // I_GRENADES
	EntityTypeCartridges                     // I_CARTRIDGES
	EntityTypeHealth                         // I_HEALTH
	EntityTypeBoost                          // I_BOOST
	EntityTypeGreenArmour                    // I_GREENARMOUR
	EntityTypeYellowArmour                   // I_YELLOWARMOUR
	EntityTypeQuad                           // I_QUAD
	EntityTypeTeleport                       // TELEPORT attr1 = idx, attr2 = model, attr3 = tag
	EntityTypeTeledest                       // TELEDEST attr1 = angle, attr2 = idx
	EntityTypeMonster                        // MONSTER attr1 = angle, attr2 = monstertype
	EntityTypeCarrot                         // CARROT attr1 = tag, attr2 = type
	EntityTypeJumpPad                        // JUMPPAD attr1 = zpush, attr2 = ypush, attr3 = xpush
	EntityTypeBase                           // BASE
	EntityTypeRespawnPoint                   // RESPAWNPOINT
	EntityTypeBox                            // BOX attr1 = angle, attr2 = idx, attr3 = weight
	EntityTypeBarrel                         // BARREL attr1 = angle, attr2 = idx, attr3 = weight, attr4 = health
	EntityTypePlatform                       // PLATFORM attr1 = angle, attr2 = idx, attr3 = tag, attr4 = speed
	EntityTypeElevator                       // ELEVATOR attr1 = angle, attr2 = idx, attr3 = tag, attr4 = speed
	EntityTypeFlag                           // FLAG attr1 = angle, attr2 = team
)

var ENTITY_TYPE_MAP = map[EntityType]string{
	EntityTypeEmpty:        "empty",
	EntityTypeLight:        "light",
	EntityTypeMapModel:     "mapmodel",
	EntityTypePlayerStart:  "playerstart",
	EntityTypeEnvMap:       "envmap",
	EntityTypeParticles:    "particles",
	EntityTypeSound:        "sound",
	EntityTypeSpotlight:    "spotlight",
	EntityTypeShells:       "shells",
	EntityTypeBullets:      "bullets",
	EntityTypeRockets:      "rockets",
	EntityTypeRounds:       "rounds",
	EntityTypeGrenades:     "grenades",
	EntityTypeCartridges:   "cartridges",
	EntityTypeHealth:       "health",
	EntityTypeBoost:        "boost",
	EntityTypeGreenArmour:  "greenarmour",
	EntityTypeYellowArmour: "yellowarmour",
	EntityTypeQuad:         "quad",
	EntityTypeTeleport:     "teleport",
	EntityTypeTeledest:     "teledest",
	EntityTypeMonster:      "monster",
	EntityTypeCarrot:       "carrot",
	EntityTypeJumpPad:      "jumppad",
	EntityTypeBase:         "base",
	EntityTypeRespawnPoint: "respawnpoint",
	EntityTypeBox:          "box",
	EntityTypeBarrel:       "barrel",
	EntityTypePlatform:     "platform",
	EntityTypeElevator:     "elevator",
	EntityTypeFlag:         "flag",
}

func (e EntityType) String() string {
	value, ok := ENTITY_TYPE_MAP[e]
	if !ok {
		return ""
	}
	return value
}

func (e EntityType) FromString(value string) {
	for type_, key := range ENTITY_TYPE_MAP {
		if key == value {
			e = type_
			return
		}
	}
	e = EntityTypeEmpty
}

const DEMO_VERSION = 1
const DEMO_MAGIC = "SAUERBRATEN_DEMO"
