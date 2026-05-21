package cscript

import (
	"fmt"
	"math"
	"strings"
)

func (vm *VM) registerBuiltins() {
	// We register builtins as special-cased names in execCommand.
	// The built-in commands need access to the VM and raw AST nodes
	// (for lazy evaluation of code blocks), so they can't be plain
	// Go callbacks. Instead, we register them as IdCommand with
	// special marker functions, and handle them in callBuiltin.

	builtinNames := []string{
		"if", "loop", "loopwhile", "while", "do", "result", "?",
		"concat", "concatword", "format", "at", "substr", "sublist",
		"listlen", "listfind", "looplist", "loopconcat", "loopconcatword",
		"looplistconcat", "looplistconcatword",
		"listfilter", "listdel", "prettylist", "indexof", "listsplice",
		"sortlist",
		"alias", "getalias", "set", "push", "local",
		"echo", "error", "cond", "case", "casef", "cases",
		"+", "-", "*", "+f", "-f", "*f",
		"=", "!=", "<", ">", "<=", ">=",
		"=f", "!=f", "<f", ">f", "<=f", ">=f",
		"=s", "!=s", "<s", ">s", "<=s", ">=s",
		"strcmp",
		"!", "&&", "||",
		"^", "&", "|", "~", "^~", "&~", "|~", "<<", ">>",
		"div", "mod", "divf", "modf",
		"sin", "cos", "tan", "asin", "acos", "atan", "atan2",
		"sqrt", "pow", "loge", "log2", "log10", "exp",
		"min", "max", "minf", "maxf", "abs", "absf",
		"floor", "ceil", "round",
		"strlen", "strstr", "strreplace", "strsplice",
		"strcode", "codestr", "strlower", "strupper",
		"escape", "unescape",
		"pushif", "append", "appendword",
		"loopsublist",
	}

	for _, name := range builtinNames {
		id := &Ident{
			Type:  IdCommand,
			Name:  name,
			Index: len(vm.identMap),
			Fun:   builtinMarker{},
		}
		vm.idents[name] = id
		vm.identMap = append(vm.identMap, id)
	}

	// Also register "numargs" as a variable
	vm.newIdent("numargs", 0)
}

type builtinMarker struct{}

