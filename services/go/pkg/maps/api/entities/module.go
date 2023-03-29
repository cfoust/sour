package entities

import (
	C "github.com/cfoust/sour/pkg/game/constants"
)

type EntityInfo interface {
	Type() C.EntityType
}

type ByteVector struct {
	X byte
	Y byte
	Z byte
}

type Color struct {
	R byte
	G byte
	B byte
}

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Entity struct {
	Position Vector
	Info     EntityInfo
}

type Light struct {
	Radius int16
	Color  Color
}

func (e *Light) Type() C.EntityType { return C.EntityTypeLight }

type MapModel struct {
	Angle int16
	Index int16
}

func (m *MapModel) Type() C.EntityType { return C.EntityTypeMapModel }

type PlayerStart struct {
	Angle int16
	Tag   int16
}

func (e *PlayerStart) Type() C.EntityType { return C.EntityTypePlayerStart }

type EnvMap struct {
	Radius int16
	Size   int16
	Blur   int16
}

func (e *EnvMap) Type() C.EntityType { return C.EntityTypeEnvMap }

type Sound struct {
	Index int16
}

func (e *Sound) Type() C.EntityType { return C.EntityTypeSound }

type Spotlight struct {
	Radius int16
	Color  Color
}

func (e *Spotlight) Type() C.EntityType { return C.EntityTypeSpotlight }

type Powerup struct{}
type Shells Powerup
type Bullets Powerup
type Rockets Powerup
type Rounds Powerup
type Grenades Powerup
type Cartridges Powerup
type Health Powerup
type Boost Powerup
type GreenArmour Powerup
type YellowArmour Powerup
type Quad Powerup

func (e *Shells) Type() C.EntityType       { return C.EntityTypeShells }
func (e *Bullets) Type() C.EntityType      { return C.EntityTypeBullets }
func (e *Rockets) Type() C.EntityType      { return C.EntityTypeRockets }
func (e *Rounds) Type() C.EntityType       { return C.EntityTypeRounds }
func (e *Grenades) Type() C.EntityType     { return C.EntityTypeGrenades }
func (e *Cartridges) Type() C.EntityType   { return C.EntityTypeCartridges }
func (e *Health) Type() C.EntityType       { return C.EntityTypeHealth }
func (e *Boost) Type() C.EntityType        { return C.EntityTypeBoost }
func (e *GreenArmour) Type() C.EntityType  { return C.EntityTypeGreenArmour }
func (e *YellowArmour) Type() C.EntityType { return C.EntityTypeYellowArmour }
func (e *Quad) Type() C.EntityType         { return C.EntityTypeQuad }

var ENTITY_TYPES = []EntityInfo{
	&Light{},
	&MapModel{},
	&PlayerStart{},
	&EnvMap{},
	&Particles{},
	&Sound{},
	&Spotlight{},
	&Shells{},
	&Bullets{},
	&Rockets{},
	&Rounds{},
	&Grenades{},
	&Cartridges{},
	&Health{},
	&Boost{},
	&GreenArmour{},
	&YellowArmour{},
	&Quad{},
}

var ENTITY_TYPE_MAP = map[C.EntityType]EntityInfo{}

func init() {
	for _, type_ := range ENTITY_TYPES {
		ENTITY_TYPE_MAP[type_.Type()] = type_
	}
}
