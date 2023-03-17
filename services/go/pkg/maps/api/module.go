package api

type BVector struct {
	X byte
	Y byte
	Z byte
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
