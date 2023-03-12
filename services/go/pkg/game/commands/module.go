package commands

import (
	"context"
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

func (cmd *Command) Help() string {
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
	// Just commands (for help)
	commands map[string]*Command
	// Includes aliases
	references map[string]*Command
	message    func(User, string)
}

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

	haveOptional := false

	var u User
	userType := reflect.TypeOf(u)
	for i := 0; i < type_.NumIn(); i++ {
		argType := type_.In(i)

		switch argType.Kind() {
		case reflect.Slice:
			if argType.Elem().Kind() != reflect.String {
				return fmt.Errorf("slice parameter %s can only be string", argType.String())
			}
		case reflect.Int, reflect.String, reflect.Bool, reflect.Float64:
			if haveOptional {
				return fmt.Errorf("required parameter cannot follow optional")
			}
			continue
		case reflect.Interface:
			if argType == userType {
				continue
			}

			argValue := reflect.New(argType)
			if _, ok := argValue.Interface().(*context.Context); !ok {
				return fmt.Errorf("context and user type are only allowable interfaces")
			}
		case reflect.Pointer:
			if argType == userType {
				continue
			}

			haveOptional = true

			elemType := argType.Elem()
			switch elemType.Kind() {
			// String omitted intentionally
			case reflect.Int, reflect.Bool, reflect.Float64:
				continue
			default:
				return fmt.Errorf("invalid optional callback parameter type %s", elemType.String())
			}
		default:
			if argType == userType {
				continue
			}

			return fmt.Errorf("invalid callback parameter type %s", argType.String())
		}
	}

	return nil
}

func (c *CommandGroup[User]) Register(commands ...Command) error {
	for _, command := range commands {
		err := c.validateCallback(command.Callback)
		if err != nil {
			return err
		}

		copied := command
		c.commands[command.Name] = &copied
		c.references[command.Name] = &copied

		for _, alias := range command.Aliases {
			c.references[alias] = &copied
		}

	}

	return nil
}

func NewCommandGroup[User any](namespace string, color game.TextColor) *CommandGroup[User] {
	return &CommandGroup[User]{
		namespace: namespace,
		color:     color,
		commands:  make(map[string]*Command),
		references:  make(map[string]*Command),
	}
}

func (c *CommandGroup[User]) Name() string {
	return c.color.Wrap(c.namespace)
}

func (c *CommandGroup[User]) Prefix(message string) string {
	return fmt.Sprintf("%s %s", c.Name(), message)
}

func (c *CommandGroup[User]) Help() string {
	commands := make([]string, 0)

	for command := range c.commands {
		commands = append(commands, command)
	}

	sort.Strings(commands)
	return c.Prefix(strings.Join(commands, ", "))
}

func (c *CommandGroup[User]) resolve(args []string) (*Command, []string) {
	if len(args) == 0 {
		return nil, nil
	}

	// First check if the namespace is included.
	target := args[0]
	commandArguments := args[1:]
	if target == c.namespace || strings.HasPrefix(c.namespace, target) {
		// You can't just address the namespace.
		if len(args) == 1 {
			return nil, nil
		}

		target = args[1]
		commandArguments = args[2:]
	}

	command, ok := c.references[target]
	if !ok {
		// A command invocation can also be any prefix of a valid
		// command, do one last check
		for name, command := range c.references {
			if strings.HasPrefix(name, target) {
				return command, commandArguments
			}
		}

		return nil, nil
	}

	return command, commandArguments
}

// Whether or not this command group can respond to this command.
func (c *CommandGroup[User]) CanHandle(args []string) bool {
	command, _ := c.resolve(args)
	return command != nil
}

var NIL = reflect.ValueOf(nil)

var TRUTHY = []string{
	"true",
	"yes",
	"1",
	"on",
}

var FALSY = []string{
	"false",
	"no",
	"0",
	"off",
}

