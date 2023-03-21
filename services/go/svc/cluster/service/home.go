package service

import (
	"context"
	"fmt"
	"github.com/cfoust/sour/pkg/game"
)

func (c *Cluster) GoHome(ctx context.Context, user *User) error {
	var err error

	logger := user.Logger()

	isLoggedIn := user.IsLoggedIn()
	if !isLoggedIn {
		err := c.runCommandWithTimeout(ctx, user, "go lobby")
		if err != nil {
			return err
		}
		user.Message(game.Red("you must log in to go home"))
		return nil
	}

	home, err := user.GetHomeSpace(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("could not find home space")
		return fmt.Errorf("could not find home space")
	}

	instance, err := c.spaces.StartSpace(ctx, home.UUID)
	if err != nil {
		return err
	}

	_, err = user.ConnectToSpace(instance.Server, home.UUID)

	return err
}
