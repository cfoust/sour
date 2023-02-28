package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/mastermode"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/role"
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
	c.Send(nmc.ServerMessage, "available commands: "+strings.Join(helpLines, ", "))
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
			c.Send(nmc.ServerMessage, cmd.Detailed())
		} else {
			c.Send(nmc.ServerMessage, cubecode.Fail("unknown command '"+name+"'"))
		}

	default:
		cmd, ok := sc.byAlias[command]
		if !ok {
			c.Send(nmc.ServerMessage, cubecode.Fail("unknown command '"+command+"'"))
			return
		}

		if c.Role < cmd.minRole {
			return
		}

		cmd.f(sc.s, c, args)
	}
}

var QueueMap = &ServerCommand{
	name:        "queuemap",
	argsFormat:  "[map...]",
	aliases:     []string{"queued", "queue", "queuedmap", "queuemaps", "queuedmaps", "mapqueue", "mapsqueue"},
	description: "prints the current queue or adds the map(s) to the queue",
	minRole:     role.Master,
	f: func(s *Server, c *Client, args []string) {
		for _, mapp := range args {
			err := s.MapRotation.QueueMap(s.GameMode.ID(), mapp)
			if err != "" {
				c.Send(nmc.ServerMessage, cubecode.Fail(err))
			}
		}
		queuedMaps := s.MapRotation.QueuedMaps()
		switch len(queuedMaps) {
		case 0:
			c.Send(nmc.ServerMessage, "no maps queued")
		case 1:
			c.Send(nmc.ServerMessage, "queued map: "+queuedMaps[0])
		default:
			c.Send(nmc.ServerMessage, "queued maps: "+strings.Join(queuedMaps, ", "))
		}
	},
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
				s.Clients.Broadcast(nmc.ServerMessage, "teams will be kept")
			} else {
				s.Clients.Broadcast(nmc.ServerMessage, "teams will be shuffled")
			}
		} else {
			if s.KeepTeams {
				c.Send(nmc.ServerMessage, "teams will be kept")
			} else {
				c.Send(nmc.ServerMessage, "teams will be shuffled")
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
				s.Clients.Broadcast(nmc.ServerMessage, "competitive mode will be enabled with next game")
			} else {
				s.Clients.Broadcast(nmc.ServerMessage, "competitive mode will be disabled with next game")
			}
		} else {
			if s.CompetitiveMode {
				c.Send(nmc.ServerMessage, "competitive mode is on")
			} else {
				c.Send(nmc.ServerMessage, "competitive mode is off")
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
				s.Clients.Broadcast(nmc.ServerMessage, "stats will be reported at intermission")
			} else {
				s.Clients.Broadcast(nmc.ServerMessage, "stats will not be reported")
			}
		} else {
			if s.ReportStats {
				c.Send(nmc.ServerMessage, "stats reporting is on")
			} else {
				c.Send(nmc.ServerMessage, "stats reporting is off")
			}
		}
	},
}

