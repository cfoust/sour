package entities

import (
	"fmt"
	"reflect"

	C "github.com/cfoust/sour/pkg/game/constants"
)

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

var PARTICLE_TYPE_STRINGS = map[ParticleType]string{
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

func (e ParticleType) String() string {
	value, ok := PARTICLE_TYPE_STRINGS[e]
	if !ok {
		return ""
	}
	return value
}

func (e ParticleType) FromString(value string) {
	for type_, key := range PARTICLE_TYPE_STRINGS {
		if key == value {
			e = type_
			return
		}
	}
	e = ParticleTypeFire
}

type ParticleInfo interface {
	Type() ParticleType
}

type Particles struct {
	Particle ParticleType
	Data     ParticleInfo
}

func (p *Particles) Type() C.EntityType { return C.EntityTypeParticles }

type Fire struct {
	Radius int16
	Height int16
}

func (p *Fire) Type() ParticleType { return ParticleTypeFire }

var PARTICLE_TYPES = []ParticleInfo{
	&Fire{},
}

var PARTICLE_TYPE_MAP = map[ParticleType]ParticleInfo{}

func (p *Particles) Decode(a *Attributes) error {
	type_, err := a.Get()
	if err != nil {
		return err
	}

	particleType, ok := PARTICLE_TYPE_MAP[ParticleType(type_)]
	if !ok {
		return fmt.Errorf("unknown particle type %d", particleType)
	}

	p.Particle = ParticleType(type_)

	decodedType := reflect.TypeOf(particleType)
	decoded := reflect.New(decodedType.Elem())
	err = decodeValue(a, decodedType.Elem(), decoded)
	if err != nil {
		return err
	}

	if value, ok := decoded.Interface().(ParticleInfo); ok {
		p.Data = value
	} else {
		return fmt.Errorf("could not decode particle info")
	}

	return nil
}

var _ Decodable = (*Particles)(nil)

func (p *Particles) Encode(a *Attributes) error {
	return nil
}

var _ Encodable = (*Particles)(nil)
