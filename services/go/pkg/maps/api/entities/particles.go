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

// Particles can have colors with a weird encoding scheme. Each element
// corresponds to the upper four bits in the corresponding element of a 24-bit
// RGBA color. Instead of messing with this, the Go API treats this as a normal
// 24-bit color and cuts off the 0x0F bits on encode.
type Color16 struct {
	R byte
	G byte
	B byte
}

func (c *Color16) Decode(a *Attributes) error {
	value, err := a.Get()
	if err != nil {
		return err
	}

	if value == 0 {
		return Empty
	}

	c.R = byte(((value >> 4) & 0xF0) + 0x0F)
	c.G = byte((value & 0xF0) + 0x0F)
	c.B = byte(((value << 4) & 0xF0) + 0x0F)
	return nil
}

var _ Decodable = (*Color16)(nil)

func (c Color16) Encode(a *Attributes) error {
	// Handle the default
	if c.R == 0x90 && c.G == 0x30 && c.B == 0x20 {
		a.Put(0)
		return nil
	}

	var value int16 = 0
	value += (int16(c.R) & 0xF0) << 4
	value += (int16(c.G) & 0xF0)
	value += (int16(c.B) & 0xF0) >> 4
	a.Put(value)
	return nil
}

var _ Encodable = (*Color16)(nil)

type Fire struct {
	Radius float32
	Height float32
	Color  Color16
}

func (p Fire) Defaults() Defaultable {
	radius := p.Radius
	fire := Fire{
		Radius: 1.5,
		Color: Color16{
			R: 0x90,
			G: 0x30,
			B: 0x20,
		},
	}

	if radius == 0 {
		radius = 1.5
	}

	fire.Height = radius / 3

	return fire
}

func (p *Fire) Type() ParticleType { return ParticleTypeFire }

var PARTICLE_TYPES = []ParticleInfo{
	&Fire{},
}

var PARTICLE_TYPE_MAP = map[ParticleType]ParticleInfo{}

func init() {
	for _, type_ := range PARTICLE_TYPES {
		PARTICLE_TYPE_MAP[type_.Type()] = type_
	}
}

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
	a.Put(int16(p.Particle))

	err := encodeValue(a, reflect.TypeOf(p.Data), reflect.ValueOf(p.Data))
	if err != nil {
		return err
	}

	return nil
}

var _ Encodable = (*Particles)(nil)
