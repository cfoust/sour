package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/servers"

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
	Clients *clients.ClientManager

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
		Clients:       clients.NewClientManager(),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
	}

	return server
}

func (server *Cluster) PollServers(ctx context.Context) {
	forceDisconnects := server.manager.ReceiveDisconnects()
	gamePackets := server.manager.ReceivePackets()

	for {
		select {
		case event := <-forceDisconnects:
			log.Info().Msgf("client forcibly disconnected %d %s", event.Reason, event.Text)

			client := server.Clients.FindClient(uint16(event.Client))

			if client == nil {
				continue
			}

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			client.Disconnect(int(event.Reason), event.Text)
		case packet := <-gamePackets:
			client := server.Clients.FindClient(uint16(packet.Client))

			if client == nil {
				continue
			}

			parseData := packet.Packet.Data
			parsed := game.Packet(parseData)
			msgType, haveType := parsed.GetInt()
			if haveType && msgType != -1 {
				log.Debug().Str("code", game.MessageCode(msgType).String()).Msg("server -> client")
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
	go server.PollServers(ctx)
	for _, serverConfig := range server.settings.Servers {
		gameServer, err := server.manager.NewServer(ctx, serverConfig.Preset)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create server")
		}

		gameServer.Alias = serverConfig.Alias

		go gameServer.Start(ctx)
	}
	go server.manager.PruneServers(ctx)
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

	// A context valid JUST for the lifetime of the client
	clientCtx, cancel := context.WithCancel(ctx)

	logger := log.With().Uint16("client", client.Id()).Logger()

	defer client.Destroy()

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
			cancel()
			return
		case msg := <-toServerTagged:
			data := msg.Data

			packet := game.Packet(data)
			type_, haveType := packet.GetInt()
			command, haveText := packet.GetString()

			passthrough := func() {
				if DEBUG {
					logger.Debug().Str("code", game.MessageCode(type_).String()).Msg("client -> server")
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
				type_ == int32(game.N_TEXT) &&
				haveText &&
				strings.HasPrefix(command, "#") {

				command := command[1:]
				logger.Info().Str("command", command).Msg("intercepted command")

				// Only send this packet after we've checked
				// whether the cluster should handle it
				go func() {
					response, err := server.RunCommandWithTimeout(clientCtx, command, client, state)

					if len(response) == 0 && err == nil {
						passthrough()
						return
					}

					if err != nil {
						clients.SendServerMessage(client, game.Red(err.Error()))
						return
					} else if len(response) > 0 {
						clients.SendServerMessage(client, response)
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
				response, err := server.RunCommandWithTimeout(clientCtx, command, client, state)
				outChannel <- clients.CommandResult{
					Err:      err,
					Response: response,
				}
			}()
		case <-disconnect:
			cancel()
			state.Mutex.Lock()
			if state.Server != nil {
				state.Server.SendDisconnect(client.Id())
				state.Server = nil
			}
			state.Mutex.Unlock()
		}
	}
}

func (server *Cluster) PollClients(ctx context.Context) {
	newClients := server.Clients.ReceiveClients()

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
