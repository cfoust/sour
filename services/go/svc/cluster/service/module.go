package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/mmr"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/go-redis/redis/v9"
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

	startTime     time.Time
	authDomain    string
	settings      config.ClusterSettings
	auth          *auth.DiscordService
	manager       *servers.ServerManager
	matches       *Matchmaker
	serverCtx     context.Context
	serverMessage chan []byte
}

func NewCluster(
	ctx context.Context,
	serverManager *servers.ServerManager,
	settings config.ClusterSettings,
	authDomain string,
	auth *auth.DiscordService,
	redis *redis.Client,
) *Cluster {
	clients := clients.NewClientManager(redis, settings.Matchmaking.Duel)
	server := &Cluster{
		serverCtx:     ctx,
		settings:      settings,
		authDomain:    authDomain,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		Clients:       clients,
		matches:       NewMatchmaker(serverManager, clients, settings.Matchmaking.Duel),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
		startTime:     time.Now(),
		auth:          auth,
	}

	return server
}

func (server *Cluster) GetServerInfo() *servers.ServerInfo {
	info := server.manager.GetServerInfo()

	settings := server.settings.ServerInfo

	info.TimeLeft = int32(settings.TimeLeft)
	info.MaxClients = 999
	info.GameSpeed = int32(settings.GameSpeed)
	info.Map = settings.Map
	info.Description = settings.Description

	return info
}

// We need client information, so this is not on the ServerManager like GetServerInfo is
func (server *Cluster) GetClientInfo() []*servers.ClientExtInfo {
	info := make([]*servers.ClientExtInfo, 0)

	server.Clients.Mutex.Lock()
	server.manager.Mutex.Lock()

	for _, gameServer := range server.manager.Servers {
		clients := gameServer.GetClientInfo()
		for _, client := range clients {
			newClient := *client

			// Replace with clientID
			for client, _ := range server.Clients.State {
				if client.GetServer() == gameServer && int(client.GetClientNum()) == newClient.Client {
					newClient.Client = int(client.Id)
				}
			}

			info = append(info, &newClient)
		}
	}

	server.manager.Mutex.Unlock()
	server.Clients.Mutex.Unlock()

	return info
}

func (server *Cluster) GetUptime() int {
	return int(time.Now().Sub(server.startTime).Round(time.Second) / time.Second)
}

