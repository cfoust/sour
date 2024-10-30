package service

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
	C "github.com/cfoust/sour/pkg/game/constants"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/utils"
)

//go:embed purgatory.ogz
var PURGATORY []byte

func sendServerInfo(ctx context.Context, u *User, domain string) (P.Connect, error) {
	info := u.GetServerInfo()
	info.Domain = domain

	msg, err := u.Response(
		ctx,
		P.N_CONNECT,
		info,
	)
	if err != nil {
		return P.Connect{}, err
	}

	connect := msg.(P.Connect)

	return connect, err
}

const (
	CONSENT_EXPIRATION = 30 * 24 * time.Hour
	AUTOEXEC_KEY       = "autoexec-%s"
)

const INITIAL_SCRIPT = `
authkey %s _DO_NOTHING_ %s
saveauthkeys

autoauth 1

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
	private := utils.HashString(fmt.Sprintf("private-%d", time.Now()))

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

	_, err := sendServerInfo(
		ctx,
		u,
		script,
	)
	if err != nil {
		return err
	}

	u.Message("run '/do (getservauth)' to allow the server to securely send maps and assets you are missing")

	serverInfo := u.To.Intercept(P.N_SERVINFO)
	servCmd := u.From.Intercept(P.N_SERVCMD)

	defer serverInfo.Remove()
	defer servCmd.Remove()

	for {
		select {
		case <-u.Ctx().Done():
			return nil
		case msg := <-serverInfo.Receive():
			info := msg.Message.(P.ServerInfo)
			info.Domain = script
			msg.Replace(info)
		case msg := <-servCmd.Receive():
			cmd := msg.Message.(P.ServCMD)
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
	domain := getDomain(c.authDomain)

	// First determine whether they already have an autoexec key
	connect, err := sendServerInfo(
		ctx,
		u,
		domain,
	)
	if err != nil {
		return err
	}

	public := connect.AuthName
	if public == "" {
		public = utils.HashString(fmt.Sprintf("public-%d", time.Now()))[:10]
		return c.waitForConsent(ctx, u, public)
	}

	private, err := c.redis.Get(ctx, fmt.Sprintf(AUTOEXEC_KEY, public)).Result()
	if err != nil {
		u.Message(game.Red(
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

	if len(script) > C.MAXSTRLEN {
		return fmt.Errorf("script too long (%d > %d)", len(script), C.MAXSTRLEN)
	}

	server := u.GetServer()
	if server == nil {
		return fmt.Errorf("user was not connected to server")
	}

	_, err := sendServerInfo(
		ctx,
		u,
		script,
	)
	if err != nil {
		return err
	}

	u.Send(P.MapChange{
		Name:     key,
		Mode:     int32(C.MODE_COOP),
		HasItems: false,
	})
	u.From.Take(ctx, P.N_MAPCRC)

	_, err = sendServerInfo(
		ctx,
		u,
		"",
	)
	return err
}
