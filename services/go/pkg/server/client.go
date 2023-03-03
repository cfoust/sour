package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cfoust/sour/pkg/server/relay"
	"github.com/cfoust/sour/pkg/server/game"
	"github.com/cfoust/sour/pkg/server/protocol/disconnectreason"
	"github.com/cfoust/sour/pkg/server/protocol/nmc"
	"github.com/cfoust/sour/pkg/server/protocol/role"
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
	SessionID           int32
	Ping                int32
	Positions           *relay.Publisher
	Packets             *relay.Publisher
	Authentications     map[string]*Authentication
}

func NewClient(cn uint32) *Client {
	return &Client{
		Player:          game.NewPlayer(cn),
		SessionID:       rng.Int31(),
		Authentications: map[string]*Authentication{},
	}
}

// Resets the client object.
func (c *Client) Reset() {
	c.Player.Reset()
	c.Role = role.None
	c.Joined = false
	c.AuthRequiredBecause = disconnectreason.None
	c.SessionID = rng.Int31()
	c.Ping = 0
	if c.Positions != nil {
		c.Positions.Close()
	}
	if c.Packets != nil {
		c.Packets.Close()
	}
	for domain := range c.Authentications {
		delete(c.Authentications, domain)
	}
}

func (c *Client) String() string {
	return fmt.Sprintf("%s (%d)", c.Name, c.CN)
}

func (c *Client) Send(typ nmc.ID, args ...interface{}) {
	//c.Peer.Send(1, packet.Encode(typ, packet.Encode(args...)))
	panic("TODO")
}
