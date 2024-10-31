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
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/gameserver/protocol/gamemode"
	"github.com/cfoust/sour/pkg/server/servers"

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
		helpString, err := commandable.GetHelp(helpArgs)
		if err == nil {
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
			for _, gameServer := range s.servers.Servers {
				if !gameServer.IsReference(target) {
					continue
				}

				_, err := user.Connect(gameServer)
				return err
			}

			return fmt.Errorf("could not find server '%s'", target)
		},
	}

	createGameCommand := commands.Command{
		Name:        "creategame",
		ArgFormat:   "[coop|ffa|insta|ctf|..etc] [map]",
		Description: "create a private game for you and your friends",
		Callback: func(ctx context.Context, user *User, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("you must provide at least one argument")
			}

			params, err := s.inferCreateParams(args)
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

	err := s.commands.Register(
		goCommand,
		createGameCommand,
		duelCommand,
		stopDuelCommand,
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
