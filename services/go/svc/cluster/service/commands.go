package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/commands"
	"github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/server/protocol/gamemode"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

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
		numClients := gameServer.NumClients()
		gameServer.Mutex.Unlock()

		if numClients < 2 {
			user.Message(message)
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

type CreateParams struct {
	Map    opt.Option[string]
	Preset opt.Option[string]
	Mode   opt.Option[int]
}

func (server *Cluster) inferCreateParams(args []string) (*CreateParams, error) {
	params := CreateParams{}

	for _, arg := range args {
		mode := constants.GetModeNumber(arg)
		if opt.IsSome(mode) {
			params.Mode = mode
			continue
		}

		map_ := server.servers.Maps.FindMap(arg)
		if map_ != nil {
			params.Map = opt.Some(arg)
			continue
		}

		preset := server.servers.FindPreset(arg, false)
		if opt.IsSome(preset) {
			params.Preset = opt.Some(preset.Value.Name)
			continue
		}

		return nil, fmt.Errorf("argument '%s' neither corresponded to a map nor a game mode", arg)
	}

	return &params, nil
}

func (server *Cluster) CreateGame(ctx context.Context, params *CreateParams, user *User) error {
	logger := user.Logger()
	server.createMutex.Lock()
	defer server.createMutex.Unlock()

	lastCreate, hasLastCreate := server.lastCreate[user.Connection.Host()]
	if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
		return errors.New("too soon since last server create")
	}

	existingServer, hasExistingServer := server.hostServers[user.Connection.Host()]
	if hasExistingServer {
		server.servers.RemoveServer(existingServer)
	}

	logger.Info().Msg("starting server")

	presetName := ""
	if opt.IsSome(params.Preset) {
		presetName = params.Preset.Value
	}

	gameServer, err := server.servers.NewServer(server.serverCtx, presetName, false)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		return errors.New("failed to create server")
	}

	logger = logger.With().Str("server", gameServer.Reference()).Logger()

	mode := int32(params.Mode.Value)
	if opt.IsSome(params.Mode) && !gamemode.Valid(gamemode.ID(mode)) {
		return fmt.Errorf("game mode not yet supported")
	}

	if opt.IsSome(params.Mode) && opt.IsSome(params.Map) {
		gameServer.ChangeMap(mode, params.Map.Value)
	} else if opt.IsSome(params.Mode) {
		gameServer.SetMode(mode)
	} else if opt.IsSome(params.Map) {
		gameServer.SetMap(params.Map.Value)
	}

	server.lastCreate[user.Connection.Host()] = time.Now()
	server.hostServers[user.Connection.Host()] = gameServer

	connected, err := user.ConnectToServer(gameServer, "", false, true)
	go server.GivePrivateMatchHelp(server.serverCtx, user, user.Server)

	go func() {
		ctx, cancel := context.WithTimeout(user.Connection.Session().Ctx(), time.Second*10)
		defer cancel()

		select {
		case status := <-connected:
			if !status {
				return
			}

			user.ServerClient.GrantMaster()
		case <-ctx.Done():
			return
		}
	}()

	return nil
}

func (s *Cluster) runCommand(ctx context.Context, user *User, command string) error {
	contexts := make([]commands.Commandable, 0)

	args := strings.Split(command, " ")
	if len(args) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	// First check cluster commands
	if s.commands.CanHandle(args) {
		return s.commands.Handle(ctx, user, args)
	}

	contexts = append(contexts, s.commands)

	// TODO then do space

	server := user.GetServer()
	if server != nil && server.Commands.CanHandle(args) {
		client := server.Clients.GetClientByID(uint32(user.Id))
		if client != nil {
			return server.Commands.Handle(ctx, client, args)
		}
	}

	if server != nil {
		contexts = append(contexts, server.Commands)
	}

	// Then help
	first := args[0]
	if first != "help" && first != "?" {
		return fmt.Errorf("unrecognized command")
	}

	helpArgs := args[1:]
	if len(helpArgs) == 0 {
		user.RawMessage("available commands: (say '#help [command]' for more information)")
		for _, commandable := range contexts {
			user.RawMessage(commandable.Help())
		}
		return nil
	}

	// Help for a specific command
	for _, commandable := range contexts {
		helpString := commandable.GetHelp(helpArgs)
		if helpString != "" {
			user.RawMessage(helpString)
			return nil
		}
	}

	// Did not match anything
	return fmt.Errorf("could not find help for command")
}

func (s *Cluster) runCommandWithTimeout(ctx context.Context, user *User, command string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)

	resultChannel := make(chan error)
	defer cancel()

	go func() {
		resultChannel <- s.runCommand(ctx, user, command)
	}()

	select {
	case result := <-resultChannel:
		return result
	case <-ctx.Done():
		return fmt.Errorf("command timed out")
	}
}

