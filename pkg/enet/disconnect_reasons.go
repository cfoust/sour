package enet

import "strconv"

type ID uint32

const (
	None ID = iota
	EOP
	LocalMode // LOCAL
	Kick
	MessageError // MSGERR
	IPBanned     // IPBAN
	PrivateMode  // PRIVATE
	Full         // MAXCLIENTS
	Timeout
	Overflow
	WrongPassword // PASSWORD
)

func (dr ID) String() string {
	switch dr {
	case None:
		return ""
	case EOP:
		return "end of packet"
	case LocalMode:
		return "server is in local mode"
	case Kick:
		return "kicked/banned"
	case MessageError:
		return "message error"
	case IPBanned:
		return "ip is banned"
	case PrivateMode:
		return "server is in private mode"
	case Full:
		return "server full"
	case Timeout:
		return "connection timed out"
	case Overflow:
		return "overflow"
	case WrongPassword:
		return "invalid password"
	default:
		return strconv.Itoa(int(dr))
	}
}