var LookupIPs = &ServerCommand{
	name:        "ip",
	argsFormat:  "[name|cn]...",
	aliases:     []string{"ips"},
	description: "prints the IP of the player(s) identified by their name or cn, or your own when called with no argument",
	minRole:     role.Admin,
	f: func(s *Server, c *Client, args []string) {
		if len(args) < 1 {
			args = []string{c.Name}
		}
		for _, query := range args {
			var target *Client
			// try CN
			cn, err := strconv.Atoi(query)
			if err == nil {
				target = s.Clients.GetClientByCN(uint32(cn))
			}
			if err != nil || target == nil {
				target = s.Clients.FindClientByName(query)
			}

			if target != nil {
				c.Send(nmc.ServerMessage, fmt.Sprintf("%s has IP %s", s.Clients.UniqueName(target), target.Peer.Address.IP))
			} else {
				c.Send(nmc.ServerMessage, fmt.Sprintf("could not find a client matching '%s'", query))
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
			c.Send(nmc.ServerMessage, cubecode.Error("could not parse duration: "+err.Error()))
			return
		}

		if d == 0 {
			d = 1 * time.Second // 0 forces intermission without updating the client's game timer
			s.Broadcast(nmc.ServerMessage, fmt.Sprintf("%s forced intermission", s.Clients.UniqueName(c)))
		} else {
			s.Broadcast(nmc.ServerMessage, fmt.Sprintf("%s set the time remaining to %s", s.Clients.UniqueName(c), d))
		}

		s.Clock.SetTimeLeft(d)
	},
}

var RegisterPubkey = &ServerCommand{
	name:        "register",
	argsFormat:  "[name] pubkey",
	aliases:     []string{},
	description: "registers an account with the stats server (if you are gauthed you can omit the name and you gauth name will be used)",
	minRole:     role.None,
	f: func(s *Server, c *Client, args []string) {
		if s.StatsServer == nil {
			c.Send(nmc.ServerMessage, cubecode.Error("not connected to stats server"))
			return
		}

		if s.StatsServerAdmin == nil {
			c.Send(nmc.ServerMessage, cubecode.Error("no admin connection to stats server"))
		}

		if statsAuth, ok := c.Authentications[s.StatsServerAuthDomain]; ok {
			c.Send(nmc.ServerMessage, cubecode.Fail("you're already authenticated with "+s.StatsServerAuthDomain+" as "+statsAuth.name))
			return
		}

		if len(args) < 1 {
			c.Send(nmc.ServerMessage, cubecode.Fail(fmt.Sprintf("follow the instructions at https://%s/gen/%s", s.StatsServerAuthDomain, c.Name)))
			return
		}

		name, pubkey := "", ""
		if len(args) < 2 {
			gauth, ok := c.Authentications[""]
			if !ok {
				c.Send(nmc.ServerMessage, cubecode.Fail("you have to claim gauth (to use your gauth name) or provide a name: #register [name] <pubkey>"))
				return
			}
			name, pubkey = gauth.name, args[0]
		} else {
			name, pubkey = args[0], args[1]
		}

		if pubkey == "" {
			c.Send(nmc.ServerMessage, cubecode.Fail("you have to provide your public key: #register [name] <pubkey>"))
			return
		}

		s.StatsServerAdmin.AddAuth(name, pubkey,
			func(err string) {
				if err != "" {
					c.Send(nmc.ServerMessage, cubecode.Error("creating your account failed: "+err))
					return
				}
				c.Send(nmc.ServerMessage, cubecode.Green("you successfully registered as "+name))
				c.Send(nmc.ServerMessage, cubecode.Fail("this is alpha functionality, the account will be lost at stats server restart!"))
				c.Send(nmc.ServerMessage, "type '/autoauth 1', then '/reconnect' to try out your new key")
			},
		)
	},
}

var CheckAuthStatus = &ServerCommand{
	name:        "auth",
	argsFormat:  "[name|cn]...",
	aliases:     []string{"checkauth", "authstatus", "auths"},
	description: "prints the auth status of the player(s) identified by their name or cn, or your own when called with no argument",
	minRole:     role.Auth,
	f: func(s *Server, c *Client, args []string) {
		if len(args) < 1 {
			args = []string{c.Name}
		}

		for _, query := range args {
			var target *Client
			// try CN
			cn, err := strconv.Atoi(query)
			if err == nil {
				target = s.Clients.GetClientByCN(uint32(cn))
			}
			if err != nil || target == nil {
				target = s.Clients.FindClientByName(query)
			}

			if target != nil {
				if len(c.Authentications) == 0 {
					c.Send(nmc.ServerMessage, fmt.Sprintf("%s has not authenticated", s.Clients.UniqueName(target)))
					continue
				}
				auths := []string{}
				// always put gauth first
				if auth, ok := c.Authentications[""]; ok {
					auths = append(auths, fmt.Sprintf("'%s'", cubecode.Magenta(auth.name)))
				}
				for domain, auth := range c.Authentications {
					if domain == "" {
						continue
					}
					auths = append(auths, fmt.Sprintf("'%s' [%s]", cubecode.Magenta(auth.name), cubecode.Green(domain)))
				}
				c.Send(nmc.ServerMessage, fmt.Sprintf("%s authenticated as %s", s.Clients.UniqueName(target), strings.Join(auths, ", ")))
			} else {
				c.Send(nmc.ServerMessage, fmt.Sprintf("could not find a client matching '%s'", query))
			}
		}
	},
}