// Override callCommand to handle builtins
func (vm *VM) callBuiltin(name string, argNodes []Node) (Value, bool) {
	switch name {
	// Control flow
	case "if":
		return vm.builtinIf(argNodes), true
	case "?":
		return vm.builtinTernary(argNodes), true
	case "loop":
		return vm.builtinLoop(argNodes), true
	case "loopwhile":
		return vm.builtinLoopWhile(argNodes), true
	case "while":
		return vm.builtinWhile(argNodes), true
	case "do":
		return vm.builtinDo(argNodes), true
	case "result":
		return vm.builtinResult(argNodes), true
	case "cond":
		return vm.builtinCond(argNodes), true
	case "case":
		return vm.builtinCase(argNodes, false), true
	case "casef":
		return vm.builtinCaseF(argNodes), true
	case "cases":
		return vm.builtinCaseS(argNodes), true

	// Aliases
	case "alias":
		return vm.builtinAlias(argNodes), true
	case "getalias":
		return vm.builtinGetAlias(argNodes), true
	case "set":
		return vm.builtinSet(argNodes), true
	case "local":
		return vm.builtinLocal(argNodes), true
	case "push":
		return vm.builtinPush(argNodes), true
	case "pushif":
		return vm.builtinPushIf(argNodes), true
	case "append":
		return vm.builtinAppend(argNodes, true), true
	case "appendword":
		return vm.builtinAppend(argNodes, false), true

	// String operations
	case "concat":
		return vm.builtinConcat(argNodes, true), true
	case "concatword":
		return vm.builtinConcat(argNodes, false), true
	case "format":
		return vm.builtinFormat(argNodes), true
	case "echo":
		return vm.builtinEcho(argNodes), true
	case "error":
		return vm.builtinEcho(argNodes), true
	case "escape":
		return vm.builtinEscape(argNodes), true
	case "unescape":
		return vm.builtinUnescape(argNodes), true
	case "strlower":
		return vm.builtinStrLower(argNodes), true
	case "strupper":
		return vm.builtinStrUpper(argNodes), true

	// Integer arithmetic
	case "+":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a + b }), true
	case "-":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a - b }), true
	case "*":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a * b }), true
	case "div":
		return vm.builtinIntOp2(argNodes, func(a, b int) int {
			if b == 0 { return 0 }; return a / b
		}), true
	case "mod":
		return vm.builtinIntOp2(argNodes, func(a, b int) int {
			if b == 0 { return 0 }; return a % b
		}), true

	// Float arithmetic
	case "+f":
		return vm.builtinFloatOp2(argNodes, func(a, b float32) float32 { return a + b }), true
	case "-f":
		return vm.builtinFloatOp2(argNodes, func(a, b float32) float32 { return a - b }), true
	case "*f":
		return vm.builtinFloatOp2(argNodes, func(a, b float32) float32 { return a * b }), true
	case "divf":
		return vm.builtinFloatOp2(argNodes, func(a, b float32) float32 {
			if b == 0 { return 0 }; return a / b
		}), true
	case "modf":
		return vm.builtinFloatOp2(argNodes, func(a, b float32) float32 {
			if b == 0 { return 0 }; return float32(math.Mod(float64(a), float64(b)))
		}), true

	// Integer comparison
	case "=":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a == b }), true
	case "!=":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a != b }), true
	case "<":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a < b }), true
	case ">":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a > b }), true
	case "<=":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a <= b }), true
	case ">=":
		return vm.builtinIntCmp(argNodes, func(a, b int) bool { return a >= b }), true

	// Float comparison
	case "=f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a == b }), true
	case "!=f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a != b }), true
	case "<f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a < b }), true
	case ">f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a > b }), true
	case "<=f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a <= b }), true
	case ">=f":
		return vm.builtinFloatCmp(argNodes, func(a, b float32) bool { return a >= b }), true

	// String comparison
	case "strcmp", "=s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a == b }), true
	case "!=s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a != b }), true
	case "<s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a < b }), true
	case ">s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a > b }), true
	case "<=s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a <= b }), true
	case ">=s":
		return vm.builtinStrCmp(argNodes, func(a, b string) bool { return a >= b }), true

	// Logical
	case "!":
		if len(argNodes) < 1 { return IntVal(1), true }
		v := vm.evalNode(argNodes[0])
		if v.GetBool() { return IntVal(0), true }
		return IntVal(1), true
	case "&&":
		return vm.builtinAnd(argNodes), true
	case "||":
		return vm.builtinOr(argNodes), true

	// Bitwise
	case "^":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a ^ b }), true
	case "&":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a & b }), true
	case "|":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a | b }), true
	case "~":
		if len(argNodes) < 1 { return IntVal(0), true }
		return IntVal(^vm.evalNode(argNodes[0]).GetInt()), true
	case "^~":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a ^ ^b }), true
	case "&~":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a & ^b }), true
	case "|~":
		return vm.builtinIntOp2(argNodes, func(a, b int) int { return a | ^b }), true
	case "<<":
		return vm.builtinIntOp2(argNodes, func(a, b int) int {
			if b >= 32 || b < 0 { return 0 }; return a << uint(b)
		}), true
	case ">>":
		return vm.builtinIntOp2(argNodes, func(a, b int) int {
			return a >> uint(clampInt(b, 0, 31))
		}), true

	// Math functions
	case "sin":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Sin(a * RAD) }), true
	case "cos":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Cos(a * RAD) }), true
	case "tan":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Tan(a * RAD) }), true
	case "asin":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Asin(a) / RAD }), true
	case "acos":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Acos(a) / RAD }), true
	case "atan":
		return vm.builtinMath1(argNodes, func(a float64) float64 { return math.Atan(a) / RAD }), true
	case "atan2":
		if len(argNodes) < 2 { return FloatVal(0), true }
		y := float64(vm.evalNode(argNodes[0]).GetFloat())
		x := float64(vm.evalNode(argNodes[1]).GetFloat())
		return FloatVal(float32(math.Atan2(y, x) / RAD)), true
	case "sqrt":
		return vm.builtinMath1(argNodes, math.Sqrt), true
	case "pow":
		if len(argNodes) < 2 { return FloatVal(0), true }
		a := float64(vm.evalNode(argNodes[0]).GetFloat())
		b := float64(vm.evalNode(argNodes[1]).GetFloat())
		return FloatVal(float32(math.Pow(a, b))), true
	case "loge":
		return vm.builtinMath1(argNodes, math.Log), true
	case "log2":
		return vm.builtinMath1(argNodes, math.Log2), true
	case "log10":
		return vm.builtinMath1(argNodes, math.Log10), true
	case "exp":
		return vm.builtinMath1(argNodes, math.Exp), true
	case "abs":
		if len(argNodes) < 1 { return IntVal(0), true }
		return IntVal(absInt(vm.evalNode(argNodes[0]).GetInt())), true
	case "absf":
		if len(argNodes) < 1 { return FloatVal(0), true }
		return FloatVal(float32(math.Abs(float64(vm.evalNode(argNodes[0]).GetFloat())))), true
	case "floor":
		if len(argNodes) < 1 { return FloatVal(0), true }
		return FloatVal(float32(math.Floor(float64(vm.evalNode(argNodes[0]).GetFloat())))), true
	case "ceil":
		if len(argNodes) < 1 { return FloatVal(0), true }
		return FloatVal(float32(math.Ceil(float64(vm.evalNode(argNodes[0]).GetFloat())))), true
	case "round":
		return vm.builtinRound(argNodes), true
	case "min":
		return vm.builtinMinMax(argNodes, true, false), true
	case "max":
		return vm.builtinMinMax(argNodes, false, false), true
	case "minf":
		return vm.builtinMinMax(argNodes, true, true), true
	case "maxf":
		return vm.builtinMinMax(argNodes, false, true), true

	// String functions
	case "strlen":
		if len(argNodes) < 1 { return IntVal(0), true }
		return IntVal(len(vm.evalNode(argNodes[0]).GetStr())), true
	case "strstr":
		if len(argNodes) < 2 { return IntVal(-1), true }
		a := vm.evalNode(argNodes[0]).GetStr()
		b := vm.evalNode(argNodes[1]).GetStr()
		return IntVal(strings.Index(a, b)), true
	case "strreplace":
		if len(argNodes) < 3 { return StrVal(""), true }
		s := vm.evalNode(argNodes[0]).GetStr()
		old := vm.evalNode(argNodes[1]).GetStr()
		new_ := vm.evalNode(argNodes[2]).GetStr()
		if old == "" { return StrVal(s), true }
		return StrVal(strings.ReplaceAll(s, old, new_)), true
	case "strsplice":
		return vm.builtinStrSplice(argNodes), true
	case "strcode":
		if len(argNodes) < 1 { return IntVal(0), true }
		s := vm.evalNode(argNodes[0]).GetStr()
		idx := 0
		if len(argNodes) > 1 { idx = vm.evalNode(argNodes[1]).GetInt() }
		if idx >= 0 && idx < len(s) { return IntVal(int(s[idx])), true }
		return IntVal(0), true
	case "codestr":
		if len(argNodes) < 1 { return StrVal(""), true }
		return StrVal(string(rune(vm.evalNode(argNodes[0]).GetInt()))), true

	// List operations
	case "at":
		return vm.builtinAt(argNodes), true
	case "listlen":
		if len(argNodes) < 1 { return IntVal(0), true }
		s := vm.evalNode(argNodes[0]).GetStr()
		return IntVal(listLen(s)), true
	case "substr":
		return vm.builtinSubstr(argNodes), true
	case "sublist":
		return vm.builtinSublist(argNodes), true
	case "indexof":
		if len(argNodes) < 2 { return IntVal(-1), true }
		list := vm.evalNode(argNodes[0]).GetStr()
		elem := vm.evalNode(argNodes[1]).GetStr()
		return IntVal(listIndexOf(list, elem)), true
	case "listdel":
		return vm.builtinListDel(argNodes), true
	case "listsplice":
		return vm.builtinListSplice(argNodes), true
	case "prettylist":
		return vm.builtinPrettyList(argNodes), true
	case "listfind":
		return vm.builtinListFind(argNodes), true
	case "looplist":
		return vm.builtinLoopList(argNodes), true
	case "loopsublist":
		return vm.builtinLoopSubList(argNodes), true
	case "loopconcat":
		return vm.builtinLoopConcat(argNodes, true), true
	case "loopconcatword":
		return vm.builtinLoopConcat(argNodes, false), true
	case "looplistconcat":
		return vm.builtinLoopListConcat(argNodes, true), true
	case "looplistconcatword":
		return vm.builtinLoopListConcat(argNodes, false), true
	case "listfilter":
		return vm.builtinListFilter(argNodes), true
	case "sortlist":
		// sortlist is complex, skip for now (rarely used in asset cfgs)
		return StrVal(""), true

	default:
		return NullVal(), false
	}
}

