package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/state"
)

func (c *Cluster) DoAuthChallenge(ctx context.Context, user *User, id string) error {
	logger := user.Logger()

	if c.auth == nil {
		return nil
	}

	var dbUser state.User
	err := c.db.WithContext(ctx).Where(state.User{
		UUID: id,
	}).First(&dbUser).Error
	if err != nil {
	    return err
	}

	pair := auth.KeyPair{
		Public: dbUser.PublicKey,
		Private: dbUser.PrivateKey,
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		return fmt.Errorf("failed to generate auth challenge")
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
		return err
	}

	answer := msg.(P.AuthAns)

	if answer.Description != c.authDomain {
		return fmt.Errorf("user provided key for invalid authdomain")
	}

	if !challenge.Check(answer.Answer) {
		user.Message(game.Red("failed to login, please regenerate your key"))
		return fmt.Errorf("client failed auth challenge")
	}

	authUser, err := c.auth.AuthenticateId(ctx, challenge.Id)
	if err != nil {
		user.Message(game.Red("failed to login, please regenerate your key"))
		return fmt.Errorf("could not authenticate by id")
	}

	// XXX we really need to move all the ENet auth to ingress/enet.go...
	user.Authentication <- authUser

	user.Message(game.Blue(fmt.Sprintf("logged in with Discord as %s", authUser.Reference())))
	logger = user.Logger()
	logger.Info().Msg("logged in with Discord")

	return nil
}

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	user.AnnounceELO()

	logger := user.Logger()

	auth := user.GetAuth()
	if auth == nil {
		user.Message("You are not logged in. Your rating will not be saved.")
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

	user.Mutex.Lock()
	user.wasGreeted = true
	user.Mutex.Unlock()

	if user.Connection.Type() != ingress.ClientTypeENet {
		return
	}

	go server.setupCubeScript(user.Ctx(), user)
}
