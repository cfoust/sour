package gameserver

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver/game"
	"github.com/cfoust/sour/pkg/gameserver/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/gameserver/protocol/role"
	"github.com/cfoust/sour/pkg/gameserver/relay"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

type Authentication struct {
	reqID uint32
	name  string
}

// Describes a client.
type Client struct {
	game.Player

	Role                role.ID
	Joined              bool                // true if the player is actually in the game
	AuthRequiredBecause disconnectreason.ID // e.g. server is in private mode
	SessionID           uint32
	Ping                int32
	Positions           *relay.Publisher
	Packets             *relay.Publisher
	Authentications     map[string]*Authentication

	connected chan bool
	outgoing  Outgoing

	server *Server
}

func NewClient(cn uint32, sessionId uint32, outgoing Outgoing) *Client {
	return &Client{
		Player:          game.NewPlayer(cn),
		SessionID:       sessionId,
		Authentications: map[string]*Authentication{},
		outgoing:        outgoing,
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
	c.Send(protocol.ServerMessage{Text: text})
}

func (c *Client) Send(messages ...protocol.Message) {
	c.outgoing <- ServerPacket{
		Session:  c.SessionID,
		Channel:  1,
		Messages: messages,
	}
}
