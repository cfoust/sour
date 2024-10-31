package mastermode

import "strconv"

type ID int32

const (
	Auth ID = iota - 1
	Open
	Veto
	Locked
	Private
)

func (mm ID) String() string {
	switch mm {
	case Auth:
		return "auth"
	case Open:
		return "open"
	case Veto:
		return "veto"
	case Locked:
		return "locked"
	case Private:
		return "private"
	default:
		return strconv.Itoa(int(mm))
	}
}
