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

type EntityData interface {
	Type() C.EntityType
	Encode() Attributes
	Decode(Attributes)
}

type Entity struct {
	Position Vector
	Data     EntityData
}

type Light struct {
	Radius int16
	Color  BVector
}

func (l *Light) Type() C.EntityType {
	return C.EntityTypeLight
}

func (l *Light) Encode() Attributes {
	return Attributes{
		Attr1: l.Radius,
		Attr2: int16(l.Color.X),
		Attr3: int16(l.Color.Y),
		Attr4: int16(l.Color.Z),
	}
}

func (l *Light) Decode(a Attributes) {
	l.Radius = a.Attr1
	l.Color.X = byte(a.Attr2)
	l.Color.Y = byte(a.Attr3)
	l.Color.Z = byte(a.Attr4)
}

var _ EntityData = (*Light)(nil)
