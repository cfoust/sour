package service

import (
	"context"
	"crypto/sha256"
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
authkey %s _DO_NOTHING_ %s
//saveauthkeys

mapstart = [
	if (>= (strstr (getservauth) "%s") 0) [
		do (getservauth)
	]
]

servcmd %s
`

func getDomain(authDomain string) string {
	return fmt.Sprintf("%s-autoexec", authDomain)
}

func (c *Cluster) waitForConsent(ctx context.Context, u *User) error {
	logger := u.Logger()
	domain := getDomain(c.authDomain)
	private := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("private-%d", time.Now()))))[:10]
	public := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("public-%d", time.Now()))))[:10]

	script := fmt.Sprintf(
		INITIAL_SCRIPT,
		public,
		domain,
		private,
		public,
	)

	if len(script) > 260 {
		logger.Fatal().Msgf("script too long %d", len(script))
	}

	sendServerInfo(
		u,
		script,
	)
	u.From.NextTimeout(ctx, DEFAULT_TIMEOUT, game.N_CONNECT)

	serverInfo := u.To.Intercept(game.N_SERVINFO)
	servCmd := u.From.Intercept(game.N_SERVCMD)
	for {
		select {
		case <-u.Context().Done():
			return nil
		case msg := <-serverInfo.Receive():
			info := msg.Message.Contents().(*game.ServerInfo)
			info.Domain = script
			p := game.Packet{}
			p.PutInt(int32(game.N_SERVINFO))
			p.Put(*info)
			msg.Replace(p)
		case msg := <-servCmd.Receive():
			cmd := msg.Message.Contents().(*game.ServCMD)
			logger.Info().Msgf("%+v", cmd)
			if cmd.Command != public {
				msg.Pass()
				continue
			}
			logger.Info().Msg("user consented")
		}
	}
}

func (c *Cluster) setupCubeScript(ctx context.Context, u *User) error {
	logger := u.Logger()
	domain := getDomain(c.authDomain)

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
