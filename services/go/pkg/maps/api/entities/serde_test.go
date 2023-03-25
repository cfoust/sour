package entities

import (
	"testing"

	C "github.com/cfoust/sour/pkg/game/constants"

	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	a := Attributes{}
	a.Put(1)
	a.Put(2)

	info, err := Decode(C.EntityTypeMapModel, &a)
	assert.NoError(t, err)
	assert.Equal(t, info, &MapModel{
		Angle: 1,
		Index: 2,
	})
}
