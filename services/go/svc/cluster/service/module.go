package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/mmr"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/verse"

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
	// State
	createMutex sync.Mutex
	// host -> time a client from that host last created a server. We
	// REALLY don't want clients to be able to DDOS us
	lastCreate map[string]time.Time
	// host -> the server created by that host
	// each host can only have one server at once
	hostServers   map[string]*servers.GameServer
	startTime     time.Time
	authDomain    string
	settings      config.ClusterSettings
	serverCtx     context.Context
	serverMessage chan []byte

	// Services
	Clients   *clients.ClientManager
	Users     *UserOrchestrator
	MapSender *MapSender
	auth      *auth.DiscordService
	manager   *servers.ServerManager
	matches   *Matchmaker
	redis     *redis.Client
	spaces    *verse.SpaceManager
	verse     *verse.Verse
}

func NewCluster(
	ctx context.Context,
	serverManager *servers.ServerManager,
	maps *assets.MapFetcher,
	sender *MapSender,
	settings config.ClusterSettings,
	authDomain string,
	auth *auth.DiscordService,
	redis *redis.Client,
) *Cluster {
	clients := clients.NewClientManager()
	v := verse.NewVerse(redis)
	server := &Cluster{
		Users:         NewUserOrchestrator(redis, settings.Matchmaking.Duel),
		MapSender:     sender,
		serverCtx:     ctx,
		settings:      settings,
		authDomain:    authDomain,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		Clients:       clients,
		matches:       NewMatchmaker(serverManager, settings.Matchmaking.Duel),
		serverMessage: make(chan []byte, 1),
		manager:       serverManager,
		startTime:     time.Now(),
		auth:          auth,
		redis:         redis,
		verse:         v,
		spaces:        verse.NewSpaceManager(v, serverManager, maps),
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

func (server *Cluster) GetTeamInfo() *servers.TeamInfo {
	info := servers.TeamInfo{
		IsDeathmatch: true,
		GameMode:     0,
		TimeLeft:     9999,
		Scores:       make([]servers.TeamScore, 0),
	}
	return &info
}

// We need client information, so this is not on the ServerManager like GetServerInfo is
func (server *Cluster) GetClientInfo() []*servers.ClientExtInfo {
	info := make([]*servers.ClientExtInfo, 0)

	server.manager.Mutex.Lock()

	for _, gameServer := range server.manager.Servers {
		clients := gameServer.GetClientInfo()
		for _, client := range clients {
			newClient := *client

			// TODO do we still want client ids here?

			info = append(info, &newClient)
		}
	}

	server.manager.Mutex.Unlock()

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

			server.Users.Mutex.Lock()
			for _, user := range server.Users.Users {
				if user == winner || user == loser {
					continue
				}
				user.SendServerMessage(message)
			}
			server.Users.Mutex.Unlock()
		case queue := <-queues:
			server.Users.Mutex.Lock()
			for _, client := range server.Users.Users {
				if client == queue.User {
					continue
				}

				client.SendServerMessage(fmt.Sprintf(
					"%s queued for %s",
					client.Reference(),
					queue.Type,
				))
			}
			server.Users.Mutex.Unlock()
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
			user := server.Users.FindUser(uint16(join.Client))

			if user == nil {
				continue
			}

			user.Mutex.Lock()
			if user.Server != nil {
				instance := server.spaces.FindInstance(user.Server)
				if instance != nil {
					user.Space = instance
				}
				user.Status = clients.ClientStatusConnected
				user.Num = join.ClientNum
			}
			user.Mutex.Unlock()

			logger := user.Logger()
			logger.Info().Msg("connected to server")

			isHome, err := user.IsAtHome(ctx)
			if err != nil {
				logger.Warn().Err(err).Msg("failed seeing if user was at home")
				continue
			}

			if isHome {
				space := user.GetSpace()
				message := fmt.Sprintf(
					"welcome to your home (space %s).",
					space.GetID(),
				)

				if user.IsLoggedIn() {
					user.SendServerMessage(message)
					user.SendServerMessage("editing by others is disabled. say #edit to enable it.")
				} else {
					user.SendServerMessage(message + " anyone can edit it. because you are not logged in, it will be deleted in 4 hours")
				}
			}

		case event := <-names:
			user := server.Users.FindUser(uint16(event.Client))

			if user == nil {
				continue
			}

			user.Mutex.Lock()
			user.Name = event.Name
			user.Mutex.Unlock()

			logger := user.Logger()
			logger.Info().Msg("client has new name")
			server.NotifyNameChange(ctx, user, event.Name)

		case event := <-forceDisconnects:
			log.Info().Msgf("client forcibly disconnected %d %s", event.Reason, event.Text)

			user := server.Users.FindUser(uint16(event.Client))

			if user == nil {
				continue
			}

			user.DisconnectFromServer()

			// TODO ideally we would move clients back to the lobby if they
			// were not kicked for violent reasons
			user.Connection.Disconnect(int(event.Reason), event.Text)
		case packet := <-gamePackets:
			user := server.Users.FindUser(uint16(packet.Client))

			if user == nil {
				continue
			}

			if user.GetServer() != packet.Server {
				continue
			}

			logger := user.Logger()

			parseData := packet.Packet.Data
			messages, err := game.Read(parseData, false)
			if err != nil {
				logger.Debug().
					Err(err).
					Msg("cluster -> client (failed to decode message)")

				// Forward it anyway
				user.Send(game.GamePacket{
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
				logger.Debug().
					Str("type", game.N_CLIENT.String()).
					Msgf("cluster -> client (%d messages)", len(messages)-1)

				user.Send(game.GamePacket{
					Channel: channel,
					Data:    packet.Packet.Data,
				})
			}

			// As opposed to client -> server, we don't actually need to do any filtering
			// so we won't repackage the messages individually
			for _, message := range messages {
				logger.Debug().
					Str("type", message.Type().String()).
					Msg("cluster -> client")

				// Inject the auth domain to N_SERVINFO so the
				// client sends us N_CONNECT with their name
				// field filled
				if message.Type() == game.N_SERVINFO {
					info := message.Contents().(*game.ServerInfo)
					info.Domain = server.authDomain
					p := game.Packet{}
					p.PutInt(int32(game.N_SERVINFO))
					p.Put(*info)
					user.Send(game.GamePacket{
						Channel: channel,
						Data:    p,
					})
					continue
				}

				if message.Type() == game.N_SPAWNSTATE {
					state := message.Contents().(*game.SpawnState)
					user.Mutex.Lock()
					user.LifeSequence = state.LifeSequence
					user.Mutex.Unlock()
				}

				user.Send(game.GamePacket{
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

func (server *Cluster) DoAuthChallenge(ctx context.Context, user *User, id string) {
	if server.auth == nil {
		return
	}

	pair, err := server.auth.GetAuthKey(ctx, id)

	logger := user.Logger()

	if err != nil || pair == nil {
		logger.Warn().
			Err(err).
			Msg("no key for client to do auth challenge")
		return
	}

	challenge, err := auth.GenerateChallenge(id, pair.Public)
	if err != nil {
		logger.Warn().
			Err(err).
			Msg("failed to generate auth challenge")
		return
	}

	user.Mutex.Lock()
	user.Challenge = challenge
	user.Mutex.Unlock()

	p := game.Packet{}
	p.PutInt(int32(game.N_AUTHCHAL))
	challengeMessage := game.AuthChallenge{
		Desc:      server.authDomain,
		Id:        0,
		Challenge: challenge.Question,
	}
	p.Put(challengeMessage)
	user.Send(game.GamePacket{
		Channel: 1,
		Data:    p,
	})
}

func (server *Cluster) HandleChallengeAnswer(
	ctx context.Context,
	user *User,
	challenge *auth.Challenge,
	answer string,
) {
	logger := user.Logger()
	if !challenge.Check(answer) {
		logger.Warn().Msg("client failed auth challenge")
		user.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	authUser, err := server.auth.AuthenticateId(ctx, challenge.Id)
	if err != nil {
		logger.Warn().Err(err).Msg("could not authenticate by id")
		user.SendServerMessage(game.Red("failed to login, please regenerate your key"))
		return
	}

	// XXX we really need to move all the ENet auth to ingress/enet.go...
	user.Authentication <- authUser

	user.SendServerMessage(game.Blue(fmt.Sprintf("logged in with Discord as %s", authUser.Discord.Reference())))
	logger = user.Logger()
	logger.Info().Msg("logged in with Discord")
}

func (server *Cluster) GreetClient(ctx context.Context, user *User) {
	user.AnnounceELO()
	if user.Auth == nil {
		user.SendServerMessage("You are not logged in. Your rating will not be saved.")
	}
	server.NotifyClientChange(ctx, user, true)
}

func (server *Cluster) NotifyClientChange(ctx context.Context, user *User, joined bool) {
	userServer := user.GetServer()
	name := user.GetFormattedName()
	serverName := user.GetServerName()

	event := "join"
	if !joined {
		event = "leave"
	}

	// To users on another server
	message := fmt.Sprintf("%s: %s (%s)", event, name, serverName)

	server.Users.Mutex.Lock()
	for _, other := range server.Users.Users {
		if other == user {
			continue
		}

		otherServer := other.GetServer()

		// On the same server, we can just use chat
		if userServer == otherServer {
			continue
		}
		other.Client.SendMessage(message)
	}
	server.Users.Mutex.Unlock()
}

func (server *Cluster) NotifyNameChange(ctx context.Context, user *User, oldName string) {
	newName := user.GetName()

	if newName == oldName {
		return
	}

	clientServer := user.GetServer()
	serverName := user.GetServerName()
	message := fmt.Sprintf("%s now known as %s [%s]", oldName, newName, serverName)

	server.Users.Mutex.Lock()
	for _, other := range server.Users.Users {
		if other == user {
			continue
		}

		otherServer := other.GetServer()

		// On the same server, we can just use chat
		if clientServer == otherServer {
			continue
		}
		other.Client.SendMessage(message)
	}
	server.Users.Mutex.Unlock()
}

func (c *Cluster) AnnounceInServer(ctx context.Context, server *servers.GameServer, message string) {
	c.Users.Mutex.Lock()

	serverUsers, ok := c.Users.Servers[server]
	if !ok {
		return
	}

	for _, user := range serverUsers {
		user.SendServerMessage(message)
	}

	c.Users.Mutex.Unlock()
}

func (server *Cluster) ForwardGlobalChat(ctx context.Context, sender *User, message string) {
	server.Users.Mutex.Lock()
	senderServer := sender.GetServer()
	senderNum := sender.GetClientNum()

	name := sender.GetFormattedName()

	// To users who share the same server
	sameMessage := fmt.Sprintf("%s: %s", name, game.Green(message))

	serverName := senderServer.GetFormattedReference()

	// To users on another server
	otherMessage := fmt.Sprintf("%s [%s]: %s", name, serverName, game.Green(message))

	for _, user := range server.Users.Users {
		if user == sender {
			continue
		}

		otherServer := user.GetServer()

		// On the same server, we can just use chat
		if senderServer == otherServer {
			if user.Connection.Type() == ingress.ClientTypeWS {
				user.Connection.SendGlobalChat(sameMessage)
			} else {
				// We lose the formatting, but that's OK
				m := game.Packet{}
				m.Put(
					game.N_TEXT,
					message,
				)

				p := game.Packet{}
				p.Put(
					game.N_CLIENT,
					senderNum,
					len(m),
				)

				p = append(p, m...)

				user.Send(game.GamePacket{
					Channel: 1,
					Data:    p,
				})
			}
			continue
		}

		user.Connection.SendGlobalChat(otherMessage)
	}
	server.Users.Mutex.Unlock()
}

// TODO
//func (server *Cluster) SendDesktopMap(ctx context.Context, client *clients.Client) {
//if server.MapSender.IsHandling(client) {
//return
//}

//gameServer := client.GetServer()
//if gameServer == nil {
//return
//}

//gameServer.Mutex.Lock()
//mapName := gameServer.Map
//isBuilt := gameServer.IsBuiltMap
//gameServer.Mutex.Unlock()

//if !isBuilt {
//return
//}

//server.MapSender.SendMap(ctx, client, mapName)
//}

func (c *Cluster) SendMap(ctx context.Context, user *User) error {
	server := user.GetServer()

	instance := c.spaces.FindInstance(server)

	if instance != nil {
		e := instance.Editing
		err := e.Checkpoint(ctx)
		if err != nil {
			return err
		}

		data, err := e.Map.LoadMapData(ctx)
		if err != nil {
			return err
		}

		p := game.Packet{}
		p.Put(game.N_SENDMAP)
		p = append(p, data...)

		user.Send(game.GamePacket{
			Channel: 2,
			Data:    p,
		})

		return nil
	}

	return nil
}

func (c *Cluster) PollUser(ctx context.Context, user *User) {
	toServer := user.Connection.ReceivePackets()
	commands := user.Connection.ReceiveCommands()
	authentication := user.ReceiveAuthentication()
	disconnect := user.Connection.ReceiveDisconnect()

	// A context valid JUST for the lifetime of the user
	userCtx := user.Context()

	logger := user.Logger()

	go func() {
		err := RecordSession(userCtx, c.settings.SessionDirectory, user)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to record client session")
		}
	}()

	defer user.Connection.Destroy()

	for {
		logger = user.Logger()
		select {
		case <-ctx.Done():
			return
		case authUser := <-authentication:
			if authUser == nil {
				c.GreetClient(userCtx, user)
				continue
			}

			logger = logger.With().Str("id", authUser.Discord.Id).Logger()

			user.Mutex.Lock()
			user.Auth = authUser
			user.Mutex.Unlock()

			verseUser, err := c.verse.GetOrCreateUser(userCtx, user.Auth.GetID())
			if err != nil {
				logger.Error().Err(err).Msg("failed to get verse state for user")
				continue
			}
			user.Verse = verseUser

			err = user.HydrateELOState(ctx, authUser)
			if err == nil {
				c.GreetClient(userCtx, user)
				continue
			}

			if err != redis.Nil {
				logger.Error().Err(err).Msg("failed to hydrate state for user")
				continue
			}

			// We save the initialized state that was there already
			err = user.SaveELOState(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("failed to save elo state for user")
			}
			c.GreetClient(userCtx, user)
		case msg := <-toServer:
			data := msg.Data

			user.Intercept.From <- msg

			gameMessages, err := game.Read(data, true)
			if err != nil {
				logger.Error().Err(err).
					Msg("client -> server (failed to decode message)")

				// Forward it anyway
				user.Mutex.Lock()
				if user.Server != nil {
					user.Server.SendData(user.Id, uint32(msg.Channel), msg.Data)
				}
				user.Mutex.Unlock()
				continue
			}

			passthrough := func(message game.Message) {
				user.Mutex.Lock()
				if user.Server != nil {
					user.Server.SendData(user.Id, uint32(msg.Channel), message.Data())
				}
				user.Mutex.Unlock()
			}

			for _, message := range gameMessages {
				if message.Type() == game.N_TEXT {
					text := message.Contents().(*game.Text).Text

					if text == "a" && c.MapSender.IsHandling(user) {
						c.MapSender.TriggerSend(ctx, user)
						continue
					}

					if strings.HasPrefix(text, "#") {
						command := text[1:]
						logger.Info().Str("command", command).Msg("intercepted command")

						// Only send this packet after we've checked
						// whether the cluster should handle it
						go func() {
							handled, response, err := c.RunCommandWithTimeout(userCtx, command, user)

							if !handled {
								passthrough(message)
								return
							}

							if err != nil {
								user.SendServerMessage(game.Red(err.Error()))
								return
							} else if len(response) > 0 {
								user.SendServerMessage(response)
								return
							}

							if command == "help" {
								passthrough(message)
							}
						}()
						continue
					} else {
						// We do our own chat, don't pass on to the server
						c.ForwardGlobalChat(userCtx, user, text)
						continue
					}
				}

				// Skip messages that aren't allowed while the
				// client is connecting, otherwise the server
				// (rightfully) disconnects us. This solves a
				// race condition when switching servers.
				user.Mutex.Lock()
				status := user.Status
				if status == clients.ClientStatusConnecting && !game.IsConnectingMessage(message.Type()) {
					user.Mutex.Unlock()
					continue
				}
				user.Mutex.Unlock()

				logger.Debug().Str("code", message.Type().String()).Msg("client -> server")

				if message.Type() == game.N_CONNECT {
					connect := message.Contents().(*game.Connect)

					description := connect.AuthDescription
					name := connect.AuthName

					connect.AuthDescription = ""
					connect.AuthName = ""
					p := game.Packet{}
					p.PutInt(int32(game.N_CONNECT))
					p.Put(*connect)
					user.Server.SendData(user.Id, uint32(msg.Channel), p)

					if description == c.authDomain && user.GetAuth() == nil {
						c.DoAuthChallenge(ctx, user, name)
					}
					continue
				}

				if message.Type() == game.N_AUTHANS {
					answerMessage := message.Contents().(*game.AuthAns)

					if answerMessage.Description == c.authDomain && user.Challenge != nil {
						c.HandleChallengeAnswer(
							ctx,
							user,
							user.Challenge,
							answerMessage.Answer,
						)
						continue
					}
				}

				if message.Type() == game.N_TELEPORT {
					//teleport := message.Contents().(*game.Teleport)
					//log.Info().Msgf("client %s teleported to %d", client.Reference(), teleport.Destination)
					//if teleport.Destination == 10 {
					//go server.RunCommandWithTimeout(clientCtx, "creategame complex", client)
					//}
				}

				if game.IsOwnerOnly(message.Type()) {
					isOwner, err := user.IsOwner(ctx)
					if err != nil {
						continue
					}

					space := user.GetSpace()
					if space != nil {
						canEditSpace := isOwner || space.IsOpenEdit()
						if !canEditSpace {
							user.ConnectToSpace(space.Server, space.GetID())
							user.SendServerMessage("You cannot edit this space.")
							continue
						}
					}

					server := user.GetServer()
					// For now, users can't edit on named servers (ie the lobby)
					if server != nil && server.Alias != "" {
						user.Connect(server)
						user.SendServerMessage("You cannot edit this server.")
						continue
					}
				}

				if message.Type() == game.N_MAPCRC {
					user.RestoreMessages()

					crc := message.Contents().(*game.MapCRC)
					// The client does not have the map
					if crc.Crc == 0 {
						go func() {
							err := c.SendMap(ctx, user)
							if err != nil {
								logger.Warn().Err(err).Msg("failed to send map to client")
							}
						}()
					}
				}

				if message.Type() == game.N_GETDEMO && c.MapSender.IsHandling(user) {
					demo := message.Contents().(*game.GetDemo)
					c.MapSender.SendDemo(ctx, user, demo.Tag)
					continue
				}

				user.Mutex.Lock()
				if user.Server != nil {
					user.Server.SendData(user.Id, uint32(msg.Channel), message.Data())
				}
				user.Mutex.Unlock()
			}

		case request := <-commands:
			command := request.Command
			outChannel := request.Response

			go func() {
				handled, response, err := c.RunCommandWithTimeout(userCtx, command, user)
				outChannel <- ingress.CommandResult{
					Handled:  handled,
					Err:      err,
					Response: response,
				}
			}()
		case <-disconnect:
			c.NotifyClientChange(ctx, user, false)
			user.DisconnectFromServer()
		}
	}
}

func (server *Cluster) PollUsers(ctx context.Context, newConnections chan ingress.Connection) {
	newClients := server.Clients.ReceiveClients()

	for {
		select {
		case connection := <-newConnections:
			server.Clients.AddClient(connection)
		case client := <-newClients:
			user := server.Users.AddUser(ctx, client)
			go server.PollUser(ctx, user)
		case <-ctx.Done():
			return
		}
	}
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
	server.MapSender.Shutdown()
}
