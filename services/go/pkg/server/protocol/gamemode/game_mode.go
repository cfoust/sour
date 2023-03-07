package gamemode

import (
	"strings"
)

type ID int32

const Unknown ID = -1

const (
	FFA ID = iota
	CoopEdit
	Teamplay
	Insta
	InstaTeam
	Effic // 5
	EfficTeam
	Tactics
	TacticsTeam
	Capture
	RegenCapture // 10
	CTF
	InstaCTF
	Protect
	InstaProtect
	Hold // 15
	InstaHold
	EfficCTF
	EfficProtect
	EfficHold
	Collect // 20
	InstaCollect
	EfficCollect
)

func (gm ID) String() string {
	switch gm {
	case FFA:
		return "ffa"
	case CoopEdit:
		return "coop edit"
	case Teamplay:
		return "teamplay"
	case Insta:
		return "insta"
	case InstaTeam:
		return "insta team"
	case Effic:
		return "effic"
	case EfficTeam:
		return "effic team"
	case Tactics:
		return "tactics"
	case TacticsTeam:
		return "tactics team"
	case Capture:
		return "capture"
	case RegenCapture:
		return "regen capture"
	case CTF:
		return "ctf"
	case InstaCTF:
		return "insta ctf"
	case Protect:
		return "protect"
	case InstaProtect:
		return "insta protect"
	case Hold:
		return "hold"
	case InstaHold:
		return "insta hold"
	case EfficCTF:
		return "effic ctf"
	case EfficProtect:
		return "effic protect"
	case EfficHold:
		return "effic hold"
	case Collect:
		return "collect"
	case InstaCollect:
		return "insta collect"
	case EfficCollect:
		return "effic collect"
	default:
		return "unknown"
	}
}

func Parse(s string) ID {
	switch strings.ToLower(s) {
	case "ffa", "free for all":
		return FFA
	case "coop", "coop edit", "edit":
		return CoopEdit
	case "teamplay":
		return Teamplay
	case "i", "insta", "instagib":
		return Insta
	case "i team", "iteam", "insta team", "instateam", "instagib team", "instagibteam":
		return InstaTeam
	case "e", "effic", "efficiency":
		return Effic
	case "e team", "eteam", "effic team", "efficteam", "efficiency team", "efficiencyteam":
		return EfficTeam
	case "tac", "tactics":
		return Tactics
	case "tac team", "tacteam", "tactics team", "tacticsteam":
		return TacticsTeam
	case "cap", "capture":
		return Capture
	case "rcap", "regen", "regen cap", "regen capture":
		return RegenCapture
	case "ctf", "capture the flag":
		return CTF
	case "i ctf", "ictf", "insta ctf", "instactf", "instagib ctf", "instagibctf", "instagib capture the flag":
		return InstaCTF
	case "protect":
		return Protect
	case "i protect", "iprotect", "insta protect", "instaprotect", "instagib protect", "instagibprotect":
		return InstaProtect
	case "hold":
		return Hold
	case "i hold", "ihold", "insta hold", "instahold", "instagib hold", "instagibhold":
		return InstaHold
	case "e ctf", "ectf", "effic ctf", "efficctf", "efficiency ctf", "efficiencyctf", "efficiency capture the flag":
		return EfficCTF
	case "e protect", "eprotect", "effic protect", "efficprotect", "efficiency protect", "efficiencyprotect":
		return EfficProtect
	case "e hold", "ehold", "effic hold", "effichold", "efficiency hold", "efficiencyhold":
		return EfficHold
	case "coll", "collect":
		return Collect
	case "i coll", "icoll", "i collect", "icollect", "insta collect", "instacollect", "instagib collect", "instagibcollect":
		return InstaCollect
	case "e coll", "ecoll", "e collect", "ecollect", "effic collect", "efficcollect", "efficiency collect", "efficiencycollect":
		return EfficCollect
	default:
		return Unknown
	}
}

func Valid(gm ID) bool {
	switch gm {
	case FFA, CoopEdit, Insta, Effic, Tactics,
		Teamplay, InstaTeam, EfficTeam, TacticsTeam,
		CTF, InstaCTF, EfficCTF:
		return true
	default:
		return false
	}
}

func IsCTF(gm ID) bool {
	switch gm {
	case InstaCTF, EfficCTF:
		return true
	default:
		return false
	}
}

func IsCapture(gm ID) bool {
	switch gm {
	case Capture, RegenCapture:
		return true
	default:
		return false
	}
}
