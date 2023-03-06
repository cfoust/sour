package io

import (
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
)

func cmp[T any](t *testing.T, before T) {
	p := Packet{}
	err := p.Put(before)
	assert.NoError(t, err)

	var after T
	err = p.Get(&after)
	assert.NoError(t, err)

	t.Logf("%+v", after)
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

func TestEmbeddedStruct(t *testing.T) {
	type Other struct {
		A int
	}
	type Value struct {
		Other
		B int
	}

	val := Value{
		B: 3,
	}
	val.A = 2

	cmp(t, val)
}

type CustomMarshal struct {
	Num string
}

func (s *CustomMarshal) Unmarshal(p *Packet) error {
	value, ok := p.GetInt()
	if !ok {
	    return fmt.Errorf("failure")
	}

	if value == 1 {
		s.Num = "ok"
	} else {
		s.Num = "not ok"
	}

	return nil
}

func (s *CustomMarshal) Marshal(p *Packet) error {
	if s.Num == "ok" {
		return p.Put(1)
	} else {
		return p.Put(0)
	}
}

func TestMarshalable(t *testing.T) {
	type Value struct {
		A CustomMarshal
	}

	cmp(t, Value{
		A: CustomMarshal{
			Num: "ok",
		},
	})
}