// Helper: evaluate code block (string body or CodeBlock)
func (vm *VM) evalCode(node Node) Value {
	v := vm.evalNode(node)
	if v.Type == ValCode {
		return vm.execBlock(Parse(v.Sval))
	}
	// If it's a string, execute it as code
	return vm.execBlock(Parse(v.GetStr()))
}

func (vm *VM) evalCodeBool(node Node) bool {
	return vm.evalCode(node).GetBool()
}

// Built-in implementations

func (vm *VM) builtinIf(args []Node) Value {
	if len(args) < 2 { return NullVal() }
	cond := vm.evalNode(args[0])
	if cond.GetBool() {
		return vm.evalCode(args[1])
	}
	if len(args) >= 3 {
		return vm.evalCode(args[2])
	}
	return NullVal()
}

func (vm *VM) builtinTernary(args []Node) Value {
	if len(args) < 3 { return NullVal() }
	cond := vm.evalNode(args[0])
	if cond.GetBool() {
		return vm.evalNode(args[1])
	}
	return vm.evalNode(args[2])
}

func (vm *VM) builtinLoop(args []Node) Value {
	if len(args) < 3 { return NullVal() }
	varName := getIdentName(args[0])
	n := vm.evalNode(args[1]).GetInt()
	if n <= 0 { return NullVal() }

	id := vm.getIdent(varName)
	if id == nil {
		id = vm.newIdent(varName, 0)
	}

	id.Push(IntVal(0))
	for i := 0; i < n; i++ {
		id.Val = IntVal(i)
		vm.evalCode(args[2])
	}
	id.Pop()
	return NullVal()
}

