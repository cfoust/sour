package entities

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/utils"
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

func (c Color16) MarshalJSON() ([]byte, error) {
	var color uint32
	c.truncateColors()
	color = color | (uint32(c.R) << 16)
	color = color | (uint32(c.G) << 8)
	color = color | uint32(c.B)
	return json.Marshal(fmt.Sprintf("#%06x", color))
}

func (c *Color16) truncateColors() {
	c.R = (c.R & 0xF0) + 0x0F
	c.G = (c.G & 0xF0) + 0x0F
	c.B = (c.B & 0xF0) + 0x0F
}

func (c *Color16) UnmarshalJSON(data []byte) error {
	var hex string
	err := json.Unmarshal(data, &hex)
	if err == nil {
		color, err := strconv.ParseUint(hex[1:], 16, 32)
		if err != nil {
			return err
		}

		c.R = byte((color >> 16) & 0xFF)
		c.G = byte((color >> 8) & 0xFF)
		c.B = byte(color & 0xFF)
		c.truncateColors()
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	elements := [3]byte{}
	err = json.Unmarshal(data, &elements)
	if err == nil {
		c.R = elements[0]
		c.G = elements[1]
		c.B = elements[2]
		c.truncateColors()
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	full := struct {
		R byte
		G byte
		B byte
	}{}
	err = json.Unmarshal(data, &full)
	if err == nil {
		c.R = full.R
		c.G = full.G
		c.B = full.B
		c.truncateColors()
		return nil
	}
	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		return err
	}

	return fmt.Errorf("could not deserialize color")
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

type Direction byte

const (
	DirectionZ Direction = iota
	DirectionX
	DirectionY
	DirectionNegZ
	DirectionNegX
	DirectionNegY
)

func (d *Direction) Decode(a *Attributes) error {
	value, err := a.Get()
	if err != nil {
		return err
	}

	*d = Direction(value)
	return nil
}

var _ Decodable = (*Direction)(nil)

func (d *Direction) Encode(a *Attributes) error {
	a.Put(int16(*d))
	return nil
}

var _ Encodable = (*Direction)(nil)

type Basic struct {
	Radius float32 `json:"radius"`
	Height float32 `json:"height"`
	Color  Color16 `json:"color"`
}

type Fire Basic

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

type SteamVent struct {
	Direction Direction
}

func (p *SteamVent) Type() ParticleType { return ParticleTypeSteamVent }

type Fountain struct {
	Direction Direction
	// TODO handle material colors
	Color Color16
}

func (p *Fountain) Type() ParticleType { return ParticleTypeFountain }

type Fireball struct {
	Size  uint16  `json:"size"`
	Color Color16 `json:"color"`
}

func (p *Fireball) Type() ParticleType { return ParticleTypeFireball }

type Shape struct {
	// TODO handle all the fancy shapes
	// This is NOT the same thing as Direction above, it's used to
	// parametrize the size of particles
	Direction uint16  `json:"direction"`
	Radius    uint16  `json:"radius"`
	Color     Color16 `json:"color"`
	Fade      uint16  `json:"fade"` // ms, if 0, 200ms
}

type Tape Shape

func (p *Tape) Type() ParticleType { return ParticleTypeTape }

type Lightning Shape

func (p *Lightning) Type() ParticleType { return ParticleTypeLightning }

type Steam Shape

func (p *Steam) Type() ParticleType { return ParticleTypeSteam }

type Water Shape

func (p *Water) Type() ParticleType { return ParticleTypeWater }

type Snow Shape

func (p *Snow) Type() ParticleType { return ParticleTypeSnow }

type Meter struct {
	Progress uint16  `json:"progress"`
	ColorA   Color16 `json:"colorA"`
	ColorB   Color16 `json:"colorB"`
}

func (p *Meter) Type() ParticleType { return ParticleTypeMeter }

type MeterVS Meter

func (p *MeterVS) Type() ParticleType { return ParticleTypeMeterVS }

// how is this different from Fire?
type Flame Basic

func (p *Flame) Type() ParticleType { return ParticleTypeFlame }

type SmokePlume Basic

func (p *SmokePlume) Type() ParticleType { return ParticleTypeSmoke }

type LensFlare struct {
	Color utils.Color `json:"color"`
}

func (p *LensFlare) Type() ParticleType { return ParticleTypeLensFlare }

type LensFlareSparkle LensFlare

func (p *LensFlareSparkle) Type() ParticleType { return ParticleTypeLensFlareSparkle }

type LensFlareSun LensFlare

func (p *LensFlareSun) Type() ParticleType { return ParticleTypeLensFlareSun }

type LensFlareSparkleSun LensFlare

func (p *LensFlareSparkleSun) Type() ParticleType { return ParticleTypeLensFlareSparkleSun }

var PARTICLE_TYPES = []ParticleInfo{
	&Fire{},
	&SteamVent{},
	&Fountain{},
	&Tape{},
	&Lightning{},
	&Steam{},
	&Water{},
	&Snow{},
	&Meter{},
	&MeterVS{},
	&Flame{},
	&SmokePlume{},
	&LensFlareSparkleSun{},
	&LensFlareSparkle{},
	&LensFlareSun{},
	&LensFlare{},
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
