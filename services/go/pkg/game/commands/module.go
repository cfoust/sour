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

func (c *CommandGroup[User]) Register(command Command) {
	err := c.validateCallback(command.Callback)
	if err != nil {
		panic(err.Error())
	}

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
		commandArguments = args[2:]
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

func parseArg(type_ reflect.Type, argument string) (reflect.Value, error) {
	switch type_.Kind() {
	case reflect.Int:
		value, err := strconv.Atoi(argument)
		if err != nil {
			return NIL, fmt.Errorf("expected number argument")
		}

		return reflect.ValueOf(value), nil
	case reflect.Float64:
		value, err := strconv.ParseFloat(argument, 64)
		if err != nil {
			return NIL, fmt.Errorf("expected decimal argument")
		}

		return reflect.ValueOf(value), nil
	case reflect.Bool:
		value := false

		if argument == "yes" || argument == "1" || argument == "on" {
			value = true
		} else if argument == "no" || argument == "0" || argument == "off" {
			value = false
		} else {
			return NIL, fmt.Errorf("expected boolean argument")
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
	originalArgs := callbackArgs

	for i := 0; i < callbackType.NumIn(); i++ {
		argType := callbackType.In(i)

		var value reflect.Value
		switch argType.Kind() {
		case reflect.Slice:
			value = reflect.ValueOf(originalArgs)
		case reflect.Pointer:
			if len(commandArgs) == 0 {
				value = reflect.ValueOf(nil)
				break
			}
			argument := commandArgs[0]
			commandArgs = commandArgs[1:]
			parsedValue, err := parseArg(argType.Elem(), argument)
			if err != nil {
				return err
			}

			value = parsedValue.Addr()
		case reflect.Int, reflect.String, reflect.Bool, reflect.Float64:
			argument := commandArgs[0]
			commandArgs = commandArgs[1:]
			parsedValue, err := parseArg(argType, argument)
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