func (s *Cluster) registerCommands() {
	goCommand := commands.Command{
		Name:        "go",
		Aliases:     []string{"join"},
		ArgFormat:   "[name|id|alias]",
		Description: "move to a space, server, or map by name, id, or alias",
		Callback: func(ctx context.Context, user *User, target string) error {
			if target == "home" {
				return s.runCommandWithTimeout(ctx, user, "home")
			}

			logger := user.Logger()

			user.Mutex.RLock()
			if user.Server != nil && user.Server.IsReference(target) {
				logger.Info().Msg("user already connected to target")
				user.Mutex.RUnlock()
				return fmt.Errorf("you are already there")
			}
			user.Mutex.RUnlock()

			for _, gameServer := range s.servers.Servers {
				if !gameServer.IsReference(target) {
					continue
				}

				_, err := user.Connect(gameServer)
				if err != nil {
					return err
				}

				return nil
			}

			// Look for a space
			space, err := s.spaces.SearchSpace(ctx, target)
			if err != nil {
				return fmt.Errorf("space not found")
			}

			if space == nil {
				return fmt.Errorf("failed to find server or space matching %s", target)
			}

			instance, err := s.spaces.StartSpace(context.Background(), target)
			if err != nil {
				return err
			}

			// Appears in the user's URL bar
			serverName := instance.Space.GetID()

			config, err := space.GetConfig(ctx)
			if err != nil {
				return err
			}

			alias := config.Alias

			if alias != "" {
				serverName = alias
			}

			_, err = user.ConnectToSpace(instance.Server, serverName)
			return err
		},
	}

	createGameCommand := commands.Command{
		Name:        "creategame",
		ArgFormat:   "[coop|ffa|insta|ctf|..etc] [map]",
		Description: "create a private game for you and your friends",
		Callback: func(ctx context.Context, user *User, mode string, map_ string) error {
			params, err := s.inferCreateParams([]string{mode, map_})
			if err != nil {
				return err
			}

			return s.CreateGame(ctx, params, user)
		},
	}

	duelCommand := commands.Command{
		Name:        "duel",
		ArgFormat:   "[ffa|insta]",
		Aliases:     []string{"queue"},
		Description: "queue for 1v1 matchmaking",
		Callback: func(ctx context.Context, user *User, duelType string) error {
			err := s.matches.Queue(user, duelType)
			if err != nil {
				// Theoretically, there might also just not be a default, but whatever.
				return fmt.Errorf("duel type '%s' does not exist", duelType)
			}

			return nil
		},
	}

	stopDuelCommand := commands.Command{
		Name:        "stopduel",
		Description: "unqueue from 1v1 matchmaking",
		Callback: func(ctx context.Context, user *User) {
			s.matches.Dequeue(user)
		},
	}

	homeCommand := commands.Command{
		Name:        "home",
		Description: "go to your home space (also available via #go home)",
		Callback: func(ctx context.Context, user *User) error {
			err := s.GoHome(s.serverCtx, user)
			if err != nil {
				logger := user.Logger()
				logger.Error().Err(err).Msg("could not go home")
				return fmt.Errorf("could not go home")
			}
			return nil
		},
	}

	aliasCommand := commands.Command{
		Name:        "alias",
		ArgFormat:   "[alias]",
		Description: "set the alias for the space",
		Callback: func(ctx context.Context, user *User, alias string) error {
			if !user.IsLoggedIn() {
				return fmt.Errorf("you must be logged in to make an alias for a space")
			}

			isOwner, err := user.IsOwner(ctx)
			if err != nil {
				return err
			}

			if !isOwner {
				return fmt.Errorf("this is not your space")
			}

			instance := user.GetSpace()
			space := instance.Space

			if len(alias) < 7 {
				return fmt.Errorf("alias too short")
			}

			if !verse.IsValidAlias(alias) {
				return fmt.Errorf("aliases must consist of lowercase letters, numbers, or hyphens")
			}

			if len(alias) > 16 {
				return fmt.Errorf("alias too long")
			}

			// Ensure the alias does not match any maps in our asset indices, either
			found := s.assets.FindMap(alias)
			if found != nil {
				return fmt.Errorf("alias taken by a pre-built map")
			}

			err = space.SetAlias(ctx, alias)
			if err != nil {
				return err
			}

			s.AnnounceInServer(ctx, instance.Server, fmt.Sprintf("space alias set to %s", alias))
			return nil

		},
	}

	descCommand := commands.Command{
		Name:        "desc",
		ArgFormat:   "[description]",
		Description: "set the description for the space",
		Callback: func(ctx context.Context, user *User, description string) error {
			isOwner, err := user.IsOwner(ctx)
			if err != nil {
				return err
			}

			if !isOwner {
				return fmt.Errorf("this is not your space")
			}

			instance := user.GetSpace()
			space := instance.Space
			gameServer := instance.Server

			if len(description) == 0 {
				return fmt.Errorf("description too short")
			}

			if len(description) > 32 {
				description = description[:32]
			}

			err = space.SetDescription(ctx, description)
			if err != nil {
				return err
			}

			gameServer.SetDescription(description)
			return nil
		},
	}

	editCommand := commands.Command{
		Name:        "edit",
		ArgFormat:   "[yes|no]",
		Description: "enable or disable open editing on the space.",
		Callback: func(ctx context.Context, user *User, enabled *bool) error {
			isOwner, err := user.IsOwner(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to change edit state")
				return fmt.Errorf("failed to change edit state")
			}

			if !isOwner {
				return fmt.Errorf("this is not your space")
			}

			space := user.GetSpace()
			editing := space.Editing

			current := editing.IsOpenEdit()
			if enabled != nil {
				editing.SetOpenEdit(*enabled)
			} else {
				editing.SetOpenEdit(!current)
			}

			gameServer := space.Server

			canEdit := editing.IsOpenEdit()

			if canEdit {
				s.AnnounceInServer(ctx, gameServer, "editing is now enabled")
			} else {
				s.AnnounceInServer(ctx, gameServer, "editing is now disabled")
			}

			return nil
		},
	}

	err := s.commands.Register(
		goCommand,
		createGameCommand,
		duelCommand,
		stopDuelCommand,
		homeCommand,
		aliasCommand,
		descCommand,
		editCommand,
	)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to register cluster command")
	}
}

func (s *Cluster) HandleCommand(ctx context.Context, user *User, command string) {
	err := s.runCommandWithTimeout(ctx, user, command)
	logger := user.Logger()
	if err != nil {
		logger.Error().Err(err).Msgf("user command failed: %s", command)
		user.Message(game.Red(fmt.Sprintf("command failed: %s", err.Error())))
		return
	}
}