func (vm *VM) builtinLoopWhile(args []Node) Value {
	if len(args) < 4 { return NullVal() }
	varName := getIdentName(args[0])
	n := vm.evalNode(args[1]).GetInt()
	if n <= 0 { return NullVal() }

	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	id.Push(IntVal(0))
	for i := 0; i < n; i++ {
		id.Val = IntVal(i)
		if !vm.evalCodeBool(args[2]) { break }
		vm.evalCode(args[3])
	}
	id.Pop()
	return NullVal()
}

func (vm *VM) builtinWhile(args []Node) Value {
	if len(args) < 2 { return NullVal() }
	for i := 0; i < 1000000; i++ { // safety limit
		if !vm.evalCodeBool(args[0]) { break }
		vm.evalCode(args[1])
	}
	return NullVal()
}

func (vm *VM) builtinDo(args []Node) Value {
	if len(args) < 1 { return NullVal() }
	return vm.evalCode(args[0])
}

func (vm *VM) builtinResult(args []Node) Value {
	if len(args) < 1 { return NullVal() }
	v := vm.evalNode(args[0])
	vm.result = v
	return v
}

func (vm *VM) builtinCond(args []Node) Value {
	for i := 0; i+1 < len(args); i += 2 {
		if vm.evalCodeBool(args[i]) {
			return vm.evalCode(args[i+1])
		}
	}
	if len(args)%2 == 1 {
		return vm.evalCode(args[len(args)-1])
	}
	return NullVal()
}

func (vm *VM) builtinCase(args []Node, _ bool) Value {
	if len(args) < 1 { return NullVal() }
	val := vm.evalNode(args[0]).GetInt()
	for i := 1; i+1 < len(args); i += 2 {
		cv := vm.evalNode(args[i])
		if cv.IsNull() || cv.GetInt() == val {
			return vm.evalCode(args[i+1])
		}
	}
	return NullVal()
}

