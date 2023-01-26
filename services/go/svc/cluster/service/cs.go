package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
)

func makeServerInfo(u *User, domain string) []byte {
	u.Mutex.RLock()
	info := game.ServerInfo{
		Client:      int(u.Num),
		Protocol:    game.PROTOCOL_VERSION,
		SessionId:   0,
		HasPassword: 0,
		Description: u.lastDescription,
		Domain:      domain,
	}
	u.Mutex.RUnlock()

	p := game.Packet{}
	p.Put(game.N_SERVINFO)
	p.Put(info)
	return p
}

func sendServerInfo(u *User, domain string) {
	u.Send(game.GamePacket{
		Channel: 1,
		Data:    makeServerInfo(u, domain),
	})
}

const DEFAULT_TIMEOUT = 5 * time.Second

const INITIAL_SCRIPT = `
echo test
`

func (c *Cluster) waitForConsent(ctx context.Context, u *User) error {
	sendServerInfo(
		u,
		INITIAL_SCRIPT,
	)
	u.From.NextTimeout(ctx, DEFAULT_TIMEOUT, game.N_CONNECT)

	serverInfo := u.To.Intercept(game.N_SERVINFO)
	for {
		select {
		case <-u.Context().Done():
			return nil
		case msg := <- serverInfo.Receive():
			info := msg.Message.Contents().(*game.ServerInfo)
			info.Domain = INITIAL_SCRIPT
			p := game.Packet{}
			p.PutInt(int32(game.N_SERVINFO))
			p.Put(*info)
			msg.Replace(p)
		}
	}
}

func (c *Cluster) setupCubeScript(ctx context.Context, u *User) error {
	logger := u.Logger()
	domain := fmt.Sprintf("%s-autoexec", c.authDomain)

	// First determine whether they already have an autoexec key
	sendServerInfo(
		u,
		domain,
	)

	msg, err := u.From.NextTimeout(ctx, DEFAULT_TIMEOUT, game.N_CONNECT)
	if err != nil {
		logger.Warn().Msg("never got N_CONNECT")
		return err
	}

	connect := msg.Contents().(*game.Connect)
	logger.Info().Msgf("%+v", connect)
	if connect.AuthName == "" {
		return c.waitForConsent(ctx, u)
	}

	return nil
}
