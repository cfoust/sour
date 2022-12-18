package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/messages"
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
	matches       *Matchmaker
	serverCtx     context.Context
	serverMessage chan []byte
}

func NewCluster(ctx context.Context, serverManager *servers.ServerManager, settings config.ClusterSettings) *Cluster {
	clients := clients.NewClientManager()
	server := &Cluster{
		serverCtx:     ctx,
		settings:      settings,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		Clients:       clients,
		matches:       NewMatchmaker(serverManager, clients),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
	}

	return server
}

func (server *Cluster) PollServers(ctx context.Context) {
	connects := server.manager.ReceiveConnects()
	forceDisconnects := server.manager.ReceiveDisconnects()
	gamePackets := server.manager.ReceivePackets()

	for {
		select {
		case id := <-connects:
			client := server.Clients.FindClient(uint16(id))

			if client == nil {
				continue
			}

			state := server.Clients.GetState(client)
			if state == nil {
				continue
			}

			state.Mutex.Lock()
			log.Info().
				Uint16("client", client.Id()).
				Str("server", state.Server.Reference()).
				Msg("connected to server")
			state.Status = clients.ClientStatusConnected
			state.Mutex.Unlock()

		case event := <-forceDisconnects:
			log.Info().Msgf("client forcibly disconnected %d %s", event.Reason, event.Text)

			client := server.Clients.FindClient(uint16(event.Client))

			if client == nil {
				continue
			}

			server.Clients.ClientDisconnected(client)

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			client.Disconnect(int(event.Reason), event.Text)
		case packet := <-gamePackets:
			client := server.Clients.FindClient(uint16(packet.Client))

			if client == nil {
				continue
			}

			state := server.Clients.GetState(client)
			if state == nil {
				continue
			}

			parseData := packet.Packet.Data
			messages, err := messages.Read(parseData)
			if err != nil {
				log.Error().
					Err(err).
					Uint16("client", client.Id()).
					Msg("cluster -> client (failed to decode message)")

				// Forward it anyway
				client.Send(game.GamePacket{
					Channel: uint8(packet.Packet.Channel),
					Data:    packet.Packet.Data,
				})
				continue
			}

			channel := uint8(packet.Packet.Channel)

			// As opposed to client -> server, we don't actually need to do any filtering
			// so we won't repackage the messages individually
			for _, message := range messages {
				log.Debug().
					Str("type", message.Type().String()).
					Uint16("client", client.Id()).
					Msg("cluster -> client")

				client.Send(game.GamePacket{
					Channel: channel,
					Data:    message.Data(),
				})
			}

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
	go server.matches.Poll(ctx)
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

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case msg := <-toServer:
			data := msg.Data

			gameMessages, err := messages.Read(data)
			if err != nil {
				log.Error().
					Err(err).
					Uint16("client", client.Id()).
					Msg("client -> server (failed to decode message)")

				// Forward it anyway
				state.Mutex.Lock()
				if state.Server != nil {
					state.Server.SendData(client.Id(), uint32(msg.Channel), msg.Data)
				}
				state.Mutex.Unlock()
				continue
			}

			passthrough := func(message messages.Message) {
				state.Mutex.Lock()
				if state.Server != nil {
					state.Server.SendData(client.Id(), uint32(msg.Channel), message.Data())
				}
				state.Mutex.Unlock()
			}

			for _, message := range gameMessages {
				if message.Type() == game.N_TEXT {
					text := message.Contents().(*messages.Text).Text

					if strings.HasPrefix(text, "#") {
						command := text[1:]
						logger.Info().Str("command", command).Msg("intercepted command")

						// Only send this packet after we've checked
						// whether the cluster should handle it
						go func() {
							handled, response, err := server.RunCommandWithTimeout(clientCtx, command, client, state)

							if !handled {
								passthrough(message)
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
								passthrough(message)
							}
						}()
					}
				}

				// Skip messages that aren't allowed while the
				// client is connecting, otherwise the server
				// (rightfully) disconnects us. This solves a
				// race condition when switching servers.
				state.Mutex.Lock()
				status := state.Status
				if status == clients.ClientStatusConnecting && !game.IsConnectingMessage(message.Type()) {
					state.Mutex.Unlock()
					continue
				}
				state.Mutex.Unlock()

				logger.Debug().Str("code", message.Type().String()).Msg("client -> server")

				state.Mutex.Lock()
				if state.Server != nil {
					state.Server.SendData(client.Id(), uint32(msg.Channel), message.Data())
				}
				state.Mutex.Unlock()
			}

		case request := <-commands:
			command := request.Command
			outChannel := request.Response

			go func() {
				handled, response, err := server.RunCommandWithTimeout(clientCtx, command, client, state)
				outChannel <- clients.CommandResult{
					Handled:  handled,
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
