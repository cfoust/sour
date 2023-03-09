package command

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/cfoust/sour/pkg/game"
	"github.com/rs/zerolog/log"
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

func (c *CommandGroup[User]) Register(command Command) error {
	err := c.validateCallback(command.Callback)
	if err != nil {
		return err
	}

	c.commands[command.Name] = &command

	for _, alias := range command.Aliases {
		c.commands[alias] = &command
	}

	return nil
}

func NewCommandGroup[User any](namespace string, color game.TextColor, message func(User, string)) *CommandGroup[User] {
	return &CommandGroup[User]{
		namespace: namespace,
		color:     color,
		message:   message,
		commands:  make(map[string]*Command),
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

func (c *CommandGroup[User]) resolve(args []string) (*Command, []string) {
	if len(args) == 0 {
		return nil, nil
	}

	// First check if the namespace is included.
	target := args[0]
	commandArguments := args[1:]
	if target == c.namespace {
		// You can't just address the namespace.
		if len(args) == 1 {
			return nil, nil
		}

		target = args[1]
		commandArguments = args[1:]
	}

	command, ok := c.commands[target]
	if !ok {
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
			return NIL, fmt.Errorf("expected number argument")
		}

		if isPointer {
			return reflect.ValueOf(&value), nil
		}

		return reflect.ValueOf(value), nil
	case reflect.Float64:
		value, err := strconv.ParseFloat(argument, 64)
		if err != nil {
			return NIL, fmt.Errorf("expected decimal argument")
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
			return NIL, fmt.Errorf("expected boolean argument")
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

func (c *CommandGroup[User]) Handle(user User, args []string) error {
	command, commandArgs := c.resolve(args)
	if command == nil {
		return fmt.Errorf("%s: unknown command", c.Name())
	}

	callback := command.Callback
	callbackType := reflect.TypeOf(callback)
	callbackArgs := make([]reflect.Value, 0)
	originalArgs := commandArgs

	for i := 0; i < callbackType.NumIn(); i++ {
		argType := callbackType.In(i)

		var value reflect.Value
		switch argType.Kind() {
		case reflect.Slice:
			value = reflect.ValueOf(originalArgs)
		case reflect.Pointer:
			if argType == reflect.TypeOf(user) {
				value = reflect.ValueOf(user)
				break
			}

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
				return err
			}

			value = parsedValue
		case reflect.Int, reflect.String, reflect.Bool, reflect.Float64:
			argument := commandArgs[0]
			commandArgs = commandArgs[1:]
			parsedValue, err := parseArg(argType, argument, false)
			if err != nil {
				return err
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

	log.Info().Msgf("%+v", callbackArgs)

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
	CanHandle([]string) bool
	Help() string
}

// cluster, space, server

// #creategame ffa complex
// #space edit
// #space help
// #server queuemap complex
