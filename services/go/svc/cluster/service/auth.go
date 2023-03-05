package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/ingress"
)

func (c *Cluster) DoAuthChallenge(ctx context.Context, user *User, id string) error {
	if c.auth == nil {
		return nil
	}

	pair, err := c.auth.GetAuthKey(ctx, id)

	if err != nil || pair == nil {
		return fmt.Errorf("no key for client to do auth challenge")
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		return fmt.Errorf("failed to generate auth challenge")
	}

	p := game.Packet{}
	p.PutInt(int32(game.N_AUTHCHAL))
	challengeMessage := game.AuthChallenge{
		Desc:      c.authDomain,
		Id:        0,
		Challenge: challenge.Question,
	}
	p.Put(challengeMessage)
	user.Send(game.GamePacket{
		Channel: 1,
		Data:    p,
	})

	msg, err := user.From.NextTimeout(
		ctx,
		5*time.Second,
		game.N_AUTHANS,
	)
	if err != nil {
		return err
	}

	logger := user.Logger()
	answer := msg.Contents().(*game.AuthAns)

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

	user.Message(game.Blue(fmt.Sprintf("logged in with Discord as %s", authUser.Discord.Reference())))
	logger = user.Logger()
	logger.Info().Msg("logged in with Discord")

	return nil
}

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	user.AnnounceELO()
	if user.Auth == nil {
		user.Message("You are not logged in. Your rating will not be saved.")
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

	go server.setupCubeScript(user.Context(), user)
}
