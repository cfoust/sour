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

type EntityInfo interface {
	Type() C.EntityType
}

type Decodable interface {
	Decode(*Attributes)
}

type Encodable interface {
	Encode(*Attributes)
}

type BVector struct {
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
