package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cfoust/sour/svc/cluster/verse"
)

func (c *Cluster) GoHome(ctx context.Context, user *User) error {
	var err error
	var space *verse.UserSpace
	authUser := user.GetAuth()
	isLoggedIn := authUser != nil
	if authUser == nil {
		space, err = c.verse.NewSpace(ctx, "")
		if err != nil {
			return err
		}

		err = space.Expire(ctx, time.Hour*4)
		if err != nil {
			return err
		}
	} else {
		space, err = user.Verse.GetHomeSpace(ctx)
		if err != nil {
			return err
		}
	}

	instance, err := c.spaces.StartSpace(ctx, space.GetID())
	if err != nil {
	    return err
	}

	_, err = user.ConnectToSpace(instance.Server, space.GetID())

	message := fmt.Sprintf(
		"welcome to your home (space %s).",
		space.GetID(),
	)

	if isLoggedIn {
		user.SendServerMessage(message)
		user.SendServerMessage("editing by others is disabled. say #openedit to enable it.")
	} else {
		user.SendServerMessage(message + " Because you are not logged in, it will be deleted in 4 hours.")
	}
	return err
}
