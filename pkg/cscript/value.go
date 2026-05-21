// Package cscript implements a pure Go CubeScript interpreter.
// It is a comprehensive port of the C++ CubeScript engine from Sauerbraten,
// designed to be thread-safe (each VM instance has independent state).
package cscript

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Value types
const (
	ValNull  = iota
	ValInt
	ValFloat
	ValStr
	ValCode // a block of code: [...]
)

// Value represents a dynamically-typed CubeScript value.
type Value struct {
	Type int
	Ival int
	Fval float32
	Sval string
}

func NullVal() Value   { return Value{Type: ValNull} }
func IntVal(i int) Value    { return Value{Type: ValInt, Ival: i} }
func FloatVal(f float32) Value { return Value{Type: ValFloat, Fval: f} }
func StrVal(s string) Value { return Value{Type: ValStr, Sval: s} }
func CodeVal(s string) Value { return Value{Type: ValCode, Sval: s} }

func (v Value) GetInt() int {
	switch v.Type {
	case ValInt:
		return v.Ival
	case ValFloat:
		return int(v.Fval)
	case ValStr, ValCode:
		return parseInt(v.Sval)
	default:
		return 0
	}
}

func (v Value) GetFloat() float32 {
	switch v.Type {
	case ValFloat:
		return v.Fval
	case ValInt:
		return float32(v.Ival)
	case ValStr, ValCode:
		return parseFloat(v.Sval)
	default:
		return 0
	}
}

func (v Value) GetStr() string {
	switch v.Type {
	case ValStr, ValCode:
		return v.Sval
	case ValInt:
		return intStr(v.Ival)
	case ValFloat:
		return floatStr(v.Fval)
	default:
		return ""
	}
}

func (v Value) GetBool() bool {
	switch v.Type {
	case ValFloat:
		return v.Fval != 0
	case ValInt:
		return v.Ival != 0
	case ValStr, ValCode:
		return getBoolStr(v.Sval)
	default:
		return false
	}
}

func (v Value) IsNull() bool {
	return v.Type == ValNull
}

func getBoolStr(s string) bool {
	if s == "" {
		return false
	}
	switch s[0] {
	case '+', '-':
		if len(s) > 1 {
			if s[1] == '0' {
				// check further
			} else if s[1] == '.' {
				if len(s) > 2 && isDigit(s[2]) {
					return parseFloat(s) != 0
				}
				return true
			} else {
				return true
			}
		}
		fallthrough
	case '0':
		i := parseInt(s)
		if i != 0 {
			return true
		}
		// check for float
		for _, c := range s {
			if c == 'e' || c == 'E' || c == '.' {
				return parseFloat(s) != 0
			}
		}
		return false
	case '.':
		if len(s) > 1 && isDigit(s[1]) {
			return parseFloat(s) != 0
		}
		return true
	default:
		return true
	}
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Support 0x hex
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0
	}
	return int(i)
}

func parseFloat(s string) float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Try hex first
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		i, err := strconv.ParseInt(s, 0, 64)
		if err == nil {
			return float32(i)
		}
	}
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0
	}
	return float32(f)
}

func intStr(v int) string {
	return strconv.Itoa(v)
}

func floatStr(v float32) string {
	if float32(int(v)) == v {
		return fmt.Sprintf("%.1f", v)
	}
	return strconv.FormatFloat(float64(v), 'g', 6, 32)
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

const RAD = math.Pi / 180.0
