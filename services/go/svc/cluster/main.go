package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/messages"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Client struct {
	id   uint16
	host string

	server     *servers.GameServer
	sendPacket chan game.GamePacket
	closeSlow  func()
}

const (
	CREATE_SERVER_COOLDOWN = time.Duration(10 * time.Second)
	DEBUG                  = false
)

type Cluster struct {
	clients *clients.ClientManager

	createMutex sync.Mutex
	// host -> time a client from that host last created a server. We
	// REALLY don't want clients to be able to DDOS us
	lastCreate map[string]time.Time
	// host -> the server created by that host
	// each host can only have one server at once
	hostServers map[string]*servers.GameServer

	settings      config.ClusterSettings
	manager       *servers.ServerManager
	serverCtx     context.Context
	serverMessage chan []byte
}

func NewCluster(ctx context.Context, serverManager *servers.ServerManager, settings config.ClusterSettings) *Cluster {
	server := &Cluster{
		serverCtx:     ctx,
		settings:      settings,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		clients:       clients.NewClientManager(),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
	}

	return server
}

func (server *Cluster) PollServer(ctx context.Context, gameServer *servers.GameServer) {
	forceDisconnects := gameServer.ReceiveDisconnects()
	gamePackets := gameServer.ReceivePackets()

	for {
		select {
		case event := <-forceDisconnects:
			log.Info().Msgf("client forcibly disconnected %d %s", event.Reason, event.Text)

			client := server.clients.FindClient(uint16(event.Client))

			if client == nil {
				continue
			}

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			client.Disconnect(int(event.Reason), event.Text)
		case packet := <-gamePackets:
			client := server.clients.FindClient(uint16(packet.Client))

			if client == nil {
				continue
			}

			parseData := packet.Packet.Data
			parsed := game.Packet(parseData)
			msgType, haveType := parsed.GetInt()
			if haveType && msgType != -1 {
				log.Debug().Str("code", messages.MessageCode(msgType).String()).Msg("server -> client")
			}

			gamePacket := game.GamePacket{
				Channel: uint8(packet.Packet.Channel),
				Data:    packet.Packet.Data,
			}

			client.Send(gamePacket)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) StartServers(ctx context.Context) {
	for _, serverConfig := range server.settings.Servers {
		gameServer, err := server.manager.NewServer(ctx, serverConfig.Preset)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create server")
		}

		gameServer.Alias = serverConfig.Alias

		go gameServer.Start(ctx)
		go server.PollServer(ctx, gameServer)
	}
	go server.manager.PruneServers(ctx)
}

func (server *Cluster) SendServerMessage(client clients.Client, message string) {
	packet := game.Packet{}
	packet.PutInt(int32(messages.N_SERVMSG))
	message = fmt.Sprintf("%s %s", game.Yellow("sour"), message)
	packet.PutString(message)
	client.Send(game.GamePacket{
		Channel: 1,
		Data:    packet,
	})
}

func (server *Cluster) GivePrivateMatchHelp(ctx context.Context, client clients.Client, gameServer *servers.GameServer) {
	// TODO this is broken; the context is from the timeout for the command so it never runs again
	tick := time.NewTicker(30 * time.Second)

	message := fmt.Sprintf("This is your private server. Have other players join by saying '#join %s' in any Sour server.", gameServer.Id)

	for {
		gameServer.Mutex.Lock()
		clients := gameServer.NumClients
		gameServer.Mutex.Unlock()

		log.Info().Msgf("warning: %d", clients)

		if clients < 2 {
			server.SendServerMessage(client, message)
		} else {
			return
		}

		select {
		case <-tick.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) RunCommand(ctx context.Context, command string, client clients.Client, state *clients.ClientState) (string, error) {
	logger := log.With().Uint16("client", client.Id()).Str("command", command).Logger()
	logger.Info().Msg("running command")

	args := strings.Split(command, " ")

	if len(args) == 0 {
		return "", errors.New("invalid command")
	}

	switch args[0] {
	case "creategame":
		server.createMutex.Lock()
		defer server.createMutex.Unlock()

		lastCreate, hasLastCreate := server.lastCreate[client.Host()]
		if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
			return "", errors.New("too soon since last server create")
		}

		existingServer, hasExistingServer := server.hostServers[client.Host()]
		if hasExistingServer {
			server.manager.RemoveServer(existingServer)
		}

		logger.Info().Msg("starting server")

		presetName := ""
		if len(args) > 1 {
			presetName = args[1]
		}

		gameServer, err := server.manager.NewServer(server.serverCtx, presetName)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create server")
			return "", errors.New("failed to create server")
		}

		logger = logger.With().Str("server", gameServer.Reference()).Logger()

		go gameServer.Start(server.serverCtx)
		go server.PollServer(server.serverCtx, gameServer)

		tick := time.NewTicker(250 * time.Millisecond)
		for {
			status := gameServer.GetStatus()
			if status == servers.ServerOK {
				logger.Info().Msg("server ok")
				break
			}

			select {
			case <-ctx.Done():
				return "", errors.New("server start timed out")
			case <-tick.C:
				continue
			}
		}

		server.lastCreate[client.Host()] = time.Now()
		server.hostServers[client.Host()] = gameServer

		state.Mutex.Lock()

		// Automatically connect clients to their servers
		if client.Type() == clients.ClientTypeWS && state.Server == nil {
			state.Mutex.Unlock()
			return gameServer.Id, nil
		}

		if client.Type() == clients.ClientTypeENet {
			go server.GivePrivateMatchHelp(ctx, client, state.Server)
		}

		state.Mutex.Unlock()
		return server.RunCommand(ctx, fmt.Sprintf("join %s", gameServer.Id), client, state)

	case "join":
		if len(args) != 2 {
			return "", errors.New("join takes a single argument")
		}

		target := args[1]

		state.Mutex.Lock()
		defer state.Mutex.Unlock()

		if state.Server != nil && state.Server.IsReference(target) {
			logger.Info().Msg("client already connected to target")
			break
		}

		for _, gameServer := range server.manager.Servers {
			if !gameServer.IsReference(target) || gameServer.Status != servers.ServerOK {
				continue
			}

			if state.Server != nil {
				state.Server.SendDisconnect(client.Id())
			}

			state.Server = gameServer

			logger.Info().Str("server", gameServer.Reference()).
				Msg("client connecting to server")

			gameServer.SendConnect(client.Id())

			client.Connect()
			return "", nil
		}

		logger.Warn().Msgf("could not find server: %s", target)
	}

	return "", nil
}

func (server *Cluster) RunCommandWithTimeout(ctx context.Context, command string, client clients.Client, state *clients.ClientState) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)

	resultChannel := make(chan clients.CommandResult)

	go func() {
		response, err := server.RunCommand(ctx, command, client, state)
		resultChannel <- clients.CommandResult{
			Err:      err,
			Response: response,
		}
	}()

	select {
	case result := <-resultChannel:
		cancel()
		return result.Response, result.Err
	case <-ctx.Done():
		cancel()
		return "", errors.New("command timed out")
	}
}

