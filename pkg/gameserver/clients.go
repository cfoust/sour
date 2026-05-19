package gameserver

import (
	"fmt"
	"strconv"
	"time"

	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/protocol/cubecode"
	"github.com/cfoust/sour/pkg/gameserver/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/gameserver/protocol/playerstate"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
)

type ClientManager struct {
	clients []*Client
}

func (cm *ClientManager) Add(sessionId uint32) *Client {
	taken := make(map[uint32]struct{})
	for _, client := range cm.clients {
		taken[client.CN] = struct{}{}
	}

	var cn uint32 = 0
	for {
		if _, ok := taken[cn]; !ok {
			break
		}
		cn++
	}

	c := NewClient(cn, sessionId)
	cm.clients = append(cm.clients, c)
	return c
}

func (cm *ClientManager) GetClientByCN(cn uint32) *Client {
	for _, c := range cm.clients {
		if c.CN == cn {
			return c
		}
	}
	return nil
}

func (cm *ClientManager) GetClientByID(sessionId uint32) *Client {
	for _, client := range cm.clients {
		if client.SessionID == sessionId {
			return client
		}
	}
	return nil
}

func (cm *ClientManager) FindClientByName(name string) *Client {
	for _, c := range cm.clients {
		if cubecode.SanitizeString(c.Name) == cubecode.SanitizeString(name) {
			return c
		}
	}
	return nil
}

func (cm *ClientManager) SendToTeam(c *Client, out *OutputBuffer, messages ...P.Message) {
	for _, _c := range cm.clients {
		if _c == c || _c.Team != c.Team {
			continue
		}
		out.Send(_c.SessionID, messages...)
	}
}

func (cm *ClientManager) MessageAll(out *OutputBuffer, message string) {
	cm.BroadcastTo(out, P.ServerMessage{Text: message})
}

func (cm *ClientManager) BroadcastTo(out *OutputBuffer, messages ...P.Message) {
	for _, c := range cm.clients {
		out.Send(c.SessionID, messages...)
	}
}

func (cm *ClientManager) RelayTo(from *Client, out *OutputBuffer, messages ...P.Message) {
	for _, c := range cm.clients {
		if c == from {
			continue
		}
		out.Send(c.SessionID, messages...)
	}
}

// SendWelcome sends welcome information to a newly joined client.
func (s *Server) SendWelcome(c *Client) {
	messages := []P.Message{
		P.Welcome{},
		P.MapChange{
			Name:     s.Map,
			Mode:     int32(s.GameMode.ID()),
			HasItems: s.GameMode.NeedsMapInfo(),
		},
		P.TimeUp{int32(s.Clock.TimeLeft(s.gameClock) / time.Second)},
	}

	if pickupMode, ok := s.GameMode.(game.PickupMode); ok && !s.GameMode.NeedsMapInfo() {
		messages = append(messages, pickupMode.PickupsInitPacket())
	}

	privileged, empty := s.PrivilegedUsersPacket()
	if !empty {
		messages = append(messages, privileged)
	}

	if s.Clock.Paused() {
		messages = append(messages, P.PauseGame{Paused: true, Client: -1})
	}

	if teamMode, ok := s.GameMode.(game.TeamMode); ok {
		teamInfo := P.TeamInfo{}
		teamMode.ForEachTeam(func(t *game.Team) {
			if t.Frags > 0 {
				teamInfo.Teams = append(teamInfo.Teams, P.Team{Team: t.Name, Frags: t.Frags})
			}
		})
		messages = append(messages, teamInfo)
	}

	messages = append(messages, P.SetTeam{
		Client: int32(c.CN),
		Team:   c.Team.Name,
		Reason: -1,
	})

	if c.State == playerstate.Spectator {
		messages = append(messages, P.Spectator{
			Client:     int32(c.CN),
			Spectating: true,
		})
	} else {
		messages = append(messages, P.SpawnState{
			Client:      int32(c.CN),
			EntityState: c.ToWire(),
		})
	}

	resume := P.Resume{}
	for _, client := range s.Clients.clients {
		if client != c {
			resume.Clients = append(
				resume.Clients,
				P.ClientState{
					Id:          int32(client.CN),
					State:       int32(client.State),
					Frags:       client.Frags,
					Flags:       client.Flags,
					Deaths:      client.Deaths,
					Quadmillis:  int32(client.QuadDeadline.TimeLeftMs(s.gameClock)),
					EntityState: client.ToWire(),
				},
			)
		}
	}
	messages = append(messages, resume)

	for _, client := range s.Clients.clients {
		if client != c {
			messages = append(messages, P.InitClient{
				Client:      int32(client.CN),
				Name:        client.Name,
				Team:        client.Team.Name,
				Playermodel: int32(client.Model),
			})
		}
	}

	s.out.Send(c.SessionID, messages...)
}

func (cm *ClientManager) Disconnect(c *Client, out *OutputBuffer, reason disconnectreason.ID) {
	cm.RelayTo(c, out, P.ClientDisconnected{Client: int32(c.CN)})

	if reason != disconnectreason.None {
		msg := fmt.Sprintf("%s disconnected because: %s", cm.UniqueName(c), reason)
		cm.RelayTo(c, out, P.ServerMessage{Text: msg})
	}

	newClients := make([]*Client, 0, len(cm.clients))
	for _, client := range cm.clients {
		if client == c {
			continue
		}
		newClients = append(newClients, client)
	}
	cm.clients = newClients
}

func (cm *ClientManager) InformOthersOfJoin(c *Client, out *OutputBuffer) {
	cm.RelayTo(c, out, P.InitClient{
		Client:      int32(c.CN),
		Name:        c.Name,
		Team:        c.Team.Name,
		Playermodel: int32(c.Model),
	})

	if c.State == playerstate.Spectator {
		cm.RelayTo(c, out, P.Spectator{
			Client:     int32(c.CN),
			Spectating: true,
		})
	}
}

func (s *Server) MapChange() {
	s.Clients.ForEach(func(c *Client) {
		c.Player.PlayerState.Reset()
		if c.State == playerstate.Spectator {
			return
		}
		s.Spawn(c)
		s.out.Send(c.SessionID, P.SpawnState{
			Client:      int32(c.CN),
			EntityState: c.ToWire(),
		})
	})
}

func (cm *ClientManager) PrivilegedUsers() (privileged []*Client) {
	cm.ForEach(func(c *Client) {
		if c.Role > role.None {
			privileged = append(privileged, c)
		}
	})
	return
}

func (s *Server) PrivilegedUsersPacket() (P.Message, bool) {
	message := P.CurrentMaster{
		MasterMode: int32(s.MasterMode),
	}

	s.Clients.ForEach(func(c *Client) {
		if c.Role > role.None {
			message.Clients = append(message.Clients, P.ClientPrivilege{
				Client:    int32(c.CN),
				Privilege: int32(c.Role),
			})
		}
	})

	return message, len(message.Clients) == 0
}

func (cm *ClientManager) GetNumClients() int {
	return len(cm.clients)
}

func (cm *ClientManager) ForEach(do func(c *Client)) {
	for _, c := range cm.clients {
		do(c)
	}
}

func (cm *ClientManager) UniqueName(c *Client) string {
	unique := true
	cm.ForEach(func(_c *Client) {
		if _c != c && _c.Name == c.Name {
			unique = false
		}
	})

	if !unique {
		return c.Name + cubecode.Magenta(" ("+strconv.FormatUint(uint64(c.CN), 10)+")")
	}
	return c.Name
}