func (vm *VM) builtinCaseF(args []Node) Value {
	if len(args) < 1 { return NullVal() }
	val := vm.evalNode(args[0]).GetFloat()
	for i := 1; i+1 < len(args); i += 2 {
		cv := vm.evalNode(args[i])
		if cv.IsNull() || cv.GetFloat() == val {
			return vm.evalCode(args[i+1])
		}
	}
	return NullVal()
}

func (vm *VM) builtinCaseS(args []Node) Value {
	if len(args) < 1 { return NullVal() }
	val := vm.evalNode(args[0]).GetStr()
	for i := 1; i+1 < len(args); i += 2 {
		cv := vm.evalNode(args[i])
		if cv.IsNull() || cv.GetStr() == val {
			return vm.evalCode(args[i+1])
		}
	}
	return NullVal()
}

func (vm *VM) builtinAlias(args []Node) Value {
	if len(args) < 2 { return NullVal() }
	name := vm.evalNode(args[0]).GetStr()
	val := vm.evalNode(args[1])
	vm.setAlias(name, val)
	return NullVal()
}

func (vm *VM) builtinGetAlias(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	name := vm.evalNode(args[0]).GetStr()
	return StrVal(vm.GetAlias(name))
}

func (vm *VM) builtinSet(args []Node) Value {
	if len(args) < 2 { return NullVal() }
	name := vm.evalNode(args[0]).GetStr()
	val := vm.evalNode(args[1])
	id := vm.getIdent(name)
	if id == nil {
		vm.setAlias(name, val)
	} else {
		switch id.Type {
		case IdAlias:
			id.Val = val
		case IdVar:
			if id.IStorage != nil {
				*id.IStorage = val.GetInt()
			} else {
				id.Val = IntVal(val.GetInt())
			}
		case IdFvar:
			if id.FStorage != nil {
				*id.FStorage = val.GetFloat()
			} else {
				id.Val = FloatVal(val.GetFloat())
			}
		case IdSvar:
			if id.SStorage != nil {
				*id.SStorage = val.GetStr()
			} else {
				id.Val = StrVal(val.GetStr())
			}
		}
	}
	return NullVal()
}

func (vm *VM) builtinLocal(args []Node) Value {
	// Push locals, execute remaining code, pop
	// In our simplified model, local just creates scoped aliases
	var names []string
	for _, a := range args {
		name := vm.evalNode(a).GetStr()
		id := vm.getIdent(name)
		if id == nil {
			id = vm.newIdent(name, 0)
		}
		id.Push(NullVal())
		names = append(names, name)
	}
	// The C++ version continues executing after local; we just scope the names
	// and they'll be popped... but we need the body. In practice, local is
	// followed by more statements in the same block.
	// For correctness, we'd need to restructure, but for cfg files this is rare.
	// We leave names scoped and they'll leak (acceptable for asset extraction).
	_ = names
	return NullVal()
}

func (vm *VM) builtinPush(args []Node) Value {
	if len(args) < 3 { return NullVal() }
	name := getIdentName(args[0])
	val := vm.evalNode(args[1])
	id := vm.getIdent(name)
	if id == nil {
		id = vm.newIdent(name, 0)
	}
	id.Push(val)
	result := vm.evalCode(args[2])
	id.Pop()
	return result
}

// getIdentName extracts a raw identifier name from an AST node without evaluating it.
func getIdentName(node Node) string {
	switch n := node.(type) {
	case *Word:
		return n.Val
	case *StringLit:
		return n.Val
	default:
		return ""
	}
}

func (vm *VM) builtinPushIf(args []Node) Value {
	if len(args) < 3 { return NullVal() }
	name := getIdentName(args[0])
	val := vm.evalNode(args[1])
	if !val.GetBool() { return NullVal() }
	id := vm.getIdent(name)
	if id == nil {
		id = vm.newIdent(name, 0)
	}
	id.Push(val)
	result := vm.evalCode(args[2])
	id.Pop()
	return result
}

func (vm *VM) builtinAppend(args []Node, space bool) Value {
	if len(args) < 2 { return NullVal() }
	name := getIdentName(args[0])
	val := vm.evalNode(args[1])
	id := vm.getIdent(name)
	if id == nil {
		id = vm.newIdent(name, 0)
	}
	existing := id.Val.GetStr()
	add := val.GetStr()
	if existing == "" {
		id.Val = StrVal(add)
	} else if space {
		id.Val = StrVal(existing + " " + add)
	} else {
		id.Val = StrVal(existing + add)
	}
	return NullVal()
}

