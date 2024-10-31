package gameserver

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/gameserver/protocol/cubecode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/mastermode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
)

type ServerCommand struct {
	name        string
	argsFormat  string
	aliases     []string
	description string
	minRole     role.ID
	f           func(s *Server, c *Client, args []string)
}

func (cmd *ServerCommand) String() string {
	return fmt.Sprintf("%s %s", cubecode.Green("#"+cmd.name), cmd.argsFormat)
}

func (cmd *ServerCommand) Detailed() string {
	aliases := ""
	if len(cmd.aliases) > 0 {
		aliases = cubecode.Gray(fmt.Sprintf("(alias %s)", strings.Join(cmd.aliases, ", ")))
	}
	return fmt.Sprintf("%s: %s\n%s", cmd.String(), aliases, cmd.description)
}

type ServerCommands struct {
	s       *Server
	byName  map[string]*ServerCommand
	byAlias map[string]*ServerCommand
}

func NewCommands(s *Server, cmds ...*ServerCommand) *ServerCommands {
	sc := &ServerCommands{
		s:       s,
		byName:  map[string]*ServerCommand{},
		byAlias: map[string]*ServerCommand{},
	}
	for _, cmd := range cmds {
		sc.Register(cmd)
	}
	return sc
}

func (sc *ServerCommands) Register(cmd *ServerCommand) {
	sc.byName[cmd.name] = cmd
	sc.byAlias[cmd.name] = cmd
	for _, alias := range cmd.aliases {
		sc.byAlias[alias] = cmd
	}
}

func (sc *ServerCommands) Unregister(cmd *ServerCommand) {
	for _, alias := range cmd.aliases {
		delete(sc.byAlias, alias)
	}
	delete(sc.byAlias, cmd.name)
	delete(sc.byName, cmd.name)
}

func (sc *ServerCommands) PrintCommands(c *Client) {
	helpLines := []string{}
	for _, cmd := range sc.byName {
		if c.Role >= cmd.minRole {
			helpLines = append(helpLines, cmd.String())
		}
	}
	c.Message("available commands: " + strings.Join(helpLines, ", "))
}

func (sc *ServerCommands) Handle(c *Client, msg string) {
	parts := strings.Split(strings.TrimSpace(msg), " ")
	command, args := parts[0], parts[1:]

	switch command {
	case "help", "commands":
		if len(args) == 0 {
			sc.PrintCommands(c)
			return
		}
		name := args[0]
		if strings.HasPrefix(name, "#") {
			name = name[1:]
		}
		if cmd, ok := sc.byAlias[name]; ok {
			c.Message(cmd.Detailed())
		} else {
			c.Message(cubecode.Fail("unknown command '" + name + "'"))
		}

	default:
		cmd, ok := sc.byAlias[command]
		if !ok {
			c.Message(cubecode.Fail("unknown command '" + command + "'"))
			return
		}

		if c.Role < cmd.minRole {
			return
		}

		cmd.f(sc.s, c, args)
	}
}

var ToggleKeepTeams = &ServerCommand{
	name:        "keepteams",
	argsFormat:  "0|1",
	aliases:     []string{"persist", "persistteams"},
	description: "keeps teams the same across map change",
	minRole:     role.Master,
	f: func(s *Server, c *Client, args []string) {
		changed := false
		if len(args) >= 1 {
			val, err := strconv.Atoi(args[0])
			if err != nil || (val != 0 && val != 1) {
				return
			}
			changed = s.KeepTeams != (val == 1)
			s.KeepTeams = val == 1
		}
		if changed {
			if s.KeepTeams {
				s.Clients.Message("teams will be kept")
			} else {
				s.Clients.Message("teams will be shuffled")
			}
		} else {
			if s.KeepTeams {
				c.Message("teams will be kept")
			} else {
				c.Message("teams will be shuffled")
			}
		}
	},
}

var ToggleCompetitiveMode = &ServerCommand{
	name:        "competitive",
	argsFormat:  "0|1",
	aliases:     []string{"comp"},
	description: "in competitive mode, the server waits for all clients to load the map and auto-pauses when a player leaves the game",
	minRole:     role.Master,
	f: func(s *Server, c *Client, args []string) {
		changed := false
		if len(args) >= 1 {
			val, err := strconv.Atoi(args[0])
			if err != nil || (val != 0 && val != 1) {
				return
			}
			changed = s.CompetitiveMode != (val == 1)
			switch val {
			case 1:
				// starts at next map
				s.CompetitiveMode = true
				// but lock server now
				s.SetMasterMode(c, mastermode.Locked)
			default:
				s.CompetitiveMode = false
			}
		}
		if changed {
			if s.CompetitiveMode {
				s.Clients.Message("competitive mode will be enabled with next game")
			} else {
				s.Clients.Message("competitive mode will be disabled with next game")
			}
		} else {
			if s.CompetitiveMode {
				c.Message("competitive mode is on")
			} else {
				c.Message("competitive mode is off")
			}
		}
	},
}

var ToggleReportStats = &ServerCommand{
	name:        "reportstats",
	argsFormat:  "0|1",
	aliases:     []string{"repstats"},
	description: "when enabled, end-game stats of players will be reported at intermission",
	minRole:     role.Admin,
	f: func(s *Server, c *Client, args []string) {
		changed := false
		if len(args) >= 1 {
			val, err := strconv.Atoi(args[0])
			if err != nil || (val != 0 && val != 1) {
				return
			}
			changed = s.ReportStats != (val == 1)
			s.ReportStats = val == 1
		}
		if changed {
			if s.ReportStats {
				s.Clients.Message("stats will be reported at intermission")
			} else {
				s.Clients.Message("stats will not be reported")
			}
		} else {
			if s.ReportStats {
				c.Message("stats reporting is on")
			} else {
				c.Message("stats reporting is off")
			}
		}
	},
}

var SetTimeLeft = &ServerCommand{
	name:        "settime",
	argsFormat:  "[Xm][Ys]",
	aliases:     []string{"time", "settimeleft", "settimeremaining", "timeleft", "timeremaining"},
	description: "sets the time remaining to play to X minutes and Y seconds",
	minRole:     role.Admin,
	f: func(s *Server, c *Client, args []string) {
		if len(args) < 1 {
			return
		}

		d, err := time.ParseDuration(args[0])
		if err != nil {
			c.Message(cubecode.Error("could not parse duration: " + err.Error()))
			return
		}

		if d == 0 {
			d = 1 * time.Second // 0 forces intermission without updating the client's game timer
			s.Message(fmt.Sprintf("%s forced intermission", s.Clients.UniqueName(c)))
		} else {
			s.Message(fmt.Sprintf("%s set the time remaining to %s", s.Clients.UniqueName(c), d))
		}

		s.Clock.SetTimeLeft(d)
	},
}
