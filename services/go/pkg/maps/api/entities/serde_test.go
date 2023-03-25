package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func cmp(t *testing.T, before EntityInfo) {
	a, err := Encode(before)
	assert.NoError(t, err)

	after, err := Decode(before.Type(), a)
	assert.NoError(t, err)

	t.Logf("%+v", after)
	assert.Equal(t, before, after, "should yield same result")
}

func TestMapModel(t *testing.T) {
	cmp(t, &MapModel{
		Angle: 1,
		Index: 2,
	})
}

func TestLight(t *testing.T) {
	cmp(t, &Light{
		Radius: 8,
		Color: Color{
			R: 0,
			G: 255,
			B: 128,
		},
	})
}

func TestParticle(t *testing.T) {
	cmp(t, &Particles{
		Particle: ParticleTypeFire,
		Data: &Fire{
			Height: 8,
			Radius: 8,
		},
	})
}