func (vm *VM) builtinAnd(args []Node) Value {
	if len(args) == 0 { return IntVal(1) }
	var result Value
	for _, a := range args {
		result = vm.evalCode(a)
		if !result.GetBool() { return result }
	}
	return result
}

func (vm *VM) builtinOr(args []Node) Value {
	if len(args) == 0 { return IntVal(0) }
	var result Value
	for _, a := range args {
		result = vm.evalCode(a)
		if result.GetBool() { return result }
	}
	return result
}

func (vm *VM) builtinConcat(args []Node, space bool) Value {
	var parts []string
	for _, a := range args {
		parts = append(parts, vm.evalNode(a).GetStr())
	}
	sep := ""
	if space { sep = " " }
	return StrVal(strings.Join(parts, sep))
}

func (vm *VM) builtinFormat(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	format := vm.evalNode(args[0]).GetStr()
	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			i++
			idx := int(format[i] - '0')
			if idx >= 1 && idx <= 9 && idx < len(args) {
				buf.WriteString(vm.evalNode(args[idx]).GetStr())
			} else {
				buf.WriteByte(format[i])
			}
		} else {
			buf.WriteByte(format[i])
		}
	}
	return StrVal(buf.String())
}

func (vm *VM) builtinEcho(args []Node) Value {
	var parts []string
	for _, a := range args {
		parts = append(parts, vm.evalNode(a).GetStr())
	}
	fmt.Println(strings.Join(parts, " "))
	return NullVal()
}

func (vm *VM) builtinEscape(args []Node) Value {
	if len(args) < 1 { return StrVal("\"\"") }
	s := vm.evalNode(args[0]).GetStr()
	var buf strings.Builder
	buf.WriteByte('"')
	for _, c := range s {
		switch c {
		case '\n': buf.WriteString("^n")
		case '\t': buf.WriteString("^t")
		case '\f': buf.WriteString("^f")
		case '"': buf.WriteString("^\"")
		case '^': buf.WriteString("^^")
		default: buf.WriteRune(c)
		}
	}
	buf.WriteByte('"')
	return StrVal(buf.String())
}

func (vm *VM) builtinUnescape(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	s := vm.evalNode(args[0]).GetStr()
	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '^' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n': buf.WriteByte('\n')
			case 't': buf.WriteByte('\t')
			case 'f': buf.WriteByte('\f')
			default: buf.WriteByte(s[i])
			}
		} else {
			buf.WriteByte(s[i])
		}
	}
	return StrVal(buf.String())
}

func (vm *VM) builtinStrLower(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	return StrVal(strings.ToLower(vm.evalNode(args[0]).GetStr()))
}

func (vm *VM) builtinStrUpper(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	return StrVal(strings.ToUpper(vm.evalNode(args[0]).GetStr()))
}

func (vm *VM) builtinStrSplice(args []Node) Value {
	if len(args) < 4 { return StrVal("") }
	s := vm.evalNode(args[0]).GetStr()
	vals := vm.evalNode(args[1]).GetStr()
	skip := vm.evalNode(args[2]).GetInt()
	count := vm.evalNode(args[3]).GetInt()
	slen := len(s)
	offset := clampInt(skip, 0, slen)
	length := clampInt(count, 0, slen-offset)
	return StrVal(s[:offset] + vals + s[offset+length:])
}

func (vm *VM) builtinRound(args []Node) Value {
	if len(args) < 1 { return FloatVal(0) }
	n := float64(vm.evalNode(args[0]).GetFloat())
	step := 0.0
	if len(args) >= 2 { step = float64(vm.evalNode(args[1]).GetFloat()) }
	if step > 0 {
		n += step * 0.5
		if n < 0 { n -= step }
		n -= math.Mod(n, step)
	} else {
		if n < 0 { n = math.Ceil(n - 0.5) } else { n = math.Floor(n + 0.5) }
	}
	return FloatVal(float32(n))
}