func (server *Cluster) PollDuels(ctx context.Context) {
	queues := server.matches.ReceiveQueues()
	results := server.matches.ReceiveResults()

	for {
		select {
		case result := <-results:
			winner := result.Winner
			loser := result.Loser

			winnerELO, _ := winner.ELO.Ratings[result.Type]
			loserELO, _ := loser.ELO.Ratings[result.Type]

			calc := mmr.NewElo()
			var score float64 = 1 // winner wins
			if result.IsDraw {
				score = 0.5
			}

			winnerOutcome, loserOutcome := calc.Outcome(
				winnerELO.Rating,
				loserELO.Rating,
				score,
			)

			winnerELO.Rating = winnerOutcome.Rating
			loserELO.Rating = loserOutcome.Rating

			if result.IsDraw {
				winnerELO.Draws++
				loserELO.Draws++
			} else {
				winnerELO.Wins++
				loserELO.Losses++
			}

			winner.SaveELOState(ctx)
			loser.SaveELOState(ctx)

			if result.IsDraw {
				message := "the duel ended in a draw, your rating is unchanged"
				winner.SendServerMessage(message)
				loser.SendServerMessage(message)
				continue
			}

			winner.SendServerMessage(
				game.Green("you won! ") + winnerOutcome.String(),
			)
			loser.SendServerMessage(
				game.Red("you lost! ") + loserOutcome.String(),
			)

			message := fmt.Sprintf(
				"%s (%s) beat %s (%s) in %s",
				winner.Name,
				winnerOutcome.String(),
				loser.Name,
				loserOutcome.String(),
				result.Type,
			)

			if result.Disconnected {
				message += " because they disconnected"
			}

			server.Clients.Mutex.Lock()
			for client, _ := range server.Clients.State {
				if client == winner || client == loser {
					continue
				}
				client.SendServerMessage(message)
			}
			server.Clients.Mutex.Unlock()
		case queue := <-queues:
			server.Clients.Mutex.Lock()
			for client, _ := range server.Clients.State {
				if client == queue.Client {
					continue
				}

				client.SendServerMessage(fmt.Sprintf(
					"%s queued for %s",
					client.Reference(),
					queue.Type,
				))
			}
			server.Clients.Mutex.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) PollServers(ctx context.Context) {
	connects := server.manager.ReceiveConnects()
	forceDisconnects := server.manager.ReceiveDisconnects()
	gamePackets := server.manager.ReceivePackets()
	names := server.manager.ReceiveNames()

	for {
		select {
		case join := <-connects:
			client := server.Clients.FindClient(uint16(join.Client))

			if client == nil {
				continue
			}

			client.Mutex.Lock()
			if client.Server != nil {
				log.Info().
					Uint16("client", client.Id).
					Int32("clientNum", join.ClientNum).
					Str("server", client.Server.Reference()).
					Msg("connected to server")
				client.Status = clients.ClientStatusConnected
				client.ClientNum = join.ClientNum
			}
			client.Mutex.Unlock()

		case event := <-names:
			client := server.Clients.FindClient(uint16(event.Client))

			if client == nil {
				continue
			}

			client.Mutex.Lock()
			client.Name = event.Name
			log.Info().
				Uint16("client", client.Id).
				Str("name", client.Name).
				Msg("client has new name")
			client.Mutex.Unlock()

		case event := <-forceDisconnects:
			log.Info().Msgf("client forcibly disconnected %d %s", event.Reason, event.Text)

			client := server.Clients.FindClient(uint16(event.Client))

			if client == nil {
				continue
			}

			client.DisconnectFromServer()

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			client.Connection.Disconnect(int(event.Reason), event.Text)
		case packet := <-gamePackets:
			client := server.Clients.FindClient(uint16(packet.Client))

			if client == nil {
				continue
			}

			if client.GetServer() != packet.Server {
				continue
			}

			parseData := packet.Packet.Data
			messages, err := game.Read(parseData, false)
			if err != nil {
				log.Debug().
					Err(err).
					Uint16("client", client.Id).
					Msg("cluster -> client (failed to decode message)")

				// Forward it anyway
				client.Connection.Send(game.GamePacket{
					Channel: uint8(packet.Packet.Channel),
					Data:    packet.Packet.Data,
				})
				continue
			}

			channel := uint8(packet.Packet.Channel)

			// Sometimes clients are expecting messages to follow
			// each other directly; this is one of those cases
			// (arbitrary message passing between clients) and took
			// me too many hours of debugging
			if len(messages) > 0 && messages[0].Type() == game.N_CLIENT {
				log.Debug().
					Str("type", game.N_CLIENT.String()).
					Uint16("client", client.Id).
					Msgf("cluster -> client (%d messages)", len(messages)-1)

				client.Connection.Send(game.GamePacket{
					Channel: channel,
					Data:    packet.Packet.Data,
				})
			}

			// As opposed to client -> server, we don't actually need to do any filtering
			// so we won't repackage the messages individually
			for _, message := range messages {
				log.Debug().
					Str("type", message.Type().String()).
					Uint16("client", client.Id).
					Msg("cluster -> client")

				// Inject the auth domain to N_SERVINFO so the
				// client sends us N_CONNECT with their name
				// field filled
				if message.Type() == game.N_SERVINFO {
					info := message.Contents().(*game.ServerInfo)
					info.Domain = server.authDomain
					p := game.Packet{}
					p.PutInt(int32(game.N_SERVINFO))
					game.Marshal(&p, *info)
					client.Connection.Send(game.GamePacket{
						Channel: channel,
						Data:    p,
					})
					continue
				}

				client.Connection.Send(game.GamePacket{
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
		gameServer, err := server.manager.NewServer(ctx, serverConfig.Preset, true)
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

func (server *Cluster) DoAuthChallenge(ctx context.Context, client *clients.Client, id string) {
	if server.auth == nil {
		return
	}

	pair, err := server.auth.GetAuthKey(ctx, id)

	if err != nil || pair == nil {
		log.Warn().
			Uint16("client", client.Id).
			Err(err).
			Msg("no key for client to do auth challenge")
		return
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		log.Warn().
			Uint16("client", client.Id).
			Err(err).
			Msg("failed to generate auth challenge")
		return
	}

	client.Mutex.Lock()
	client.Challenge = challenge
	client.Mutex.Unlock()

	p := game.Packet{}
	p.PutInt(int32(game.N_AUTHCHAL))
	challengeMessage := game.AuthChallenge{
		Desc:      server.authDomain,
		Id:        0,
		Challenge: challenge.Question,
	}
	game.Marshal(&p, challengeMessage)
	client.Connection.Send(game.GamePacket{
		Channel: 1,
		Data:    p,
	})
}

func (server *Cluster) HandleChallengeAnswer(
	ctx context.Context,
	client *clients.Client,
	challenge *auth.Challenge,
	answer string,
) {
	if !challenge.Check(answer) {
		log.Warn().Uint16("client", client.Id).Msg("client failed auth challenge")
		client.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	user, err := server.auth.AuthenticateId(ctx, challenge.Id)
	if err != nil {
		log.Warn().Uint16("client", client.Id).Err(err).Msg("could not authenticate by id")
		client.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	// XXX we really need to move all the ENet auth to ingress/enet.go...
	client.Authentication <- user

	client.SendServerMessage(game.Blue(fmt.Sprintf("logged in with Discord as %s", user.Discord.Reference())))
	log.Info().
		Uint16("client", client.Id).
		Str("user", user.Discord.Reference()).
		Msg("logged in with Discord")
}

func (server *Cluster) GreetClient(ctx context.Context, client *clients.Client) {
	client.AnnounceELO()
}

func (server *Cluster) PollClient(ctx context.Context, client *clients.Client) {
	toServer := client.Connection.ReceivePackets()
	commands := client.Connection.ReceiveCommands()
	authentication := client.ReceiveAuthentication()
	disconnect := client.Connection.ReceiveDisconnect()

	// A context valid JUST for the lifetime of the client
	clientCtx, cancel := context.WithCancel(ctx)

	logger := log.With().Uint16("client", client.Id).Logger()

	defer client.Connection.Destroy()

	// If the user hasn't authenticated in a second, greet them normally.
	go func() {
		time.Sleep(5 * time.Second)

		if clientCtx.Err() != nil {
			return
		}

		client.Mutex.Lock()
		user := client.User
		client.Mutex.Unlock()
		if user == nil {
			server.GreetClient(clientCtx, client)
			client.SendServerMessage("You are not logged in. Your rating will not be saved.")
		}
	}()

	for {
		select {
		case <-ctx.Done():
			cancel()
			log.Info().Msg("cancelGreet")
			return
		case user := <-authentication:
			client.Mutex.Lock()
			client.User = user
			client.Mutex.Unlock()
			log.Info().Msg("user set")
			log.Info().Msg("cancelGreet")

			err := client.HydrateELOState(ctx, user)
			if err == nil {
				server.GreetClient(clientCtx, client)
				continue
			}

			if err != redis.Nil {
				log.Error().
					Err(err).
					Uint16("client", client.Id).
					Str("id", user.Discord.Id).
					Msg("failed to hydrate state for user")
				continue
			}

			// We save the initialized state that was there already
			err = client.SaveELOState(ctx)
			if err != nil {
				log.Error().
					Err(err).
					Uint16("client", client.Id).
					Str("id", user.Discord.Id).
					Msg("failed to save elo state for user")
			}
			server.GreetClient(clientCtx, client)
		case msg := <-toServer:
			data := msg.Data

			gameMessages, err := game.Read(data, true)
			if err != nil {
				log.Error().
					Err(err).
					Uint16("client", client.Id).
					Msg("client -> server (failed to decode message)")

				// Forward it anyway
				client.Mutex.Lock()
				if client.Server != nil {
					client.Server.SendData(client.Id, uint32(msg.Channel), msg.Data)
				}
				client.Mutex.Unlock()
				continue
			}

			passthrough := func(message game.Message) {
				client.Mutex.Lock()
				if client.Server != nil {
					client.Server.SendData(client.Id, uint32(msg.Channel), message.Data())
				}
				client.Mutex.Unlock()
			}

			for _, message := range gameMessages {
				if message.Type() == game.N_TEXT {
					text := message.Contents().(*game.Text).Text

					if strings.HasPrefix(text, "#") {
						command := text[1:]
						logger.Info().Str("command", command).Msg("intercepted command")

						// Only send this packet after we've checked
						// whether the cluster should handle it
						go func() {
							handled, response, err := server.RunCommandWithTimeout(clientCtx, command, client)

							if !handled {
								passthrough(message)
								return
							}

							if err != nil {
								client.SendServerMessage(game.Red(err.Error()))
								return
							} else if len(response) > 0 {
								client.SendServerMessage(response)
								return
							}

							if command == "help" {
								passthrough(message)
							}
						}()
						continue
					}
				}

				// Skip messages that aren't allowed while the
				// client is connecting, otherwise the server
				// (rightfully) disconnects us. This solves a
				// race condition when switching servers.
				client.Mutex.Lock()
				status := client.Status
				if status == clients.ClientStatusConnecting && !game.IsConnectingMessage(message.Type()) {
					client.Mutex.Unlock()
					continue
				}
				client.Mutex.Unlock()

				logger.Debug().Str("code", message.Type().String()).Msg("client -> server")

				if message.Type() == game.N_CONNECT {
					connect := message.Contents().(*game.Connect)

					description := connect.AuthDescription
					name := connect.AuthName

					connect.AuthDescription = ""
					connect.AuthName = ""
					p := game.Packet{}
					p.PutInt(int32(game.N_CONNECT))
					game.Marshal(&p, *connect)
					client.Server.SendData(client.Id, uint32(msg.Channel), p)

					if description == server.authDomain && client.GetUser() == nil {
						server.DoAuthChallenge(ctx, client, name)
					}
					continue
				}

				if message.Type() == game.N_AUTHANS {
					answerMessage := message.Contents().(*game.AuthAns)

					if answerMessage.Description == server.authDomain && client.Challenge != nil {
						server.HandleChallengeAnswer(
							ctx,
							client,
							client.Challenge,
							answerMessage.Answer,
						)
						continue
					}
				}

				if message.Type() == game.N_MAPCRC {
					client.RestoreMessages()
				}

				client.Mutex.Lock()
				if client.Server != nil {
					client.Server.SendData(client.Id, uint32(msg.Channel), message.Data())
				}
				client.Mutex.Unlock()
			}

		case request := <-commands:
			command := request.Command
			outChannel := request.Response

			go func() {
				handled, response, err := server.RunCommandWithTimeout(clientCtx, command, client)
				outChannel <- clients.CommandResult{
					Handled:  handled,
					Err:      err,
					Response: response,
				}
			}()
		case <-disconnect:
			client.DisconnectFromServer()
		}
	}
}

func (server *Cluster) PollClients(ctx context.Context) {
	newClients := server.Clients.ReceiveClients()

	for {
		select {
		case client := <-newClients:
			go server.PollClient(ctx, client)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
}
