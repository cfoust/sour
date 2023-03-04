package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/net/packet"
	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/role"

	"github.com/sasha-s/go-deadlock"
)

type ClientManager struct {
	clients []*Client
	mutex   deadlock.RWMutex
}

func (cm *ClientManager) Add(sessionId uint32, outgoing Outgoing) *Client {
	cm.mutex.Lock()
	cn := uint32(len(cm.clients))
	c := NewClient(cn, sessionId, outgoing)
	cm.clients = append(cm.clients, c)
	cm.mutex.Unlock()
	return c
}

func (cm *ClientManager) GetClientByCN(cn uint32) *Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if int(cn) < 0 || int(cn) >= len(cm.clients) {
		return nil
	}

	return cm.clients[cn]
}

func (cm *ClientManager) GetClientByID(sessionId uint32) *Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for _, client := range cm.clients {
		if client.SessionID == sessionId {
			return client
		}
	}

	return nil
}

func (cm *ClientManager) FindClientByName(name string) *Client {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	name = strings.ToLower(name)
	for _, c := range cm.clients {
		if strings.Contains(c.Name, name) {
			return c
		}
	}
	return nil
}

// Send a packet to a client's team, but not the client himself, over the specified channel.
func (cm *ClientManager) SendToTeam(c *Client, typ nmc.ID, args ...interface{}) {
	excludeSelfAndOtherTeams := func(_c *Client) bool {
		return _c == c || _c.Team != c.Team
	}
	cm.broadcast(excludeSelfAndOtherTeams, typ, args...)
}

// Sends a packet to all clients currently in use.
func (cm *ClientManager) Broadcast(typ nmc.ID, args ...interface{}) {
	cm.broadcast(nil, typ, args...)
}

func (cm *ClientManager) broadcast(exclude func(*Client) bool, typ nmc.ID, args ...interface{}) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	for _, c := range cm.clients {
		c.Send(typ, args...)
	}
}

func exclude(c *Client) func(*Client) bool {
	return func(_c *Client) bool {
		return _c == c
	}
}

func (cm *ClientManager) Relay(from *Client, typ nmc.ID, args ...interface{}) {
	cm.broadcast(exclude(from), typ, args...)
}

// Sends 'welcome' information to a newly joined client like map, mode, time left, other players, etc.
func (s *GameServer) SendWelcome(c *Client) {
	typ, p := nmc.Welcome, []interface{}{
		nmc.MapChange, s.Map, s.GameMode.ID(), s.GameMode.NeedsMapInfo(), // currently played mode & map
	}

	p = append(p, nmc.TimeLeft, int32(s.Clock.TimeLeft()/time.Second)) // time left in this round

	if pickupMode, ok := s.GameMode.(game.PickupMode); ok && !s.GameMode.NeedsMapInfo() {
		p = append(p, nmc.PickupList)
		p = append(p, pickupMode.PickupsInitPacket()...)
	}

	// send list of clients which have privilege higher than PRIV_NONE and their respecitve privilege level
	pupTyp, pup, empty := s.PrivilegedUsersPacket()
	if !empty {
		p = append(p, pupTyp, pup)
	}

	if s.Clock.Paused() {
		p = append(p, nmc.PauseGame, 1, -1)
	}

	if teamMode, ok := s.GameMode.(game.TeamMode); ok {
		p = append(p, nmc.TeamInfo)
		teamMode.ForEachTeam(func(t *game.Team) {
			if t.Frags > 0 {
				p = append(p, t.Name, t.Frags)
			}
		})
		p = append(p, "")
	}

	// tell the client what team he was put in by the server
	p = append(p, nmc.SetTeam, c.CN, c.Team.Name, -1)

	// tell the client how to spawn (what health, what armour, what weapons, what ammo, etc.)
	if c.State == playerstate.Spectator {
		p = append(p, nmc.Spectator, c.CN, 1)
	} else {
		// TODO: handle spawn delay (e.g. in ctf modes)
		p = append(p, nmc.SpawnState, c.CN, c.ToWire())
	}

	// send other players' state (frags, flags, etc.)
	p = append(p, nmc.PlayerStateList)
	for _, client := range s.Clients.clients {
		if client != c {
			p = append(p, client.CN, client.State, client.Frags, client.Flags, client.Deaths, int32(client.QuadTimer.TimeLeft()/time.Millisecond), client.ToWire())
		}
	}
	p = append(p, -1)

	// send other client's state (name, team, playermodel)
	for _, client := range s.Clients.clients {
		if client != c {
			p = append(p, nmc.InitializeClient, client.CN, client.Name, client.Team.Name, client.Model)
		}
	}

	c.Send(typ, p...)
}

// Tells other clients that the client disconnected, giving a disconnect reason in case it's not a normal leave.
func (cm *ClientManager) Disconnect(c *Client, reason disconnectreason.ID) {
	cm.Relay(c, nmc.Leave, c.CN)

	msg := ""
	if reason != disconnectreason.None {
		msg = fmt.Sprintf("%s disconnected because: %s", cm.UniqueName(c), reason)
		cm.Relay(c, nmc.ServerMessage, msg)
	} else {
		msg = fmt.Sprintf("%s disconnected", cm.UniqueName(c))
	}
	log.Println(cubecode.SanitizeString(msg))

	cm.mutex.Lock()
	newClients := make([]*Client, 0)
	for _, client := range cm.clients {
		if client == c {
			continue
		}
		newClients = append(newClients, client)
	}
	cm.clients = newClients
	cm.mutex.Unlock()
}

// Informs all other clients that a client joined the game.
func (cm *ClientManager) InformOthersOfJoin(c *Client) {
	cm.Relay(c, nmc.InitializeClient, c.CN, c.Name, c.Team.Name, c.Model)
	if c.State == playerstate.Spectator {
		cm.Relay(c, nmc.Spectator, c.CN, 1)
	}
}

func (s *GameServer) MapChange() {
	s.Clients.ForEach(func(c *Client) {
		c.Player.PlayerState.Reset()
		if c.State == playerstate.Spectator {
			return
		}
		s.Spawn(c)
		c.Send(nmc.SpawnState, c.CN, c.ToWire())
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

func (s *GameServer) PrivilegedUsersPacket() (typ nmc.ID, p protocol.Packet, noPrivilegedUsers bool) {
	q := []interface{}{s.MasterMode}

	s.Clients.ForEach(func(c *Client) {
		if c.Role > role.None {
			q = append(q, c.CN, c.Role)
		}
	})

	q = append(q, -1)

	return nmc.CurrentMaster, packet.Encode(q...), len(q) <= 3
}

// Returns the number of connected clients.
func (cm *ClientManager) NumberOfClientsConnected() (n int) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.clients)
}

func (cm *ClientManager) ForEach(do func(c *Client)) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

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
