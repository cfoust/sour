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

var ENTITY_TYPES = []EntityInfo{
	&Light{},
	&MapModel{},
	&PlayerStart{},
	&EnvMap{},
}

var ENTITY_TYPE_MAP = map[C.EntityType]EntityInfo{}

func init() {
	for _, type_ := range ENTITY_TYPES {
		ENTITY_TYPE_MAP[type_.Type()] = type_
	}
}
