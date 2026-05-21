package cscript

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

const MAXRUNDEPTH = 255

// VM is a CubeScript virtual machine. Each instance has independent state
// and is safe for single-goroutine use. Create separate VMs for concurrent use.
type VM struct {
	idents   map[string]*Ident
	identMap []*Ident // indexed by Ident.Index
	result   Value
	depth    int
}

// NewVM creates a new CubeScript VM with all built-in commands registered.
func NewVM() *VM {
	vm := &VM{
		idents: make(map[string]*Ident),
	}
	// Register arg1..arg25
	for i := 0; i < MAXARGS; i++ {
		name := fmt.Sprintf("arg%d", i+1)
		vm.newIdent(name, IdfArg)
	}
	vm.registerBuiltins()
	return vm
}

func (vm *VM) newIdent(name string, flags int) *Ident {
	if existing, ok := vm.idents[name]; ok {
		return existing
	}
	id := &Ident{
		Type:  IdAlias,
		Name:  name,
		Index: len(vm.identMap),
		Flags: flags,
		Val:   NullVal(),
	}
	vm.idents[name] = id
	vm.identMap = append(vm.identMap, id)
	return id
}

func (vm *VM) getIdent(name string) *Ident {
	return vm.idents[name]
}

func (vm *VM) setAlias(name string, val Value) {
	id := vm.idents[name]
	if id != nil {
		if id.Type == IdAlias {
			id.Val = val
			id.Flags &= ^IdfUnknown
		}
		return
	}
	vm.newIdent(name, 0)
	vm.idents[name].Val = val
}

// AddCommand registers a Go callback as a CubeScript command.
// The callback can have parameters of type string, int, or float32,
// and can optionally return string, int, or float32.
func (vm *VM) AddCommand(name string, callback interface{}) error {
	t := reflect.TypeOf(callback)
	if t.Kind() != reflect.Func {
		return fmt.Errorf("callback must be a function")
	}
	if t.NumIn() > 12 {
		return fmt.Errorf("callback has too many args")
	}

	// Build format string
	var args strings.Builder
	for i := 0; i < t.NumIn(); i++ {
		switch t.In(i).Kind() {
		case reflect.Int:
			args.WriteByte('i')
		case reflect.Float32:
			args.WriteByte('f')
		case reflect.String:
			args.WriteByte('s')
		default:
			return fmt.Errorf("unsupported arg type: %s", t.In(i))
		}
	}

	id := vm.idents[name]
	if id == nil {
		id = &Ident{
			Type:  IdCommand,
			Name:  name,
			Index: len(vm.identMap),
			Args:  args.String(),
			Fun:   callback,
		}
		vm.idents[name] = id
		vm.identMap = append(vm.identMap, id)
	} else {
		id.Type = IdCommand
		id.Args = args.String()
		id.Fun = callback
	}

	return nil
}

// Run executes CubeScript source code.
func (vm *VM) Run(code string) {
	ast := Parse(code)
	vm.execBlock(ast)
}

// Execute executes CubeScript and returns the result as an int.
func (vm *VM) Execute(code string) int {
	ast := Parse(code)
	result := vm.execBlock(ast)
	return result.GetInt()
}

// ExecuteStr executes CubeScript and returns the result as a string.
func (vm *VM) ExecuteStr(code string) string {
	ast := Parse(code)
	result := vm.execBlock(ast)
	return result.GetStr()
}

func (vm *VM) execBlock(block *Block) Value {
	var result Value
	for _, stmt := range block.Stmts {
		result = vm.execNode(stmt)
	}
	return result
}

func (vm *VM) execNode(node Node) Value {
	if vm.depth >= MAXRUNDEPTH {
		return NullVal()
	}

	switch n := node.(type) {
	case *Block:
		return vm.execBlock(n)

	case *Command:
		return vm.execCommand(n)

	case *Assignment:
		val := vm.evalNode(n.Value)
		vm.setAlias(n.Name, val)
		return NullVal()

	case *Word:
		return StrVal(n.Val)

	case *StringLit:
		return StrVal(n.Val)

	case *CodeBlock:
		return CodeVal(n.Body)

	case *SubExpr:
		return vm.execBlock(n.Body)

	case *Lookup:
		name := vm.evalNode(n.Name).GetStr()
		id := vm.getIdent(name)
		if id == nil {
			return StrVal("")
		}
		switch id.Type {
		case IdAlias:
			return id.Val
		case IdVar:
			if id.IStorage != nil {
				return IntVal(*id.IStorage)
			}
			return id.Val
		case IdFvar:
			if id.FStorage != nil {
				return FloatVal(*id.FStorage)
			}
			return id.Val
		case IdSvar:
			if id.SStorage != nil {
				return StrVal(*id.SStorage)
			}
			return id.Val
		default:
			return id.Val
		}

	case *Interpolation:
		var buf strings.Builder
		for _, part := range n.Parts {
			v := vm.evalNode(part)
			buf.WriteString(v.GetStr())
		}
		return StrVal(buf.String())

	default:
		return NullVal()
	}
}

func (vm *VM) evalNode(node Node) Value {
	return vm.execNode(node)
}

