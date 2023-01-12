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

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

func (server *Cluster) GivePrivateMatchHelp(ctx context.Context, client *clients.Client, gameServer *servers.GameServer) {
	tick := time.NewTicker(30 * time.Second)

	message := fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server.", gameServer.Id)
	if client.Connection.Type() == clients.ClientTypeWS {
		message = fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server or by sending the link in your URL bar. (We also copied it for you!)", gameServer.Id)
	}

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

func getModeNames() []string {
	return []string{
		"ffa", "coop", "teamplay", "insta", "instateam", "effic", "efficteam", "tac", "tacteam", "capture", "regencapture", "ctf", "instactf", "protect", "instaprotect", "hold", "instahold", "efficctf", "efficprotect", "effichold", "collect", "instacollect", "efficcollect",
	}
}

func getModeNumber(mode string) opt.Option[int] {
	for i, name := range getModeNames() {
		if name == mode {
			return opt.Some(i)
		}
	}

	return opt.None[int]()
}

type CreateParams struct {
	Map    opt.Option[string]
	Preset opt.Option[string]
	Mode   opt.Option[int]
}

func (server *Cluster) inferCreateParams(args []string) (*CreateParams, error) {
	params := CreateParams{}

	for _, arg := range args {
		mode := getModeNumber(arg)
		if opt.IsSome(mode) {
			params.Mode = mode
			continue
		}

		map_ := server.manager.Maps.FindMap(arg)
		if opt.IsSome(map_) {
			params.Map = opt.Some(arg)
			continue
		}

		preset := server.manager.FindPreset(arg, false)
		if opt.IsSome(preset) {
			params.Preset = opt.Some(preset.Value.Name)
			continue
		}

		return nil, fmt.Errorf("argument '%s' neither corresponded to a map nor a game mode", arg)
	}

	return &params, nil
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
		params := &CreateParams{}
		if len(args) > 1 {
			params, err = server.inferCreateParams(args[1:])
			if err != nil {
				return true, "", err
			}
		}

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
		if opt.IsSome(params.Preset) {
			presetName = params.Preset.Value
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

		if opt.IsSome(params.Mode) && opt.IsSome(params.Map) {
			gameServer.SendCommand(fmt.Sprintf("changemap %s %d", params.Map.Value, params.Mode.Value))
		} else if opt.IsSome(params.Mode) {
			gameServer.SendCommand(fmt.Sprintf("setmode %d", params.Mode.Value))
		} else if opt.IsSome(params.Map) {
			gameServer.SendCommand(fmt.Sprintf("setmap %s", params.Map.Value))
		}

		server.lastCreate[client.Connection.Host()] = time.Now()
		server.hostServers[client.Connection.Host()] = gameServer

		connected, err := client.ConnectToServer(gameServer, "", false, true)
		go server.GivePrivateMatchHelp(server.serverCtx, client, client.Server)

		go func() {
			ctx, cancel := context.WithTimeout(client.Connection.SessionContext(), time.Second*10)
			defer cancel()

			select {
			case status := <-connected:
				if !status {
					return
				}

				clientNum := client.GetClientNum()
				gameServer.SendCommand(fmt.Sprintf("grantmaster %d", clientNum))
			case <-ctx.Done():
				log.Info().Msgf("context finished")
				return
			}
		}()

		return true, "", nil

	case "openedit":
		gameServer := client.GetServer()
		instance := server.spaces.FindInstance(gameServer)
		if instance == nil {
			return true, "", fmt.Errorf("you are not in a space")
		}

		user := client.User
		if user == nil || user.Verse == nil {
			return true, "", fmt.Errorf("you are not logged in")
		}

		space := instance.Space
		owner, err := space.GetOwner(ctx)
		if err != nil {
			return true, "", fmt.Errorf("failed to get owner")
		}

		if user.Verse.GetID() == owner {
			editing := instance.Editing
			current := editing.IsOpenEdit()
			editing.SetOpenEdit(!current)

			canEdit := editing.IsOpenEdit()

			if canEdit {
				server.AnnounceInServer(ctx, gameServer, "editing is now enabled")
			} else {
				server.AnnounceInServer(ctx, gameServer, "editing is now disabled")
			}

			return true, "", nil
		}

		return true, "", fmt.Errorf("you are not the owner")

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

			_, err := client.Connect(gameServer)
			if err != nil {
				return true, "", err
			}

			return true, "", nil
		}

		// Look for a space
		space, err := server.spaces.SearchSpace(ctx, target)
		if err != nil {
		    return true, "", err
		}

		if space != nil {
			instance, err := server.spaces.StartSpace(ctx, target)
			if err != nil {
			    return true, "", err
			}
			_, err = client.ConnectToSpace(instance.Server, instance.Space.GetID())
			return true, "", err
		}

		logger.Warn().Msgf("could not find server: %s", target)
		return true, "", fmt.Errorf("failed to find server or space matching %s", target)

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

	case "home":
		server.GoHome(server.serverCtx, client)
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
