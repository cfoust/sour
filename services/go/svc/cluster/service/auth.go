package service

import (
	"context"
	"fmt"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/auth"
)

func (server *Cluster) DoAuthChallenge(ctx context.Context, user *User, id string) {
	if server.auth == nil {
		return
	}

	pair, err := server.auth.GetAuthKey(ctx, id)

	logger := user.Logger()

	if err != nil || pair == nil {
		logger.Warn().
			Err(err).
			Msg("no key for client to do auth challenge")
		return
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("failed to generate auth challenge")
		return
	}

	user.Mutex.Lock()
	user.Challenge = challenge
	user.Mutex.Unlock()

	p := game.Packet{}
	p.PutInt(int32(game.N_AUTHCHAL))
	challengeMessage := game.AuthChallenge{
		Desc:      server.authDomain,
		Id:        0,
		Challenge: challenge.Question,
	}
	p.Put(challengeMessage)
	user.Send(game.GamePacket{
		Channel: 1,
		Data:    p,
	})
}

func (server *Cluster) HandleChallengeAnswer(
	ctx context.Context,
	user *User,
	challenge *auth.Challenge,
	answer string,
) {
	logger := user.Logger()
	if !challenge.Check(answer) {
		logger.Warn().Msg("client failed auth challenge")
		user.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	authUser, err := server.auth.AuthenticateId(ctx, challenge.Id)
	if err != nil {
		logger.Warn().Err(err).Msg("could not authenticate by id")
		user.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	// XXX we really need to move all the ENet auth to ingress/enet.go...
	user.Authentication <- authUser

	user.SendServerMessage(game.Blue(fmt.Sprintf("logged in with Discord as %s", authUser.Discord.Reference())))
	logger = user.Logger()
	logger.Info().Msg("logged in with Discord")
}

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	user.AnnounceELO()
	if user.Auth == nil {
		user.SendServerMessage("You are not logged in. Your rating will not be saved.")
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
	user.SendServerMessage(message)
}
