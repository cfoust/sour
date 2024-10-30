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
			Color: Color16{
				R: 0x90,
				G: 0x30,
				B: 0x20,
			},
		},
	})
}

func TestDefault(t *testing.T) {
	before := Particles{
		Particle: ParticleTypeFire,
		Data: &Fire{
			Color: Color16{
				R: 0x90,
				G: 0x30,
				B: 0x20,
			},
		},
	}
	a, err := Encode(&before)
	assert.NoError(t, err)

	after, err := Decode(before.Type(), a)
	particles := after.(*Particles)
	assert.NoError(t, err)
	assert.Equal(t, particles.Data, &Fire{
		Radius: 1.5,
		Height: 0.5,
		Color: Color16{
			R: 0x90,
			G: 0x30,
			B: 0x20,
		},
	}, "should yield same result")
}
