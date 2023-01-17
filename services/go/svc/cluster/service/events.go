package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/go-redis/redis/v9"
	"github.com/rs/zerolog/log"
)

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

	server.Users.Mutex.RLock()
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
	server.Users.Mutex.RUnlock()
}

func (server *Cluster) NotifyNameChange(ctx context.Context, user *User, oldName string) {
	newName := user.GetName()

	if newName == oldName {
		return
	}

	clientServer := user.GetServer()
	serverName := user.GetServerName()
	message := fmt.Sprintf("%s now known as %s [%s]", oldName, newName, serverName)

	server.Users.Mutex.RLock()
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
	server.Users.Mutex.RUnlock()
}

func (c *Cluster) AnnounceInServer(ctx context.Context, server *servers.GameServer, message string) {
	c.Users.Mutex.RLock()

	serverUsers, ok := c.Users.Servers[server]
	if !ok {
		return
	}

	for _, user := range serverUsers {
		user.SendServerMessage(message)
	}

	c.Users.Mutex.RUnlock()
}

func (server *Cluster) ForwardGlobalChat(ctx context.Context, sender *User, message string) {
	server.Users.Mutex.RLock()
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
	server.Users.Mutex.RUnlock()
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

func (c *Cluster) SendMap(ctx context.Context, user *User, name string) error {
	server := user.GetServer()

	instance := c.spaces.FindInstance(server)

	if instance != nil && instance.Editing != nil {
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

	data, err := c.assets.FetchMapBytes(ctx, name)
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

	log.Info().Msgf("Sent map %s (%d) to client", name, len(data))

	return nil
}

func (c *Cluster) HandleTeleport(ctx context.Context, user *User, source int) {
	logger := user.Logger()

	space := user.GetSpace()
	server := user.GetServer()
	if server != nil && space != nil {
		links, err := space.GetLinks(ctx)
		if err != nil {
			return
		}

		entities := server.GetEntities()
		if source < 0 || source >= len(entities) {
			return
		}

		teleport := entities[source]

		for _, link := range links {
			if link.ID == uint8(teleport.Attr1) {
				logger.Info().Msgf("teleported to %s", link.Destination)
				go c.RunCommandWithTimeout(
					user.Context(),
					fmt.Sprintf("go %s", link.Destination),
					user,
				)
			}
		}
	}
}

func (c *Cluster) PollMessages(ctx context.Context, user *User) {
	userCtx := user.Context()

	passthrough := func(channel uint8, message game.Message) {
		server := user.GetServer()
		if server != nil {
			server.SendData(
				user.Id,
				uint32(channel),
				message.Data(),
			)
		}
	}

	chats := user.From.Intercept(game.N_TEXT)
	blockConnecting := user.From.InterceptWith(func (code game.MessageCode) bool {
		return !game.IsConnectingMessage(code)
	})
	connects := user.From.Intercept(game.N_CONNECT)
	teleports := user.From.Intercept(game.N_TELEPORT)
	edits := user.From.InterceptWith(game.IsOwnerOnly)

	for {
		logger := user.Logger()

		select {
		case <-ctx.Done():
			return
		case msg := <-edits.Receive():
			isOwner, err := user.IsOwner(ctx)
			if err != nil {
				msg.Drop()
				continue
			}

			space := user.GetSpace()
			if space != nil {
				canEditSpace := isOwner || space.IsOpenEdit()
				if !canEditSpace {
					user.ConnectToSpace(space.Server, space.GetID())
					user.SendServerMessage("you cannot edit this space.")
					msg.Drop()
					continue
				}
			}

			server := user.GetServer()
			// For now, users can't edit on named servers (ie the lobby)
			if server != nil && server.Alias != "" {
				user.Connect(server)
				user.SendServerMessage("you cannot edit this server.")
				msg.Drop()
				continue
			}
			msg.Pass()
		case msg := <-teleports.Receive():
			message := msg.Message
			teleport := message.Contents().(*game.Teleport)
			c.HandleTeleport(ctx, user, teleport.Source)
			msg.Pass()
		case msg := <-connects.Receive():
			message := msg.Message
			connect := message.Contents().(*game.Connect)

			description := connect.AuthDescription
			name := connect.AuthName

			connect.AuthDescription = ""
			connect.AuthName = ""
			p := game.Packet{}
			p.PutInt(int32(game.N_CONNECT))
			p.Put(*connect)
			msg.Replace(p)

			if description == c.authDomain && user.GetAuth() == nil {
				c.DoAuthChallenge(ctx, user, name)
			}
			continue
		case msg := <-blockConnecting.Receive():
			// Skip messages that aren't allowed while the
			// client is connecting, otherwise the server
			// (rightfully) disconnects us. This solves a
			// race condition when switching servers.
			status := user.GetStatus()
			if status == clients.ClientStatusConnecting {
				msg.Drop()
				continue
			}
			msg.Pass()
		case msg := <-chats.Receive():
			message := msg.Message
			text := message.Contents().(*game.Text).Text

			msg.Drop()

			if text == "a" && c.MapSender.IsHandling(user) {
				c.MapSender.TriggerSend(ctx, user)
				continue
			}

			if !strings.HasPrefix(text, "#") {
				// We do our own chat, don't pass on to the server
				c.ForwardGlobalChat(userCtx, user, text)
				continue
			}

			command := text[1:]
			logger.Info().Str("command", command).Msg("intercepted command")

			// Only send this packet after we've checked
			// whether the cluster should handle it
			go func() {
				handled, response, err := c.RunCommandWithTimeout(userCtx, command, user)

				if !handled {
					passthrough(msg.Channel, message)
					msg.Pass()
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
					passthrough(msg.Channel, message)
				}
			}()
			continue
		}
	}
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
		err := RecordSession(
			userCtx,
			c.redis,
			c.settings.LogSessions,
			user,
		)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to record client session")
		}
	}()

	defer user.Connection.Destroy()

	go c.PollMessages(ctx, user)

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

			verseUser, err := c.verse.GetOrCreateUser(userCtx, authUser)
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
				server := user.GetServer()
				if server == nil {
					continue
				}

				// Forward it anyway
				server.SendData(user.Id, uint32(msg.Channel), msg.Data)
				continue
			}

			for _, message := range gameMessages {
				logger.Debug().Str("code", message.Type().String()).Msg("client -> server")

				data, err := user.From.Process(
					ctx,
					msg.Channel,
					message,
				)
				if err != nil {
					log.Error().Err(err).Msgf("failed to process message")
					continue
				}

				if data == nil {
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

				if message.Type() == game.N_MAPCRC {
					user.RestoreMessages()

					crc := message.Contents().(*game.MapCRC)
					// The client does not have the map
					if crc.Crc == 0 {
						go func() {
							err := c.SendMap(ctx, user, crc.Map)
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

				server := user.GetServer()
				if server != nil {
					server.SendData(user.Id, uint32(msg.Channel), data)
				}
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
