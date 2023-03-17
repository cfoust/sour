package api

type BVector struct {
	X byte
	Y byte
	Z byte
}

type Color struct {
	R byte
	G byte
	B byte
}

type Vector struct {
	X float32
	Y float32
	Z float32
}

type Map struct {
	WorldSize int32
	GameType  string
}

type Typable interface {
	String() string
	FromString(string)
}

