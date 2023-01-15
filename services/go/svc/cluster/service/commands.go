package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/verse"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

func (server *Cluster) GivePrivateMatchHelp(ctx context.Context, user *User, gameServer *servers.GameServer) {
	tick := time.NewTicker(30 * time.Second)

	message := fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server.", gameServer.Id)

	if user.Connection.Type() == ingress.ClientTypeWS {
		message = fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server or by sending the link in your URL bar. (We also copied it for you!)", gameServer.Id)
	}

	sessionContext := user.ServerSessionContext()

	for {
		gameServer.Mutex.Lock()
		numClients := gameServer.NumClients
		gameServer.Mutex.Unlock()

		if numClients < 2 {
			user.SendServerMessage(message)
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

func (server *Cluster) RunCommand(ctx context.Context, command string, user *User) (handled bool, response string, err error) {
	logger := user.Logger().With().Str("command", command).Logger()
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

		lastCreate, hasLastCreate := server.lastCreate[user.Connection.Host()]
		if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
			return true, "", errors.New("too soon since last server create")
		}

		existingServer, hasExistingServer := server.hostServers[user.Connection.Host()]
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

		server.lastCreate[user.Connection.Host()] = time.Now()
		server.hostServers[user.Connection.Host()] = gameServer

		connected, err := user.ConnectToServer(gameServer, "", false, true)
		go server.GivePrivateMatchHelp(server.serverCtx, user, user.Server)

		go func() {
			ctx, cancel := context.WithTimeout(user.Connection.SessionContext(), time.Second*10)
			defer cancel()

			select {
			case status := <-connected:
				if !status {
					return
				}

				gameServer.SendCommand(fmt.Sprintf("grantmaster %d", user.GetClientNum()))
			case <-ctx.Done():
				log.Info().Msgf("context finished")
				return
			}
		}()

		return true, "", nil

	case "alias":
		if !user.IsLoggedIn() {
			return true, "", fmt.Errorf("you must be logged in to make an alias for a space")
		}

		isOwner, err := user.IsOwner(ctx)
		if err != nil {
			return true, "", err
		}

		if !isOwner {
			return true, "", fmt.Errorf("this is not your space")
		}

		instance := user.GetSpace()
		space := instance.Space

		if len(command) < 7 {
			return true, "", fmt.Errorf("alias too short")
		}

		alias := command[6:]
		if !verse.IsValidAlias(alias) {
			return true, "", fmt.Errorf("aliases must consist of lowercase letters, numbers, or hyphens")
		}

		if len(alias) > 16 {
			return true, "", fmt.Errorf("alias too long")
		}

		// Ensure the alias does not match any maps in our asset indices, either
		found := server.assets.FindMap(alias)
		if opt.IsSome(found) {
			return true, "", fmt.Errorf("alias taken by a pre-built map")
		}

		err = space.SetAlias(ctx, alias)
		if err != nil {
			return true, "", err
		}

		server.AnnounceInServer(ctx, instance.Server, fmt.Sprintf("space alias set to %s", alias))
		return true, "", nil

	case "desc":
		isOwner, err := user.IsOwner(ctx)
		if err != nil {
			return true, "", err
		}

		if !isOwner {
			return true, "", fmt.Errorf("this is not your space")
		}

		instance := user.GetSpace()
		space := instance.Space
		server := instance.Server

		if len(command) < 6 {
			return true, "", fmt.Errorf("description too short")
		}

		description := command[5:]
		if len(description) > 32 {
			description = description[:32]
		}

		err = space.SetDescription(ctx, description)
		if err != nil {
			return true, "", err
		}

		server.SendCommand(fmt.Sprintf("serverdesc \"%s\"", description))
		server.SendCommand("refreshserverinfo")
		return true, "", nil


	case "edit":
		isOwner, err := user.IsOwner(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to change edit state")
			return true, "", err
		}

		if !isOwner {
			return true, "", fmt.Errorf("this is not your space")
		}

		space := user.GetSpace()
		editing := space.Editing
		current := editing.IsOpenEdit()
		editing.SetOpenEdit(!current)

		canEdit := editing.IsOpenEdit()

		if canEdit {
			server.AnnounceInServer(ctx, space.Server, "editing is now enabled")
		} else {
			server.AnnounceInServer(ctx, space.Server, "editing is now disabled")
		}

		return true, "", nil

	case "go":
		fallthrough
	case "join":
		if len(args) != 2 {
			return true, "", errors.New("join takes a single argument")
		}

		target := args[1]

		if target == "home" {
			return server.RunCommandWithTimeout(ctx, "home", user)
		}

		user.Mutex.RLock()
		if user.Server != nil && user.Server.IsReference(target) {
			logger.Info().Msg("user already connected to target")
			user.Mutex.RUnlock()
			break
		}
		user.Mutex.RUnlock()

		for _, gameServer := range server.manager.Servers {
			if !gameServer.IsReference(target) || !gameServer.IsRunning() {
				continue
			}

			_, err := user.Connect(gameServer)
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

			// Appears in the user's URL bar
			serverName := instance.Space.GetID()

			alias, err := space.GetAlias(ctx)
			if err != nil {
				return true, "", err
			}

			if alias != "" {
				serverName = alias
			}

			_, err = user.ConnectToSpace(instance.Server, serverName)
			return true, "", err
		}

		logger.Warn().Msgf("could not find server: %s", target)
		return true, "", fmt.Errorf("failed to find server or space matching %s", target)

	case "duel":
		duelType := ""
		if len(args) > 1 {
			duelType = args[1]
		}

		err := server.matches.Queue(user, duelType)
		if err != nil {
			// Theoretically, there might also just not be a default, but whatever.
			return true, "", fmt.Errorf("duel type '%s' does not exist", duelType)
		}

		return true, "", nil

	case "stopduel":
		server.matches.Dequeue(user)
		return true, "", nil

	case "home":
		err := server.GoHome(server.serverCtx, user)
		return true, "", err

	case "help":
		messages := []string{
			fmt.Sprintf("%s: create a private game", game.Blue("#creategame")),
			fmt.Sprintf("%s: join a Sour game server by room code", game.Blue("#join [code]")),
			fmt.Sprintf("%s: queue for a duel", game.Blue("#duel")),
			fmt.Sprintf("%s: leave the duel queue", game.Blue("#stopduel")),
		}

		for _, message := range messages {
			user.SendServerMessage(message)
		}

		return true, "", nil
	}

	return false, "", nil
}

func (server *Cluster) RunCommandWithTimeout(ctx context.Context, command string, user *User) (handled bool, response string, err error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)

	resultChannel := make(chan ingress.CommandResult)

	defer cancel()

	go func() {
		handled, response, err := server.RunCommand(ctx, command, user)
		resultChannel <- ingress.CommandResult{
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
