package service

import (
	"context"
	"fmt"
	"time"

	//"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/verse"
)

func (c *Cluster) GoHome(ctx context.Context, client *clients.Client) error {
	var err error
	var space *verse.Space
	user := client.GetUser()
	isLoggedIn := user != nil
	if user == nil {
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

	_, err = client.ConnectToServer(instance.Server, false, true)

	message := fmt.Sprintf(
		"Welcome to your home space (%s).",
		space.GetID(),
	)

	if isLoggedIn {
		client.SendServerMessage(message)
	} else {
		client.SendServerMessage(message + " Because you are not logged in, it will be deleted in 4 hours.")
	}
	return err
}
