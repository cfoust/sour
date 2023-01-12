package service

import (
	"context"
	"fmt"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/rs/zerolog/log"
)

func (server *Cluster) GoHome(ctx context.Context, client *clients.Client) error {
	gameServer, err := server.manager.NewServer(ctx, "", true)
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

	// Load the user's world or create a new one
	editing := servers.NewEditingState()
	gameServer.Editing = editing

	map_ := maps.NewMap()
	err = editing.LoadMap(map_)
	if err != nil {
		return err
	}

	gz, err := map_.EncodeOGZ()
	if err != nil {
		return err
	}

	go editing.PollEdits(ctx)

	connected, err := client.ConnectToServer(gameServer, false, true)
	result := <-connected
	if result == false || err != nil {
		return fmt.Errorf("client never joined")
	}

	p := game.Packet{}
	p.Put(game.N_SENDMAP)
	p = append(p, gz...)

	client.Send(game.GamePacket{
		Channel: 2,
		Data:    p,
	})

	return nil
}