func (vm *VM) execCommand(cmd *Command) Value {
	// Evaluate the command name
	nameVal := vm.evalNode(cmd.Name)
	name := nameVal.GetStr()

	id := vm.getIdent(name)

	if id == nil {
		// Unknown command - might be a number literal
		if _, err := strconv.ParseFloat(name, 64); err == nil {
			return nameVal
		}
		// silently ignore unknown commands
		return NullVal()
	}

	switch id.Type {
	case IdCommand:
		// Check if it's a built-in first
		if _, ok := id.Fun.(builtinMarker); ok {
			if result, handled := vm.callBuiltin(name, cmd.Args); handled {
				return result
			}
		}
		return vm.callCommand(id, cmd.Args)

	case IdAlias:
		if id.Val.IsNull() || (id.Flags&IdfUnknown) != 0 {
			return NullVal()
		}
		// Execute the alias body with args
		return vm.callAlias(id, cmd.Args)

	case IdVar:
		if len(cmd.Args) == 0 {
			if id.IStorage != nil {
				return IntVal(*id.IStorage)
			}
			return id.Val
		}
		val := vm.evalNode(cmd.Args[0]).GetInt()
		if id.IStorage != nil {
			*id.IStorage = clampInt(val, id.MinVal, id.MaxVal)
		} else {
			id.Val = IntVal(val)
		}
		return NullVal()

	case IdFvar:
		if len(cmd.Args) == 0 {
			if id.FStorage != nil {
				return FloatVal(*id.FStorage)
			}
			return id.Val
		}
		val := vm.evalNode(cmd.Args[0]).GetFloat()
		if id.FStorage != nil {
			*id.FStorage = float32(math.Max(float64(id.MinFVal), math.Min(float64(val), float64(id.MaxFVal))))
		} else {
			id.Val = FloatVal(val)
		}
		return NullVal()

	case IdSvar:
		if len(cmd.Args) == 0 {
			if id.SStorage != nil {
				return StrVal(*id.SStorage)
			}
			return id.Val
		}
		val := vm.evalNode(cmd.Args[0]).GetStr()
		if id.SStorage != nil {
			*id.SStorage = val
		} else {
			id.Val = StrVal(val)
		}
		return NullVal()

	default:
		return NullVal()
	}
}

func (vm *VM) callAlias(id *Ident, argNodes []Node) Value {
	vm.depth++
	defer func() { vm.depth-- }()

	// Evaluate arguments
	args := make([]Value, len(argNodes))
	for i, a := range argNodes {
		args[i] = vm.evalNode(a)
	}

	// Push args as arg1, arg2, etc.
	for i, a := range args {
		argName := fmt.Sprintf("arg%d", i+1)
		argId := vm.getIdent(argName)
		if argId != nil {
			argId.Push(a)
		}
	}

	// Also set numargs
	numArgsId := vm.getIdent("numargs")
	if numArgsId != nil {
		numArgsId.Push(IntVal(len(args)))
	}

	// Execute alias body
	body := id.Val.GetStr()
	result := vm.execBlock(Parse(body))

	// Pop args
	if numArgsId != nil {
		numArgsId.Pop()
	}
	for i := range args {
		argName := fmt.Sprintf("arg%d", i+1)
		argId := vm.getIdent(argName)
		if argId != nil {
			argId.Pop()
		}
	}

	return result
}

func (vm *VM) callCommand(id *Ident, argNodes []Node) Value {
	if id.Fun == nil {
		return NullVal()
	}

	t := reflect.TypeOf(id.Fun)
	numIn := t.NumIn()

	// Evaluate args
	evalArgs := make([]Value, len(argNodes))
	for i, a := range argNodes {
		evalArgs[i] = vm.evalNode(a)
	}

	// Build reflect call args
	callArgs := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argType := t.In(i)
		var val Value
		if i < len(evalArgs) {
			val = evalArgs[i]
		} else {
			val = NullVal()
		}

		switch argType.Kind() {
		case reflect.Int:
			callArgs[i] = reflect.ValueOf(val.GetInt())
		case reflect.Float32:
			callArgs[i] = reflect.ValueOf(val.GetFloat())
		case reflect.String:
			callArgs[i] = reflect.ValueOf(val.GetStr())
		default:
			callArgs[i] = reflect.Zero(argType)
		}
	}

	// Call the function
	results := reflect.ValueOf(id.Fun).Call(callArgs)

	// Process return value
	if len(results) > 0 {
		ret := results[0]
		switch ret.Kind() {
		case reflect.Int:
			vm.result = IntVal(int(ret.Int()))
			return vm.result
		case reflect.String:
			vm.result = StrVal(ret.String())
			return vm.result
		case reflect.Float32:
			vm.result = FloatVal(float32(ret.Float()))
			return vm.result
		}
	}

	return vm.result
}

// Intret sets the command return value to an integer.
func (vm *VM) Intret(v int) {
	vm.result = IntVal(v)
}

// Floatret sets the command return value to a float.
func (vm *VM) Floatret(v float32) {
	vm.result = FloatVal(v)
}

// Result sets the command return value to a string.
func (vm *VM) Result(s string) {
	vm.result = StrVal(s)
}

// GetAlias returns the string value of an alias, or "" if not found.
func (vm *VM) GetAlias(name string) string {
	id := vm.getIdent(name)
	if id == nil || id.Type != IdAlias {
		return ""
	}
	return id.Val.GetStr()
}
