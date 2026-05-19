package gameserver

import (
	"fmt"

	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
)

type Authentication struct {
	reqID uint32
	name  string
}

// Client is a connected player. It is pure data — no channels, no goroutines.
type Client struct {
	game.Player

	Role                role.ID
	Joined              bool
	AuthRequiredBecause disconnectreason.ID
	SessionID           uint32
	Ping                int32
	Authentications     map[string]*Authentication

	server *Server
}

func NewClient(cn uint32, sessionId uint32) *Client {
	return &Client{
		Player:          game.NewPlayer(cn),
		SessionID:       sessionId,
		Authentications: map[string]*Authentication{},
	}
}

func (c *Client) GrantMaster() {
	c.server._setRole(c, role.Master)
}

func (c *Client) RefreshWelcome() {
	c.server.SendWelcome(c)
}

func (c *Client) String() string {
	return fmt.Sprintf("%s (%d:%d)", c.Name, c.CN, c.SessionID)
}

func (c *Client) Message(text string) {
	c.server.ClientMessage(c, text)
}