type DestPacket struct {
	Data    []byte
	Channel uint8
	Dest    *servers.GameServer
}

func (server *Cluster) PollClient(ctx context.Context, client clients.Client, state *clients.ClientState) {
	toServer := client.ReceivePackets()
	commands := client.ReceiveCommands()
	disconnect := client.ReceiveDisconnect()

	logger := log.With().Uint16("client", client.Id()).Logger()

	// Tag messages with the server that the client was connected to
	toServerTagged := make(chan DestPacket, clients.CLIENT_MESSAGE_LIMIT)
	go func() {
		for {
			select {
			case packet := <-toServer:
				state.Mutex.Lock()
				tagged := DestPacket{
					Data:    packet.Data,
					Channel: packet.Channel,
					Dest:    state.Server,
				}
				state.Mutex.Unlock()

				toServerTagged <- tagged
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-toServerTagged:
			data := msg.Data

			packet := game.Packet(data)
			type_, haveType := packet.GetInt()
			command, haveText := packet.GetString()

			passthrough := func() {
				if DEBUG {
					logger.Debug().Str("code", messages.MessageCode(type_).String()).Msg("client -> server")
				}
				state.Mutex.Lock()
				if state.Server != nil && state.Server == msg.Dest {
					state.Server.SendData(client.Id(), uint32(msg.Channel), msg.Data)
				}
				state.Mutex.Unlock()
			}

			// Intercept commands and run them first
			if msg.Channel == 1 &&
				haveType &&
				type_ == int32(messages.N_TEXT) &&
				haveText &&
				strings.HasPrefix(command, "#") {

				command := command[1:]
				logger.Info().Str("command", command).Msg("intercepted command")

				// Only send this packet after we've checked
				// whether the cluster should handle it
				go func() {
					response, err := server.RunCommandWithTimeout(ctx, command, client, state)

					if len(response) == 0 && err == nil {
						passthrough()
						return
					}

					if err != nil {
						server.SendServerMessage(client, game.Red(err.Error()))
						return
					} else if len(response) > 0 {
						server.SendServerMessage(client, response)
						return
					}

					if command == "help" {
						passthrough()
					}
				}()
				continue
			}

			passthrough()

		case request := <-commands:
			command := request.Command
			outChannel := request.Response

			go func() {
				response, err := server.RunCommandWithTimeout(ctx, command, client, state)
				outChannel <- clients.CommandResult{
					Err:      err,
					Response: response,
				}
			}()
		case <-disconnect:
			state.Mutex.Lock()
			if state.Server != nil {
				state.Server.SendDisconnect(client.Id())
				state.Server = nil
			}
			state.Mutex.Unlock()
		}
	}
}

// When a new client is created, go
func (server *Cluster) PollClients(ctx context.Context) {
	newClients := server.clients.ReceiveClients()

	for {
		select {
		case client := <-newClients:
			go server.PollClient(ctx, client.Client, client.State)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if DEBUG {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	sourConfig, err := config.GetSourConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sour configuration, please specify one with the SOUR_CONFIG environment variable")
	}

	clusterConfig := sourConfig.Cluster

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maps := assets.NewMapFetcher()
	err = maps.FetchIndices(clusterConfig.Assets)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to load assets")
	}

	serverManager := servers.NewServerManager(maps, clusterConfig.ServerDescription, clusterConfig.Presets)
	cluster := NewCluster(ctx, serverManager, clusterConfig)

	err = cluster.manager.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server manager")
	}

	wsIngress := ingress.NewWSIngress(cluster.clients)

	enet := make([]*ingress.ENetIngress, 0)
	for _, enetConfig := range clusterConfig.Ingress.Desktop {
		enetIngress := ingress.NewENetIngress(cluster.clients)
		enetIngress.Serve(enetConfig.Port)
		enetIngress.InitialCommand = enetConfig.Command
		go enetIngress.Poll(ctx)
		enet = append(enet, enetIngress)
	}

	go cluster.StartServers(ctx)
	go cluster.PollClients(ctx)

	errc := make(chan error, 1)
	go func() {
		errc <- wsIngress.Serve(ctx, clusterConfig.Ingress.Web.Port)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, os.Kill)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	wsIngress.Shutdown(ctx)
	for _, enetIngress := range enet {
		enetIngress.Shutdown()
	}
	cluster.Shutdown()
}
