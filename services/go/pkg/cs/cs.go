package cs

// #cgo CXXFLAGS: -std=c++03
import "C"

import (
	"fmt"
	"github.com/sasha-s/go-deadlock"
	"reflect"
	"strconv"
)

var M deadlock.Mutex

// Keep track of the command hooks we've created in the VM.
var hooks map[string]struct{} = make(map[string]struct{})

func registerHook(name string) {
	M.Lock()
	defer M.Unlock()

	if _, ok := hooks[name]; ok {
		return
	}

	hooks[name] = struct{}{}
	Execute(fmt.Sprintf(`
%s = [
  _gocall %s $arg1 $arg2 $arg3 $arg4 $arg5 $arg6 $arg7 $arg8 $arg9 $arg10 $arg11 $arg12
]
`, name, name))
}

type VM struct {
	commands map[string]interface{}
	m        deadlock.RWMutex
}

func NewVM() *VM {
	return &VM{
		commands: make(map[string]interface{}),
	}
}

func isValidType(type_ reflect.Type) bool {
	switch type_.Kind() {
	case reflect.Int:
		fallthrough
	case reflect.Float32:
		fallthrough
	case reflect.String:
		return true
	default:
		return false
	}
}

func (c *VM) AddCommand(name string, callback interface{}) error {
	// Validate the callback function
	type_ := reflect.TypeOf(callback)
	if type_.Kind() != reflect.Func {
		return fmt.Errorf("callback must be a function")
	}

	if type_.NumIn() > 12 {
		return fmt.Errorf("callback has too many args")
	}

	for i := 0; i < type_.NumIn(); i++ {
		argType := type_.In(i)

		if !isValidType(argType) {
			return fmt.Errorf(
				"arg %d's type %s not supported",
				i,
				argType.String(),
			)
		}
	}

	if type_.NumOut() > 1 {
		return fmt.Errorf("callback has too many results")
	}

	if type_.NumOut() == 1 && !isValidType(type_.Out(0)) {
		return fmt.Errorf("callback has invalid result type")
	}

	c.m.Lock()
	c.commands[name] = callback
	c.m.Unlock()

	registerHook(name)

	return nil
}

var vm *VM = nil

//export GoCall
func GoCall(
	name *C.char,
	_1 *C.char,
	_2 *C.char,
	_3 *C.char,
	_4 *C.char,
	_5 *C.char,
	_6 *C.char,
	_7 *C.char,
	_8 *C.char,
	_9 *C.char,
	_10 *C.char,
	_11 *C.char,
	_12 *C.char,
) {
	vm.m.Lock()
	defer vm.m.Unlock()
	callback, ok := vm.commands[C.GoString(name)]
	if !ok {
		return
	}

	args := []string{
		C.GoString(_1),
		C.GoString(_2),
		C.GoString(_3),
		C.GoString(_4),
		C.GoString(_5),
		C.GoString(_6),
		C.GoString(_7),
		C.GoString(_8),
		C.GoString(_9),
		C.GoString(_10),
		C.GoString(_11),
		C.GoString(_12),
	}
	callArgs := make([]reflect.Value, 0)
	type_ := reflect.TypeOf(callback)
	for i := 0; i < type_.NumIn(); i++ {
		argType := type_.In(i)

		switch argType.Kind() {
		case reflect.Int:
			value, err := strconv.Atoi(args[i])
			if err != nil {
				value = 0
			}
			callArgs = append(callArgs, reflect.ValueOf(value))
		case reflect.String:
			callArgs = append(callArgs, reflect.ValueOf(args[i]))
		case reflect.Float32:
			value, err := strconv.ParseFloat(args[i], 32)
			if err != nil {
				value = 0
			}
			callArgs = append(callArgs, reflect.ValueOf(value))
		}
	}

	callValue := reflect.ValueOf(callback)
	results := callValue.Call(callArgs)
	
	if type_.NumOut() > 0 {
		resultVal := results[0]
		switch type_.Out(0).Kind() {
		case reflect.Int:
			value := resultVal.Int()
			Intret(int(value))
		case reflect.String:
			value := resultVal.String()
			Stringret(value)
		case reflect.Float32:
			value := resultVal.Float()
			Floatret(float32(value))
		}
	}
}

func (c *VM) Run(code string) error {
	M.Lock()
	vm = c
	Execute(code)
	defer M.Unlock()
	return nil
}