func (vm *VM) builtinMinMax(args []Node, isMin bool, isFloat bool) Value {
	if len(args) == 0 {
		if isFloat { return FloatVal(0) }
		return IntVal(0)
	}
	if isFloat {
		val := vm.evalNode(args[0]).GetFloat()
		for _, a := range args[1:] {
			v := vm.evalNode(a).GetFloat()
			if isMin { if v < val { val = v } } else { if v > val { val = v } }
		}
		return FloatVal(val)
	}
	val := vm.evalNode(args[0]).GetInt()
	for _, a := range args[1:] {
		v := vm.evalNode(a).GetInt()
		if isMin { if v < val { val = v } } else { if v > val { val = v } }
	}
	return IntVal(val)
}

// Helper functions for binary operations
func (vm *VM) builtinIntOp2(args []Node, op func(int, int) int) Value {
	a, b := 0, 0
	if len(args) >= 1 { a = vm.evalNode(args[0]).GetInt() }
	if len(args) >= 2 { b = vm.evalNode(args[1]).GetInt() }
	return IntVal(op(a, b))
}

func (vm *VM) builtinFloatOp2(args []Node, op func(float32, float32) float32) Value {
	var a, b float32
	if len(args) >= 1 { a = vm.evalNode(args[0]).GetFloat() }
	if len(args) >= 2 { b = vm.evalNode(args[1]).GetFloat() }
	return FloatVal(op(a, b))
}

func (vm *VM) builtinIntCmp(args []Node, cmp func(int, int) bool) Value {
	a, b := 0, 0
	if len(args) >= 1 { a = vm.evalNode(args[0]).GetInt() }
	if len(args) >= 2 { b = vm.evalNode(args[1]).GetInt() }
	if cmp(a, b) { return IntVal(1) }
	return IntVal(0)
}

func (vm *VM) builtinFloatCmp(args []Node, cmp func(float32, float32) bool) Value {
	var a, b float32
	if len(args) >= 1 { a = vm.evalNode(args[0]).GetFloat() }
	if len(args) >= 2 { b = vm.evalNode(args[1]).GetFloat() }
	if cmp(a, b) { return IntVal(1) }
	return IntVal(0)
}

func (vm *VM) builtinStrCmp(args []Node, cmp func(string, string) bool) Value {
	a, b := "", ""
	if len(args) >= 1 { a = vm.evalNode(args[0]).GetStr() }
	if len(args) >= 2 { b = vm.evalNode(args[1]).GetStr() }
	if cmp(a, b) { return IntVal(1) }
	return IntVal(0)
}

func (vm *VM) builtinMath1(args []Node, fn func(float64) float64) Value {
	if len(args) < 1 { return FloatVal(0) }
	a := float64(vm.evalNode(args[0]).GetFloat())
	return FloatVal(float32(fn(a)))
}

// List helpers

func (vm *VM) builtinAt(args []Node) Value {
	if len(args) < 2 { return StrVal("") }
	list := vm.evalNode(args[0]).GetStr()
	pos := vm.evalNode(args[1]).GetInt()
	return StrVal(listAt(list, pos))
}

func (vm *VM) builtinSubstr(args []Node) Value {
	if len(args) < 2 { return StrVal("") }
	s := vm.evalNode(args[0]).GetStr()
	start := vm.evalNode(args[1]).GetInt()
	start = clampInt(start, 0, len(s))
	if len(args) >= 3 {
		count := vm.evalNode(args[2]).GetInt()
		count = clampInt(count, 0, len(s)-start)
		return StrVal(s[start : start+count])
	}
	return StrVal(s[start:])
}

func (vm *VM) builtinSublist(args []Node) Value {
	if len(args) < 2 { return StrVal("") }
	list := vm.evalNode(args[0]).GetStr()
	skip := vm.evalNode(args[1]).GetInt()
	count := -1
	if len(args) >= 3 { count = vm.evalNode(args[2]).GetInt() }
	return StrVal(subList(list, skip, count))
}

func (vm *VM) builtinListDel(args []Node) Value {
	if len(args) < 2 { return StrVal("") }
	list := vm.evalNode(args[0]).GetStr()
	del := vm.evalNode(args[1]).GetStr()
	return StrVal(listDel(list, del))
}

