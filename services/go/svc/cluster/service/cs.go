package service

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
)

//go:embed purgatory.ogz
var PURGATORY []byte

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

const (
	CONSENT_EXPIRATION = 30 * 24 * time.Hour
	AUTOEXEC_KEY       = "autoexec-%s"
)

const INITIAL_SCRIPT = `
authkey %s _DO_NOTHING_ %s
saveauthkeys

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

func (c *Cluster) saveAutoexecKeys(ctx context.Context, u *User, public string, private string) error {
	err := c.redis.Set(ctx, fmt.Sprintf(AUTOEXEC_KEY, public), private, CONSENT_EXPIRATION).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) waitForConsent(ctx context.Context, u *User, public string) error {
	logger := u.Logger()
	domain := getDomain(c.authDomain)
	private := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("private-%d", time.Now()))))[:10]

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
	u.From.Take(ctx, game.N_CONNECT)

	u.SendServerMessage("run '/do (getservauth)' to allow the server to automatically send maps and assets you are missing")

	serverInfo := u.To.Intercept(game.N_SERVINFO)
	servCmd := u.From.Intercept(game.N_SERVCMD)

	defer serverInfo.Remove()
	defer servCmd.Remove()

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
			if cmd.Command != public {
				msg.Pass()
				continue
			}

			logger.Info().Msg("user consented to autoexec")
			msg.Drop()
			err := c.saveAutoexecKeys(ctx, u, public, private)
			if err != nil {
				return err
			}

			go c.setupCubeScript(ctx, u)
			return nil
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

	msg, err := u.From.Take(ctx, game.N_CONNECT)
	if err != nil {
		logger.Warn().Msg("never got N_CONNECT")
		return err
	}

	connect := msg.Contents().(*game.Connect)
	public := connect.AuthName
	if public == "" {
		public = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("public-%d", time.Now()))))[:10]
		return c.waitForConsent(ctx, u, public)
	}

	private, err := c.redis.Get(ctx, fmt.Sprintf(AUTOEXEC_KEY, public)).Result()
	if err != nil {
		u.SendServerMessage(game.Red(
			"your consent is invalid or expired",
		))
		return c.waitForConsent(ctx, u, public)
	}

	u.Mutex.Lock()
	u.autoexecKey = private
	u.Mutex.Unlock()

	return nil
}

func (u *User) GetAutoexecKey() string {
	u.Mutex.RLock()
	key := u.autoexecKey
	u.Mutex.RUnlock()
	return key
}

func (u *User) HasCubeScript() bool {
	return u.GetAutoexecKey() != ""
}

const TUNNEL_MODE = true

func (u *User) RunCubeScript(ctx context.Context, code string) error {
	key := u.GetAutoexecKey()

	script := fmt.Sprintf(`
// %s
%s
`, key, code)

	if len(script) > game.MAXSTRLEN {
		return fmt.Errorf("script too long (%d > %d)", len(script), game.MAXSTRLEN)
	}

	server := u.GetServer()
	if server == nil {
		return fmt.Errorf("user was not connected to server")
	}

	sendServerInfo(
		u,
		script,
	)
	u.From.Take(ctx, game.N_CONNECT)

	p := game.Packet{}
	p.Put(
		game.N_MAPCHANGE,
		game.MapChange{
			Name:     key,
			Mode:     int(game.MODE_COOP),
			HasItems: 0,
		},
	)
	sendClient(u, p, 1)
	u.From.Take(ctx, game.N_MAPCRC)

	sendServerInfo(
		u,
		"",
	)
	u.From.Take(ctx, game.N_CONNECT)

	return nil
}
