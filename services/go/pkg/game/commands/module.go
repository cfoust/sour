package game

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/cfoust/sour/pkg/game"
)

type Command struct {
	Name        string
	Aliases     []string
	ArgFormat   string
	Description string
	Callback    interface{}
}

func (cmd *Command) String() string {
	return fmt.Sprintf("%s %s", game.Green("#"+cmd.Name), cmd.ArgFormat)
}

func (cmd *Command) Detailed() string {
	aliases := ""
	if len(cmd.Aliases) > 0 {
		aliases = game.Gray(fmt.Sprintf("(alias %s)", strings.Join(cmd.Aliases, ", ")))
	}
	return fmt.Sprintf("%s: %s\n%s", cmd.String(), aliases, cmd.Description)
}

type CommandGroup[User any] struct {
	// e.g. cluster, space, server
	namespace string
	color     game.TextColor
	commands  map[string]*Command
	message   func(User, string)
}

var ERROR_INTERFACE = reflect.TypeOf((*error)(nil)).Elem()

func (c *CommandGroup[User]) validateCallback(callback interface{}) error {
	type_ := reflect.TypeOf(callback)

	if type_.Kind() != reflect.Func {
		return fmt.Errorf("callback must be a function")
	}

	if type_.NumOut() > 1 {
		return fmt.Errorf("callback can only have a single return value")
	}

	if type_.NumOut() == 1 {
		returnType := type_.Out(0)
		returnTypeValue := reflect.New(returnType)
		if _, ok := returnTypeValue.Interface().(*error); !ok {
			return fmt.Errorf("callback return type must be error")
		}
	}

	var u User
	userType := reflect.TypeOf(u)
	for i := 0; i < type_.NumIn(); i++ {
		argType := type_.In(i)

		switch argType.Kind() {
		case reflect.Int, reflect.String, reflect.Bool, reflect.Float64:
			continue
		default:
			if argType == userType {
				continue
			}

			return fmt.Errorf("invalid callback parameter type %s", argType.String())
		}
	}

	return nil
}

func (c *CommandGroup[User]) Register(command Command) {
	c.commands[command.Name] = &command

	for _, alias := range command.Aliases {
		c.commands[alias] = &command
	}
}

func NewCommandGroup[User any](namespace string, color game.TextColor, message func(User, string)) *CommandGroup[User] {
	return &CommandGroup[User]{
		namespace: namespace,
		color:     color,
		message:   message,
	}
}

func (c *CommandGroup[User]) Name() string {
	return c.color.Wrap(c.namespace)
}

func (c *CommandGroup[User]) Help() string {
	commands := make([]string, 0)

	for command := range c.commands {
		commands = append(commands, command)
	}

	sort.Strings(commands)
	return fmt.Sprintf("%s: %s", c.Name(), strings.Join(commands, ", "))
}

func (c *CommandGroup[User]) resolve(args []string) *Command {
	if len(args) == 0 {
		return nil
	}

	// First check if the namespace is included.
	target := args[0]
	if target == c.namespace {
		// You can't just address the namespace.
		if len(args) == 1 {
			return nil
		}

		target = args[1]
	}

	command, ok := c.commands[target]
	if !ok {
		return nil
	}

	return command
}

// Whether or not this command group can respond to this command.
func (c *CommandGroup[User]) CanHandle(args []string) bool {
	return c.resolve(args) != nil
}

func (c *CommandGroup[User]) Handle(user User, args []string) error {
	command := c.resolve(args)
	if command == nil {
		return fmt.Errorf("%s: unknown command", c.Name())
	}

	callback := command.Callback

	return nil
}

type Commandable interface {
	CanHandle([]string) bool
	Help() string
}

// cluster, space, server

// #creategame ffa complex
// #space edit
// #space help
// #server queuemap complex
