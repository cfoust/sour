package api

import (
	C "github.com/cfoust/sour/pkg/game/constants"
)

type Attributes struct {
	Attr1 int16
	Attr2 int16
	Attr3 int16
	Attr4 int16
	Attr5 int16
}

type Attributable[A Typable] interface {
	Type() A
	Encode() Attributes
	Decode(Attributes)
}

type EntityData Attributable[C.EntityType]

type Entity struct {
	Position Vector
	Data     EntityData
}

type Light struct {
	Radius int16
	Color  Color
}

func (l *Light) Type() C.EntityType {
	return C.EntityTypeLight
}

func (l *Light) Encode() Attributes {
	return Attributes{
		Attr1: l.Radius,
		Attr2: int16(l.Color.R),
		Attr3: int16(l.Color.G),
		Attr4: int16(l.Color.B),
	}
}

func (l *Light) Decode(a Attributes) {
	l.Radius = a.Attr1
	l.Color.R = byte(a.Attr2)
	l.Color.G = byte(a.Attr3)
	l.Color.B = byte(a.Attr4)
}

var _ EntityData = (*Light)(nil)

type MapModel struct {
	Angle int16
	Index int16
}

func (m *MapModel) Type() C.EntityType {
	return C.EntityTypeMapModel
}

func (m *MapModel) Encode() Attributes {
	return Attributes{
		Attr1: m.Angle,
		Attr2: m.Index,
	}
}

func (m *MapModel) Decode(a Attributes) {
	m.Angle = a.Attr1
	m.Index = a.Attr2
}

var _ EntityData = (*MapModel)(nil)

type PlayerStart struct {
	Angle int16
	Tag   int16
}

func (e *PlayerStart) Type() C.EntityType {
	return C.EntityTypePlayerStart
}

func (e *PlayerStart) Encode() Attributes {
	return Attributes{
		Attr1: e.Angle,
		Attr2: e.Tag,
	}
}

func (e *PlayerStart) Decode(a Attributes) {
	e.Angle = a.Attr1
	e.Tag = a.Attr2
}

var _ EntityData = (*PlayerStart)(nil)

type EnvMap struct {
	Radius int16
	Size   int16
	Blur   int16
}

func (e *EnvMap) Type() C.EntityType {
	return C.EntityTypeEnvMap
}

func (e *EnvMap) Encode() Attributes {
	return Attributes{
		Attr1: e.Radius,
		Attr2: e.Size,
		Attr3: e.Blur,
	}
}

func (e *EnvMap) Decode(a Attributes) {
	e.Radius = a.Attr1
	e.Size = a.Attr2
	e.Blur = a.Attr3
}

var _ EntityData = (*EnvMap)(nil)

type Particles struct {
	Data ParticleData
}

func (e *Particles) Type() C.EntityType {
	return C.EntityTypeParticles
}

func (e *Particles) Encode() Attributes {
	attributes := e.Data.Encode()
	// Override the particle type
	attributes.Attr1 = int16(e.Data.Type())
	return attributes
}

func (e *Particles) Decode(a Attributes) {
	type_ := a.Attr1
	var data ParticleData
	switch type_ {
	}
	e.Data.Decode(a)
}

var _ EntityData = (*Particles)(nil)
