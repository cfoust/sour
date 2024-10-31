package role

import (
	"strconv"
	"strings"
)

type ID int32

const (
	None ID = iota
	Master
	Auth
	Admin
)

func Parse(s string) ID {
	switch strings.ToLower(s) {
	case "none":
		return None
	case "master":
		return Master
	case "auth":
		return Auth
	case "admin":
		return Admin
	default:
		return -1
	}
}

func (r ID) String() string {
	switch r {
	case None:
		return "none"
	case Master:
		return "master"
	case Auth:
		return "auth"
	case Admin:
		return "admin"
	default:
		return strconv.Itoa(int(r))
	}
}
