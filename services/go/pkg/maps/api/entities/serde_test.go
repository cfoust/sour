package entities

import (
	"testing"

	C "github.com/cfoust/sour/pkg/game/constants"

	"github.com/stretchr/testify/assert"
)

func TestMapModel(t *testing.T) {
	a := Attributes{}
	a.Put(1)
	a.Put(2)

	info, err := DecodeEntity(C.EntityTypeMapModel, &a)
	assert.NoError(t, err)
	assert.Equal(t, info, &MapModel{
		Angle: 1,
		Index: 2,
	})
}

func TestLight(t *testing.T) {
	a := Attributes{}
	a.Put(8)
	// The color
	a.Put(0)
	a.Put(255)
	a.Put(128)

	info, err := DecodeEntity(C.EntityTypeLight, &a)
	assert.NoError(t, err)
	assert.Equal(t, info, &Light{
		Radius: 8,
		Color: Color{
			R: 0,
			G: 255,
			B: 128,
		},
	})
}
