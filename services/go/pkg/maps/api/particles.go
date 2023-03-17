package api

type ParticleType byte

const (
	ParticleTypeFire      ParticleType = iota
	ParticleTypeSteamVent              = 1
	ParticleTypeFountain               = 2
	ParticleTypeFireball               = 3
	ParticleTypeTape                   = 4
	ParticleTypeLightning              = 7
	ParticleTypeSteam                  = 9
	ParticleTypeWater                  = 10
	ParticleTypeSnow                   = 13
	// your guess is as good as mine
	ParticleTypeMeter   = 5
	ParticleTypeMeterVS = 6
	ParticleTypeFlame   = 11
	ParticleTypeSmoke   = 12
	// i'm dying
	ParticleTypeLensFlare           = 32
	ParticleTypeLensFlareSparkle    = 33
	ParticleTypeLensFlareSun        = 34
	ParticleTypeLensFlareSparkleSun = 35
)

var PARTICLE_TYPE_MAP = map[ParticleType]string{
	ParticleTypeFire:                "fire",
	ParticleTypeSteamVent:           "steamVent",
	ParticleTypeFountain:            "fountain",
	ParticleTypeFireball:            "fireball",
	ParticleTypeTape:                "tape",
	ParticleTypeLightning:           "lightning",
	ParticleTypeSteam:               "steam",
	ParticleTypeWater:               "water",
	ParticleTypeSnow:                "snow",
	ParticleTypeMeter:               "meter",
	ParticleTypeMeterVS:             "meterVs",
	ParticleTypeFlame:               "flame",
	ParticleTypeSmoke:               "smoke",
	ParticleTypeLensFlare:           "lensFlare",
	ParticleTypeLensFlareSparkle:    "lensFlareSparkle",
	ParticleTypeLensFlareSun:        "lensFlareSun",
	ParticleTypeLensFlareSparkleSun: "lensFlareSparkleSun",
}

type ParticleData Attributable[ParticleType]

func (e ParticleType) String() string {
	value, ok := PARTICLE_TYPE_MAP[e]
	if !ok {
		return ""
	}
	return value
}

func (e ParticleType) FromString(value string) {
	for type_, key := range PARTICLE_TYPE_MAP {
		if key == value {
			e = type_
			return
		}
	}
	e = ParticleTypeFire
}

type Color16 BVector

type Fire struct {
	Radius int16
	Height int16
}
