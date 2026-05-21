package cscript

const MAXARGS = 25

// Ident types
const (
	IdVar     = iota // int variable
	IdFvar          // float variable
	IdSvar          // string variable
	IdCommand       // built-in command
	IdAlias         // user alias
	IdLocal         // local keyword
)

// Ident flags
const (
	IdfPersist    = 1 << 0
	IdfOverride   = 1 << 1
	IdfHex        = 1 << 2
	IdfReadonly   = 1 << 3
	IdfOverridden = 1 << 4
	IdfUnknown    = 1 << 5
	IdfArg        = 1 << 6
)

// Ident represents a CubeScript identifier (variable, command, or alias).
type Ident struct {
	Type  int
	Name  string
	Index int
	Flags int

	// For aliases
	Val     Value
	Stack   []Value // scoping stack

	// For ID_VAR
	IStorage *int
	MinVal   int
	MaxVal   int

	// For ID_FVAR
	FStorage *float32
	MinFVal  float32
	MaxFVal  float32

	// For ID_SVAR
	SStorage *string

	// For ID_COMMAND
	Args    string // format string like "ss", "ii", "rie", etc.
	ArgMask uint32
	NumArgs int
	Fun     interface{} // the Go callback
}

func (id *Ident) GetStr() string {
	return id.Val.GetStr()
}

func (id *Ident) GetInt() int {
	return id.Val.GetInt()
}

func (id *Ident) GetFloat() float32 {
	return id.Val.GetFloat()
}

func (id *Ident) Push(v Value) {
	id.Stack = append(id.Stack, id.Val)
	id.Val = v
	id.Flags &= ^IdfUnknown
}

func (id *Ident) Pop() {
	if len(id.Stack) == 0 {
		return
	}
	id.Val = id.Stack[len(id.Stack)-1]
	id.Stack = id.Stack[:len(id.Stack)-1]
}
