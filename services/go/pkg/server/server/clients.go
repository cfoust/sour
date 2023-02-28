package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/sauerbraten/waiter/internal/net/packet"
	"github.com/cfoust/sour/pkg/server/enet"
	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/protocol"
	"github.com/cfoust/sour/pkg/server/protocol/cubecode"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/playerstate"
	"github.com/cfoust/sour/pkg/server/protocol/role"
)

type ClientManager struct {
	cs []*Client
}

// Links an ENet peer to a client object. If no unused client object can be found, a new one is created and added to the global set of clients.
func (cm *ClientManager) Add(peer *enet.Peer) *Client {
	// re-use unused client object with low cn
	for _, c := range cm.cs {
		if c.Peer == nil {
			c.Peer = peer
			return c
		}
	}

	cn := uint32(len(cm.cs))
	c := NewClient(cn, peer)
	cm.cs = append(cm.cs, c)
	return c
}

func (cm *ClientManager) GetClientByCN(cn uint32) *Client {
	if int(cn) < 0 || int(cn) >= len(cm.cs) {
		return nil
	}
	return cm.cs[cn]
}

func (cm *ClientManager) GetClientByPeer(peer *enet.Peer) *Client {
	if peer == nil {
		return nil
	}

	for _, c := range cm.cs {
		if c.Peer == peer {
			return c
		}
	}

	return nil
}

func (cm *ClientManager) FindClientByName(name string) *Client {
	name = strings.ToLower(name)
	for _, c := range cm.cs {
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
	for _, c := range cm.cs {
		if c.Peer == nil || (exclude != nil && exclude(c)) {
			continue
		}
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
func (s *Server) SendWelcome(c *Client) {
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
	for _, client := range s.Clients.cs {
		if client != c && client.Peer != nil {
			p = append(p, client.CN, client.State, client.Frags, client.Flags, client.Deaths, int32(client.QuadTimer.TimeLeft()/time.Millisecond), client.ToWire())
		}
	}
	p = append(p, -1)

	// send other client's state (name, team, playermodel)
	for _, client := range s.Clients.cs {
		if client != c && client.Peer != nil {
			p = append(p, nmc.InitializeClient, client.CN, client.Name, client.Team.Name, client.Model)
		}
	}

	c.Send(typ, p...)
}

// Tells other clients that the client disconnected, giving a disconnect reason in case it's not a normal leave.
func (cm *ClientManager) Disconnect(c *Client, reason disconnectreason.ID) {
	if c.Peer == nil {
		return
	}

	cm.Relay(c, nmc.Leave, c.CN)

	msg := ""
	if reason != disconnectreason.None {
		msg = fmt.Sprintf("%s (%s) disconnected because: %s", cm.UniqueName(c), c.Peer.Address.IP, reason)
		cm.Relay(c, nmc.ServerMessage, msg)
	} else {
		msg = fmt.Sprintf("%s (%s) disconnected", cm.UniqueName(c), c.Peer.Address.IP)
	}
	log.Println(cubecode.SanitizeString(msg))
}

// Informs all other clients that a client joined the game.
func (cm *ClientManager) InformOthersOfJoin(c *Client) {
	cm.Relay(c, nmc.InitializeClient, c.CN, c.Name, c.Team.Name, c.Model)
	if c.State == playerstate.Spectator {
		cm.Relay(c, nmc.Spectator, c.CN, 1)
	}
}

func (s *Server) MapChange() {
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

func (s *Server) PrivilegedUsersPacket() (typ nmc.ID, p protocol.Packet, noPrivilegedUsers bool) {
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
	for _, c := range cm.cs {
		if c.Peer == nil {
			continue
		}
		n++
	}
	return
}

func (cm *ClientManager) ForEach(do func(c *Client)) {
	for _, c := range cm.cs {
		if c.Peer == nil {
			continue
		}
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