func parseArg(type_ reflect.Type, argument string, isPointer bool) (reflect.Value, error) {
	switch type_.Kind() {
	case reflect.Int:
		value, err := strconv.Atoi(argument)
		if err != nil {
			return NIL, fmt.Errorf("expected number")
		}

		if isPointer {
			return reflect.ValueOf(&value), nil
		}

		return reflect.ValueOf(value), nil
	case reflect.Float64:
		value, err := strconv.ParseFloat(argument, 64)
		if err != nil {
			return NIL, fmt.Errorf("expected decimal")
		}

		if isPointer {
			return reflect.ValueOf(&value), nil
		}

		return reflect.ValueOf(value), nil
	case reflect.Bool:
		value := false
		matched := false

		for _, truthy := range TRUTHY {
			if argument == truthy {
				value = true
				matched = true
				break
			}
		}

		for _, falsy := range FALSY {
			if argument == falsy {
				value = false
				matched = true
				break
			}
		}

		if !matched {
			return NIL, fmt.Errorf("expected [true|yes|1|false|no|0]")
		}

		if isPointer {
			return reflect.ValueOf(&value), nil
		}

		return reflect.ValueOf(value), nil
	case reflect.String:
		return reflect.ValueOf(argument), nil
	}

	return NIL, fmt.Errorf("could not parse argument")
}

func (c *CommandGroup[User]) GetHelp(args []string) string {
	resolved, _ := c.resolve(args)
	return c.Prefix(resolved.Help())
}

func (c *CommandGroup[User]) Handle(ctx context.Context, user User, args []string) error {
	command, commandArgs := c.resolve(args)
	if command == nil {
		return fmt.Errorf(c.Prefix("unknown command"))
	}

	callback := command.Callback
	callbackType := reflect.TypeOf(callback)
	callbackArgs := make([]reflect.Value, 0)
	originalArgs := commandArgs

	numPrimitive := 0

	for i := 0; i < callbackType.NumIn(); i++ {
		argType := callbackType.In(i)

		var value reflect.Value
		switch argType.Kind() {
		case reflect.Slice:
			value = reflect.ValueOf(originalArgs)
		case reflect.Interface:
			if argType == reflect.TypeOf(user) {
				value = reflect.ValueOf(user)
				break
			}

			argValue := reflect.New(argType)
			if _, ok := argValue.Interface().(*context.Context); ok {
				value = reflect.ValueOf(ctx)
			}

		case reflect.Pointer:
			if argType == reflect.TypeOf(user) {
				value = reflect.ValueOf(user)
				break
			}

			numPrimitive += 1

			if len(commandArgs) == 0 {
				switch argType.Elem().Kind() {
				case reflect.Int:
					value = reflect.ValueOf((*int)(nil))
				case reflect.Bool:
					value = reflect.ValueOf((*bool)(nil))
				case reflect.Float64:
					value = reflect.ValueOf((*float64)(nil))
				}
				break
			}
			argument := commandArgs[0]
			commandArgs = commandArgs[1:]
			parsedValue, err := parseArg(argType.Elem(), argument, true)
			if err != nil {
				return fmt.Errorf("invalid argument %d: %s", numPrimitive, err.Error())
			}

			value = parsedValue
		case reflect.Int, reflect.String, reflect.Bool, reflect.Float64:
			numPrimitive += 1

			if len(commandArgs) == 0 {
				return fmt.Errorf("missing argument %d", numPrimitive)
			}

			argument := commandArgs[0]
			commandArgs = commandArgs[1:]
			parsedValue, err := parseArg(argType, argument, false)
			if err != nil {
				return fmt.Errorf("invalid argument %d: %s", numPrimitive, err.Error())
			}

			value = parsedValue
		default:
			if argType == reflect.TypeOf(user) {
				value = reflect.ValueOf(user)
				break
			}
			return fmt.Errorf("operational fault while handling command")
		}

		callbackArgs = append(callbackArgs, value)
	}

	results := reflect.ValueOf(callback).Call(callbackArgs)
	if len(results) > 0 {
		result := results[0]
		if err, ok := result.Interface().(error); ok {
			return err
		}
	}

	return nil
}

type Commandable interface {
	// Get help for a specific command (or empty string if it does not exist.)
	GetHelp([]string) string
	// Lists all commands.
	Help() string
}

var _ Commandable = (*CommandGroup[int])(nil)
