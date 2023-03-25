package api

type Map struct {
	WorldSize int32
	GameType  string
}

type Typable interface {
	String() string
	FromString(string)
}

