package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"
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
		user.Message(message)
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
				m := io.Packet{}
				m.Put(
					P.N_TEXT,
					message,
				)

				p := io.Packet{}
				p.Put(
					P.N_CLIENT,
					senderNum,
					len(m),
				)

				p = append(p, m...)

				user.Send(io.RawPacket{
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

func (c *Cluster) PollFromMessages(ctx context.Context, user *User) {
	userCtx := user.Context()

	passthrough := func(channel uint8, message P.Message) {
		server := user.GetServer()
		if server != nil {
			server.SendData(
				user.Id,
				uint32(channel),
				message.Data(),
			)
		}
	}

	chats := user.From.Intercept(P.N_TEXT)
	blockConnecting := user.From.InterceptWith(func(code P.MessageCode) bool {
		return !P.IsConnectingMessage(code)
	})
	connects := user.From.Intercept(P.N_CONNECT)
	teleports := user.From.Intercept(P.N_TELEPORT)
	edits := user.From.InterceptWith(P.IsOwnerOnly)
	crcs := user.From.Intercept(P.N_MAPCRC)
	votes := user.From.Intercept(P.N_MAPVOTE)

	for {
		logger := user.Logger()

		select {
		case <-userCtx.Done():
			return
		case msg := <-votes.Receive():
			space := user.GetSpace()
			if space == nil {
				msg.Pass()
				continue
			}

			preset := space.PresetSpace
			if preset == nil || !preset.VotingCreates {
				msg.Pass()
				continue
			}

			vote := msg.Message.Contents().(*game.MapVote)
			if vote.Mode < 0 || vote.Mode >= len(MODE_NAMES) {
				msg.Pass()
				continue
			}

			msg.Drop()

			err := c.RunOnBehalf(
				ctx,
				fmt.Sprintf(
					"creategame %s %s",
					MODE_NAMES[vote.Mode],
					vote.Map,
				),
				user,
			)
			if err != nil {
				logger.Error().Err(err).Msg("failed to create game from vote")
			}
		case msg := <-crcs.Receive():
			msg.Pass()
			user.RestoreMessages()
			crc := msg.Message.Contents().(*game.MapCRC)
			// The client does not have the map
			if crc.Crc == 0 {
				go func() {
					err := c.SendMap(ctx, user, crc.Map)
					if err != nil {
						logger.Warn().Err(err).Msg("failed to send map to client")
					}
				}()
			}
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
					user.ConnectToSpace(space.Deployment.GetServer(), space.GetID())
					user.Message("you cannot edit this space.")
					msg.Drop()
					continue
				}
			}

			server := user.GetServer()
			// For now, users can't edit on named servers (ie the lobby)
			if server != nil && server.Alias != "" {
				user.Connect(server)
				user.Message("you cannot edit this server.")
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
			msg.Pass()

			if user.Connection.Type() != ingress.ClientTypeENet {
				continue
			}

			go func() {
				user.Mutex.RLock()
				wasGreeted := user.wasGreeted
				user.Mutex.RUnlock()
				if wasGreeted {
					return
				}

				if description != c.authDomain || c.auth == nil {
					user.Authentication <- nil
					return
				}

				if !user.IsLoggedIn() {
					err := c.DoAuthChallenge(ctx, user, name)
					if err != nil {
						logger.Warn().Err(err).Msgf("failed to log in")
					}
				}
			}()
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

			if len(text) >= io.MAXSTRLEN {
				removed := len(text) - game.MAXSTRLEN
				text = text[:game.MAXSTRLEN]
				user.Message(game.Red(
					fmt.Sprintf("your message was too long; we cut off the last %d characters", removed),
				))
			}

			if !strings.HasPrefix(text, "#") {
				// We do our own chat, don't pass on to the server
				c.ForwardGlobalChat(userCtx, user, text)
				continue
			}

			command := text[1:]

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
					logger.Error().Err(err).Str("command", command).Msg("user command failed")
					user.Message(game.Red(err.Error()))
					return
				} else if len(response) > 0 {
					user.Message(response)
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

func (c *Cluster) PollToMessages(ctx context.Context, user *User) {
	userCtx := user.Context()
	serverInfo := user.To.Intercept(P.N_SERVINFO)
	spawnState := user.To.Intercept(P.N_SPAWNSTATE)

	for {
		select {
		case <-userCtx.Done():
			return
		case msg := <-serverInfo.Receive():
			// Inject the auth domain to N_SERVINFO so the
			// client sends us N_CONNECT with their name
			// field filled
			info := msg.Message.Contents().(*game.ServerInfo)

			user.Mutex.RLock()
			wasGreeted := user.wasGreeted
			user.Mutex.RUnlock()
			if !wasGreeted {
				info.Domain = c.authDomain
			}

			user.Mutex.Lock()
			user.lastDescription = info.Description
			user.Mutex.Unlock()

			p := game.Packet{}
			p.PutInt(int32(P.N_SERVINFO))
			p.Put(*info)
			msg.Replace(p)
		case msg := <-spawnState.Receive():
			msg.Pass()
			state := msg.Message.Contents().(*game.SpawnState)
			user.Mutex.Lock()
			user.LifeSequence = state.LifeSequence
			user.Mutex.Unlock()
		}
	}
}

func (c *Cluster) PollUser(ctx context.Context, user *User) {
	commands := user.Connection.ReceiveCommands()
	authentication := user.ReceiveAuthentication()
	disconnect := user.Connection.ReceiveDisconnect()

	toServer := user.Connection.ReceivePackets()
	toClient := user.ReceiveToMessages()

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

	go c.PollFromMessages(ctx, user)
	go c.PollToMessages(ctx, user)

	sendResult := func(out chan bool, packet game.GamePacket) {
		done := user.Connection.Send(game.GamePacket{
			Channel: uint8(packet.Channel),
			Data:    packet.Data,
		})

		go func() {
			select {
			case result := <-done:
				out <- result
			case <-userCtx.Done():
				return
			}

		}()
	}

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

			// We want to get the raw data from the user -- not the
			// deserialized messages
			user.Client.Intercept.From <- game.GamePacket{
				Data:    data,
				Channel: msg.Channel,
			}

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
				if !game.IsSpammyMessage(message.Type()) {
					logger.Debug().Str("code", message.Type().String()).Msg("client -> server")
				}

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

				server := user.GetServer()
				if server == nil {
					continue
				}

				newMessage, err := server.To.Process(
					ctx,
					msg.Channel,
					message,
				)
				if err != nil {
					log.Error().Err(err).Msgf("failed to process message for server")
					continue
				}

				server.SendData(user.Id, uint32(msg.Channel), newMessage.Data())
			}

		case msg := <-toClient:
			packet := msg.Packet
			done := msg.Done

			gameMessages, err := game.Read(packet.Data, false)
			if err != nil {
				logger.Warn().
					Err(err).
					Msg("cluster -> client (failed to decode message)")

				user.Client.Intercept.To <- packet

				// Forward it anyway
				sendResult(done, packet)
				continue
			}

			channel := uint8(packet.Channel)
			out := make([]byte, 0)
			filtered := make([]byte, 0)

			for _, message := range gameMessages {
				type_ := message.Type()
				if !game.IsSpammyMessage(type_) {
					logger.Debug().
						Str("type", message.Type().String()).
						Msg("cluster -> client")
				}

				newMessage, err := user.To.Process(
					ctx,
					channel,
					message,
				)
				if err != nil {
					log.Error().Err(err).Msgf("failed to process message")
					continue
				}

				if newMessage == nil {
					continue
				}

				out = append(out, newMessage.Data()...)

				if type_ == P.N_SENDDEMO || type_ == P.N_SENDMAP {
					continue
				}

				filtered = append(filtered, newMessage.Data()...)
			}

			user.Client.Intercept.To <- io.RawPacket{
				Data:    filtered,
				Channel: channel,
			}

			sendResult(done, io.RawPacket{
				Channel: channel,
				Data:    out,
			})

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
