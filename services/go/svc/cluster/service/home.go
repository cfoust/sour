package service

import (
	"context"
	"time"
	"fmt"

	"github.com/cfoust/sour/svc/cluster/verse"
)

func (c *Cluster) GoHome(ctx context.Context, user *User) error {
	var err error
	var space *verse.UserSpace

	isLoggedIn := user.IsLoggedIn()
	if !isLoggedIn {
		space, err = c.verse.NewSpace(ctx, "")
		if err != nil {
			return err
		}

		err = space.Expire(ctx, time.Hour*4)
		if err != nil {
			return err
		}

		user.TempHomeID = space.GetID()
	} else {
		space, err = user.Verse.GetHomeSpace(ctx)
		if err != nil {
			return fmt.Errorf("could not find home space")
		}
	}

	instance, err := c.spaces.StartSpace(ctx, space.GetID())
	if err != nil {
	    return err
	}

	_, err = user.ConnectToSpace(instance.Deployment.GetServer(), space.GetID())

	return err
}
