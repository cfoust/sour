package service

import (
	"context"
	"fmt"

	"github.com/cfoust/sour/pkg/game"
	//"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

	"github.com/rs/zerolog/log"
)

func (c *Cluster) GoHome(ctx context.Context, client *clients.Client) error {
	gameServer, err := c.manager.NewServer(ctx, "", true)
	if err != nil {
		log.Error().Err(err).Msg("failed to create home server")
		return err
	}

	err = gameServer.StartAndWait(ctx)
	if err != nil {
		return err
	}

	gameServer.SendCommand(fmt.Sprintf("serverdesc \"Sour %s\"", game.Blue("Home")))
	gameServer.SendCommand("publicserver 1")

	// New empty map
	gameServer.SendCommand("emptymap")

	var space *verse.Space
	user := client.GetUser()
	if user == nil {
		space, err = c.verse.NewSpace(ctx, "")
		if err != nil {
		    return err
		}
	} else {
		space, err = user.Verse.GetHomeSpace(ctx)
		if err != nil {
		    return err
		}
	}

	verseMap, err := space.GetMap(ctx)
	if err != nil {
		return err
	}

	map_, err := verseMap.LoadGameMap(ctx)
	if err != nil {
		return err
	}

	// Load the user's world or create a new one
	editing := servers.NewEditingState(c.verse, space, verseMap)
	gameServer.Editing = editing

	err = editing.LoadMap(map_)
	if err != nil {
		return err
	}

	go editing.PollEdits(ctx)

	_, err = client.ConnectToServer(gameServer, false, true)
	return err
}
