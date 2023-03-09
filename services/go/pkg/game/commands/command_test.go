package command

import (
	"strings"
	"testing"

	"github.com/cfoust/sour/pkg/game"

	"github.com/stretchr/testify/assert"
)

type User struct{}

var USER = &User{}
var SEND = func(user *User, message string) {
}

func TestCallbacks(t *testing.T) {
	g := NewCommandGroup("test", game.ColorGreen, SEND)

	// BAD
	err := g.Register(Command{
		Callback: func(val float32) {
		},
	})
	assert.NotNil(t, err)

	err = g.Register(Command{
		Callback: func(val byte) {
		},
	})
	assert.NotNil(t, err)

	err = g.Register(Command{
		Callback: func() bool {
			return false
		},
	})
	assert.NotNil(t, err)

	err = g.Register(Command{
		Callback: func() (int, int) {
			return 2, 2
		},
	})
	assert.NotNil(t, err)

	err = g.Register(Command{
		Callback: func(optional *int, required int) {
		},
	})
	assert.NotNil(t, err)

	err = g.Register(Command{
		Callback: func(slice []int) {
		},
	})
	assert.NotNil(t, err)

	// GOOD
	err = g.Register(Command{
		Callback: func() error {
			return nil
		},
	})
	assert.Nil(t, err)

	err = g.Register(Command{
		Callback: func(u *User) {
		},
	})
	assert.Nil(t, err)

	err = g.Register(Command{
		Callback: func(required bool, optional *bool) {
		},
	})
	assert.Nil(t, err)

	err = g.Register(Command{
		Callback: func(args []string) {
		},
	})
	assert.Nil(t, err)
}

func run(g *CommandGroup[*User], command string) error {
	args := strings.Split(command, " ")
	return g.Handle(USER, args)
}

func runCommand(t *testing.T, command string, callback interface{}) {
	g := NewCommandGroup("test", game.ColorGreen, SEND)
	err := g.Register(Command{
		Name:     "cmd",
		Callback: callback,
	})
	assert.Nil(t, err)

	err = run(g, command)
	assert.Nil(t, err)
}

func TestHandling(t *testing.T) {
	runCommand(t, "cmd", func(u *User) {
		assert.Equal(t, u, USER)
	})

	runCommand(t, "cmd str", func(str string) {
		assert.Equal(t, str, "str")
	})

	runCommand(t, "cmd 1337", func(num int) {
		assert.Equal(t, num, 1337)
	})

	runCommand(t, "cmd 21.2", func(float_ float64) {
		assert.Equal(t, float_, 21.2)
	})

	runCommand(t, "cmd true", func(value bool) {
		assert.Equal(t, value, true)
	})

	runCommand(t, "cmd on", func(value bool) {
		assert.Equal(t, value, true)
	})

	runCommand(t, "cmd false", func(value bool) {
		assert.Equal(t, value, false)
	})

	runCommand(t, "cmd 1 2 3", func(args []string) {
		assert.Equal(t, len(args), 3)
	})

	runCommand(t, "cmd", func(value *int) {
		// can't use assert.Equal because it uses reflection and fails
		if value != nil {
			t.Fail()
		}
	})

	runCommand(t, "cmd 2", func(value *int) {
		assert.Equal(t, *value, 2)
	})
}