func (vm *VM) builtinListSplice(args []Node) Value {
	if len(args) < 4 { return StrVal("") }
	list := vm.evalNode(args[0]).GetStr()
	vals := vm.evalNode(args[1]).GetStr()
	skip := vm.evalNode(args[2]).GetInt()
	count := vm.evalNode(args[3]).GetInt()
	return StrVal(listSplice(list, vals, skip, count))
}

func (vm *VM) builtinPrettyList(args []Node) Value {
	if len(args) < 1 { return StrVal("") }
	list := vm.evalNode(args[0]).GetStr()
	conj := ""
	if len(args) >= 2 { conj = vm.evalNode(args[1]).GetStr() }
	return StrVal(prettyList(list, conj))
}

func (vm *VM) builtinListFind(args []Node) Value {
	if len(args) < 3 { return IntVal(-1) }
	varName := getIdentName(args[0])
	list := vm.evalNode(args[1]).GetStr()
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	id.Push(NullVal())
	items := parseList(list)
	result := -1
	for i, item := range items {
		id.Val = StrVal(item)
		if vm.evalCodeBool(args[2]) {
			result = i
			break
		}
	}
	id.Pop()
	return IntVal(result)
}

func (vm *VM) builtinLoopList(args []Node) Value {
	if len(args) < 3 { return NullVal() }
	varName := getIdentName(args[0])
	list := vm.evalNode(args[1]).GetStr()
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	items := parseList(list)
	if len(items) == 0 { return NullVal() }
	id.Push(NullVal())
	for _, item := range items {
		id.Val = StrVal(item)
		vm.evalCode(args[2])
	}
	id.Pop()
	return NullVal()
}

func (vm *VM) builtinLoopSubList(args []Node) Value {
	if len(args) < 5 { return NullVal() }
	varName := vm.evalNode(args[0]).GetStr()
	list := vm.evalNode(args[1]).GetStr()
	skip := vm.evalNode(args[2]).GetInt()
	count := vm.evalNode(args[3]).GetInt()
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	items := parseList(list)
	if len(items) == 0 { return NullVal() }
	id.Push(NullVal())
	end := len(items)
	if count >= 0 && skip+count < end { end = skip + count }
	for i := skip; i < end && i < len(items); i++ {
		id.Val = StrVal(items[i])
		vm.evalCode(args[4])
	}
	id.Pop()
	return NullVal()
}

func (vm *VM) builtinLoopConcat(args []Node, space bool) Value {
	if len(args) < 3 { return StrVal("") }
	varName := getIdentName(args[0])
	n := vm.evalNode(args[1]).GetInt()
	if n <= 0 { return StrVal("") }
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	id.Push(IntVal(0))
	var parts []string
	for i := 0; i < n; i++ {
		id.Val = IntVal(i)
		v := vm.evalCode(args[2])
		parts = append(parts, v.GetStr())
	}
	id.Pop()
	sep := ""
	if space { sep = " " }
	return StrVal(strings.Join(parts, sep))
}

func (vm *VM) builtinLoopListConcat(args []Node, space bool) Value {
	if len(args) < 3 { return StrVal("") }
	varName := vm.evalNode(args[0]).GetStr()
	list := vm.evalNode(args[1]).GetStr()
	items := parseList(list)
	if len(items) == 0 { return StrVal("") }
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	id.Push(NullVal())
	var parts []string
	for _, item := range items {
		id.Val = StrVal(item)
		v := vm.evalCode(args[2])
		parts = append(parts, v.GetStr())
	}
	id.Pop()
	sep := ""
	if space { sep = " " }
	return StrVal(strings.Join(parts, sep))
}

func (vm *VM) builtinListFilter(args []Node) Value {
	if len(args) < 3 { return StrVal("") }
	varName := vm.evalNode(args[0]).GetStr()
	list := vm.evalNode(args[1]).GetStr()
	items := parseList(list)
	if len(items) == 0 { return StrVal("") }
	id := vm.getIdent(varName)
	if id == nil { id = vm.newIdent(varName, 0) }

	id.Push(NullVal())
	var result []string
	for _, item := range items {
		id.Val = StrVal(item)
		if vm.evalCodeBool(args[2]) {
			result = append(result, item)
		}
	}
	id.Pop()
	return StrVal(strings.Join(result, " "))
}
