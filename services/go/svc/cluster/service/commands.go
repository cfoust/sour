package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/rs/zerolog/log"
)

func (server *Cluster) GivePrivateMatchHelp(ctx context.Context, client *clients.Client, gameServer *servers.GameServer) {
	// TODO this is broken; the context is from the timeout for the command so it never runs again
	tick := time.NewTicker(30 * time.Second)

	message := fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server.", gameServer.Id)

	sessionContext := client.ServerSessionContext()

	for {
		gameServer.Mutex.Lock()
		numClients := gameServer.NumClients
		gameServer.Mutex.Unlock()

		if numClients < 2 {
			client.SendServerMessage(message)
		} else {
			return
		}

		select {
		case <-sessionContext.Done():
			return
		case <-tick.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) RunCommand(ctx context.Context, command string, client *clients.Client) (handled bool, response string, err error) {
	logger := log.With().Uint16("client", client.Id).Str("command", command).Logger()
	logger.Info().Msg("running command")

	args := strings.Split(command, " ")

	if len(args) == 0 {
		return false, "", errors.New("invalid command")
	}

	switch args[0] {
	case "creategame":
		server.createMutex.Lock()
		defer server.createMutex.Unlock()

		lastCreate, hasLastCreate := server.lastCreate[client.Connection.Host()]
		if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
			return true, "", errors.New("too soon since last server create")
		}

		existingServer, hasExistingServer := server.hostServers[client.Connection.Host()]
		if hasExistingServer {
			server.manager.RemoveServer(existingServer)
		}

		logger.Info().Msg("starting server")

		presetName := ""
		if len(args) > 1 {
			presetName = args[1]
		}

		gameServer, err := server.manager.NewServer(server.serverCtx, presetName, false)
		if err != nil {
			logger.Error().Err(err).Msg("failed to create server")
			return true, "", errors.New("failed to create server")
		}

		logger = logger.With().Str("server", gameServer.Reference()).Logger()

		err = gameServer.StartAndWait(server.serverCtx)
		if err != nil {
			logger.Error().Err(err).Msg("server failed to start")
			return true, "", errors.New("server failed to start")
		}

		server.lastCreate[client.Connection.Host()] = time.Now()
		server.hostServers[client.Connection.Host()] = gameServer

		client.Mutex.Lock()
		if client.Connection.Type() == clients.ClientTypeWS && client.Server == nil {
			client.Mutex.Unlock()
			return true, gameServer.Id, nil
		}

		if client.Connection.Type() == clients.ClientTypeENet {
			go server.GivePrivateMatchHelp(server.serverCtx, client, client.Server)
		}
		client.Mutex.Unlock()

		return server.RunCommand(ctx, fmt.Sprintf("join %s", gameServer.Id), client)

	case "join":
		if len(args) != 2 {
			return true, "", errors.New("join takes a single argument")
		}

		target := args[1]

		client.Mutex.Lock()
		if client.Server != nil && client.Server.IsReference(target) {
			logger.Info().Msg("client already connected to target")
			client.Mutex.Unlock()
			break
		}
		client.Mutex.Unlock()

		for _, gameServer := range server.manager.Servers {
			if !gameServer.IsReference(target) || !gameServer.IsRunning() {
				continue
			}

			_, err := client.ConnectToServer(gameServer)
			if err != nil {
				return true, "", err
			}

			return true, "", nil
		}

		logger.Warn().Msgf("could not find server: %s", target)
		return true, "", fmt.Errorf("failed to find server %s", target)

	case "duel":
		duelType := ""
		if len(args) > 1 {
			duelType = args[1]
		}

		err := server.matches.Queue(client, duelType)
		if err != nil {
			// Theoretically, there might also just not be a default, but whatever.
			return true, "", fmt.Errorf("duel type '%s' does not exist", duelType)
		}

		return true, "", nil

	case "stopduel":
		server.matches.Dequeue(client)
		return true, "", nil

	case "help":
		messages := []string{
			fmt.Sprintf("%s: create a private game", game.Blue("#creategame")),
			fmt.Sprintf("%s: join a Sour game server by room code", game.Blue("#join [code]")),
			fmt.Sprintf("%s: queue for a duel", game.Blue("#duel")),
			fmt.Sprintf("%s: leave the duel queue", game.Blue("#stopduel")),
		}

		for _, message := range messages {
			client.SendServerMessage(message)
		}

		return true, "", nil
	}

	return false, "", nil
}

func (server *Cluster) RunCommandWithTimeout(ctx context.Context, command string, client *clients.Client) (handled bool, response string, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)

	resultChannel := make(chan clients.CommandResult)

	defer cancel()

	go func() {
		handled, response, err := server.RunCommand(ctx, command, client)
		resultChannel <- clients.CommandResult{
			Handled:  handled,
			Err:      err,
			Response: response,
		}
	}()

	select {
	case result := <-resultChannel:
		return result.Handled, result.Response, result.Err
	case <-ctx.Done():
		cancel()
		return false, "", errors.New("command timed out")
	}

}
