package io

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func cmp[T any](t *testing.T, before T) {
	p := Packet{}
	err := p.Put(before)
	assert.NoError(t, err)

	var after T
	err = p.Get(&after)
	assert.NoError(t, err)

	assert.Equal(t, before, after, "should yield same result")
}

func TestStaticArray(t *testing.T) {
	type Value struct {
		Array [2]int
	}

	cmp(t, Value{
		Array: [2]int{1, 2},
	})
}

func TestDynamicArray(t *testing.T) {
	type Value struct {
		Array []int
	}

	cmp(t, Value{
		Array: []int{1, 2},
	})
}

func TestIntTerm(t *testing.T) {
	type Item struct {
		Value int
	}
	type Value struct {
		Array []Item `type:"term"`
	}

	cmp(t, Value{
		Array: []Item{
			{0},
		},
	})
}

func TestStringTerm(t *testing.T) {
	type Item struct {
		Value string
	}
	type Value struct {
		Array []Item `type:"term"`
	}

	cmp(t, Value{
		Array: []Item{
			{"test"},
		},
	})
}
