package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/cmd/server/auth"
	"github.com/cfoust/sour/cmd/server/state"
	"github.com/cfoust/sour/pkg/game"
	P "github.com/cfoust/sour/pkg/game/protocol"
)

func (c *Cluster) DoAuthChallenge(ctx context.Context, user *User, id string) (*state.User, error) {
	logger := user.Logger()

	var dbUser state.User
	err := c.db.WithContext(ctx).Where(state.User{
		UUID: id,
	}).First(&dbUser).Error
	if err != nil {
		return nil, err
	}

	pair := auth.KeyPair{
		Public:  dbUser.PublicKey,
		Private: dbUser.PrivateKey,
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth challenge")
	}

	user.Send(P.AuthChallenge{
		Desc:      c.authDomain,
		Id:        0,
		Challenge: challenge.Question,
	})

	msg, err := user.From.NextTimeout(
		ctx,
		5*time.Second,
		P.N_AUTHANS,
	)
	if err != nil {
		return nil, err
	}

	answer := msg.(P.AuthAns)

	if answer.Description != c.authDomain {
		return nil, fmt.Errorf("user provided key for invalid authdomain")
	}

	if !challenge.Check(answer.Answer) {
		user.Message(game.Red("failed to log in, please regenerate your key"))
		return nil, fmt.Errorf("client failed auth challenge")
	}

	authUser, err := c.auth.AuthenticateId(ctx, challenge.Id)
	if err != nil {
		user.Message(game.Red("failed to log in, please regenerate your key"))
		return nil, fmt.Errorf("could not authenticate by id")
	}

	user.Message(game.Blue(fmt.Sprintf("logged in with Discord as %s", authUser.Reference())))
	logger = user.Logger()
	logger.Info().Msg("logged in with Discord")

	return authUser, nil
}

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	user.AnnounceELO()

	logger := user.Logger()

	auth := user.GetAuth()
	if auth == nil {
		user.Message("you are not logged in. your rating will not be saved.")
	} else {
		// Associate with the session
		user.sessionLog.UserID = auth.ID
		err := server.db.WithContext(ctx).Save(user.sessionLog).Error
		if err != nil {
			logger.Error().Err(err).Msg("failed to associate user with session")
		}
	}

	server.NotifyClientChange(ctx, user, true)

	server.Users.Mutex.RLock()
	message := "users online: "
	users := server.Users.Users
	for i, other := range users {
		if other == user {
			message += "you"
		} else {
			message += other.Reference()
		}
		if i != len(users)-1 {
			message += ", "
		}
	}
	server.Users.Mutex.RUnlock()
	user.Message(message)
}

// Probe the user's authentication and then transfer control to the CubeScript
// runner.
func (c *Cluster) HandleDesktopLogin(ctx context.Context, user *User) error {
	logger := user.Logger()
	msg, err := user.From.WaitTimeout(
		ctx,
		15*time.Second,
		P.N_CONNECT,
	)
	if err != nil {
		return err
	}

	connect := msg.(P.Connect)

	err = user.SetName(ctx, connect.Name)
	if err != nil {
		return err
	}

	if c.auth != nil {
		// Now attempt authentication
		info := user.GetServerInfo()
		info.Domain = c.authDomain
		msg, err = user.Response(
			ctx,
			P.N_CONNECT,
			info,
		)
		if err != nil {
			return err
		}

		if len(connect.AuthName) > 0 {
			connect = msg.(P.Connect)
			authUser, err := c.DoAuthChallenge(ctx, user, connect.AuthName)
			if err != nil {
				logger.Error().Err(err).Msg("user failed to log in")
			}

			if authUser != nil {
				err := user.HandleAuthentication(ctx, authUser)
				if err != nil {
					return err
				}
			}
		}
	}

	c.GreetClient(ctx, user)

	go c.setupCubeScript(user.Ctx(), user)
	return nil
}
