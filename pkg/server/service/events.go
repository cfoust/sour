package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/chanlock"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/game/io"
	P "github.com/cfoust/sour/pkg/game/protocol"
	S "github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/server/servers"

	"github.com/rs/zerolog/log"
)

func (server *Cluster) NotifyClientChange(ctx context.Context, user *User, joined bool) {
	if status := user.GetStatus(); status == UserStatusDisconnected {
		return
	}

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
		other.RawMessage(message)
	}
	server.Users.Mutex.RUnlock()
}

func (c *Cluster) NotifyNameChange(ctx context.Context, user *User, name string) {
	logger := user.Logger()
	oldName := user.GetName()

	if name == oldName {
		return
	}

	logger.Info().Msg("client has new name")

	user.SetName(ctx, name)

	clientServer := user.GetServer()
	serverName := user.GetServerName()
	message := fmt.Sprintf("%s now known as %s [%s]", oldName, name, serverName)

	c.Users.Mutex.RLock()
	for _, other := range c.Users.Users {
		if other == user {
			continue
		}

		otherServer := other.GetServer()

		// On the same server, we can just use chat
		if clientServer == otherServer {
			continue
		}
		other.RawMessage(message)
	}
	c.Users.Mutex.RUnlock()
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

		if user.Connection.Type() == ingress.ClientTypeWS {
			ws := user.Connection.(*ingress.WSClient)
			ws.SendGlobalChat(sameMessage)
			continue
		}

		// On the same server, we can just use chat
		if senderServer == otherServer {
			// We lose the formatting, but that's OK
			user.Send(
				P.ClientPacket{Client: int32(senderNum)},
				P.Text{Text: message},
			)
			continue
		}

		user.RawMessage(otherMessage)
	}
	server.Users.Mutex.RUnlock()
}

func (c *Cluster) HandleTeleport(ctx context.Context, user *User, source int32) {
	space := user.GetSpace()
	server := user.GetServer()
	if server != nil && space != nil {
		links, err := space.GetLinks(ctx)
		if err != nil {
			return
		}

		entities := server.GetEntities()
		if source < 0 || source >= int32(len(entities)) {
			return
		}

		teleport := entities[source]

		for _, link := range links {
			if link.Teleport != uint8(teleport.Attr1) {
				continue
			}
			go c.HandleCommand(
				user.Ctx(),
				user,
				fmt.Sprintf("go %s", link.Destination),
			)
			return
		}
	}
}

func (c *Cluster) PollFromMessages(ctx context.Context, user *User) {
	userCtx := user.Ctx()

	chats := user.From.Intercept(P.N_TEXT)
	serverCommands := user.From.Intercept(P.N_SERVCMD)
	blockConnecting := user.From.InterceptWith(func(code P.MessageCode) bool {
		return !P.IsConnectingMessage(code)
	})
	teleports := user.From.Intercept(P.N_TELEPORT)
	crcs := user.From.Intercept(P.N_MAPCRC)
	votes := user.From.Intercept(P.N_MAPVOTE)
	names := user.From.Intercept(P.N_SWITCHNAME)

	for {
		logger := user.Logger()

		select {
		case <-userCtx.Done():
			return

		case msg := <-names.Receive():
			change := msg.Message.(P.SwitchName)
			c.NotifyNameChange(ctx, user, change.Name)
			msg.Pass()
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

			vote := msg.Message.(P.MapVote)
			if vote.Mode < 0 || vote.Mode >= int32(len(constants.MODE_NAMES)) {
				msg.Pass()
				continue
			}

			msg.Drop()

			err := c.runCommandWithTimeout(
				ctx,
				user,
				fmt.Sprintf(
					"creategame %s %s",
					constants.MODE_NAMES[vote.Mode],
					vote.Map,
				),
			)
			if err != nil {
				logger.Error().Err(err).Msg("failed to create game from vote")
			}
		case msg := <-crcs.Receive():
			msg.Pass()
			user.RestoreMessages()
			crc := msg.Message.(P.MapCRC)
			// The client does not have the map
			if crc.Crc == 0 {
				go func() {
					err := c.SendMap(ctx, user, crc.Map)
					if err != nil {
						logger.Warn().Err(err).Msg("failed to send map to client")
					}
				}()
			}
		case msg := <-teleports.Receive():
			message := msg.Message
			teleport := message.(P.Teleport)
			c.HandleTeleport(ctx, user, teleport.Source)
			msg.Pass()

		case msg := <-blockConnecting.Receive():
			// Skip messages that aren't allowed while the
			// client is connecting, otherwise the server
			// (rightfully) disconnects us. This solves a
			// race condition when switching servers.
			status := user.GetStatus()
			if status == UserStatusConnecting {
				msg.Drop()
				continue
			}
			msg.Pass()
		case msg := <-chats.Receive():
			message := msg.Message
			text := message.(P.Text).Text

			msg.Drop()

			if !strings.HasPrefix(text, "#") {
				// We do our own chat, don't pass on to the server
				c.ForwardGlobalChat(userCtx, user, text)
				continue
			}

			go c.HandleCommand(ctx, user, text[1:])
		case msg := <-serverCommands.Receive():
			message := msg.Message
			text := message.(P.ServCMD).Command
			msg.Drop()

			// Used by wc-ng to indicate client capabilities
			// https://github.com/tpoechtrager/wc-ng/commit/446ab52f5391076f61d0350ee081d97f8919b8f2#diff-4baabc2027e55478c19e775f6d37f1929167aba164f7741992e9eb740d70c4b4R1237
			if strings.HasPrefix(text, "__") {
				continue
			}

			go c.HandleCommand(ctx, user, text)
		}
	}
}

func (c *Cluster) PollToMessages(ctx context.Context, user *User) {
	userCtx := user.Ctx()
	serverInfo := user.To.Intercept(P.N_SERVINFO)

	for {
		select {
		case <-userCtx.Done():
			return
		case msg := <-serverInfo.Receive():
			msg.Pass()

			// Some functionality relies on manipulating fields in
			// the server info, so we store it whenever it's sent
			// to the user.
			info := msg.Message.(P.ServerInfo)
			user.Mutex.Lock()
			user.lastInfo = &info
			user.Mutex.Unlock()
		}
	}
}

func (c *Cluster) PollUser(ctx context.Context, user *User) {
	commands := user.Connection.ReceiveCommands()
	connect := user.ReceiveConnections()
	disconnect := user.Connection.ReceiveDisconnect()

	toServer := user.Connection.ReceivePackets()
	toClient := user.ReceiveToMessages()

	chanLock := chanlock.New()
	health := chanLock.Poll(ctx)

	logger := user.Logger()

	go func() {
		err := RecordSession(
			ctx,
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

	for {
		logger = user.Logger()
		select {
		case <-ctx.Done():
			c.NotifyClientChange(ctx, user, false)

			logger := user.Logger()
			logger.Info().Msg("user disconnected")

			user.DisconnectFromServer()
			return

		case <-health:
			continue

		case <-connect:
			user.Mutex.Lock()
			if user.Server != nil {
				instance := c.spaces.FindInstance(user.Server)
				if instance != nil {
					user.Space = instance
				}
			}
			user.Mutex.Unlock()

			logger := user.Logger()
			logger.Info().Msg("connected to server")

		case msg := <-toServer:
			data := msg.Data

			// We want to get the raw data from the user -- not the
			// deserialized messages
			user.RawFrom.Publish(io.RawPacket{
				Data:    data,
				Channel: msg.Channel,
			})

			messages, err := P.Decode(data, true)
			if err != nil {
				logger.Error().Err(err).
					Msg("client -> server (failed to decode message)")
				continue
			}

			server := user.GetServer()
			if server == nil {
				continue
			}

			processed := make([]P.Message, 0)
			for _, message := range messages {
				if !P.IsSpammyMessage(message.Type()) {
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

				newMessage, err := server.To.Process(
					ctx,
					msg.Channel,
					message,
				)
				if err != nil {
					log.Error().Err(err).Msgf("failed to process message for server")
					continue
				}

				processed = append(processed, newMessage)

			}

			packet := S.ServerPacket{
				Session:  uint32(user.Id),
				Channel:  msg.Channel,
				Messages: processed,
			}

			timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			select {
			case server.Incoming() <- packet:
				cancel()
			case <-timeoutCtx.Done():
				// TODO(cfoust): 08/08/23 recover from this
				logger.Error().
					Msg("client -> server (sending to server timed out)")
			}

			cancel()

		case msg := <-toClient:
			packet := msg.Packet
			done := msg.Error

			channel := uint8(packet.Channel)

			filtered := make([]P.Message, 0)
			processed := make([]P.Message, 0)

			for _, message := range packet.Messages {
				type_ := message.Type()
				if !P.IsSpammyMessage(type_) {
					printable := message

					switch printable.Type() {
					case P.N_SENDDEMO, P.N_SENDMAP:
						logger.Debug().
							Str("type", message.Type().String()).
							Msgf("cluster -> client %s", message.Type().String())
					default:
						logger.Debug().
							Str("type", message.Type().String()).
							Msgf("cluster -> client %s %+v", message.Type().String(), message)
					}
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

				// we don't want to save
				if newMessage.Type() == P.N_SENDDEMO {
					sendDemo := newMessage.(P.SendDemo)

					filtered = append(filtered, P.SendDemo{
						Tag:  sendDemo.Tag,
						Data: make([]byte, 0),
					})

					processed = append(processed, newMessage)
					continue
				}

				if newMessage.Type() == P.N_SENDMAP {
					filtered = append(filtered, P.SendMap{
						Map: make([]byte, 0),
					})

					processed = append(processed, newMessage)
					continue
				}

				filtered = append(filtered, newMessage)
				processed = append(processed, newMessage)
			}

			codes := make([]P.MessageCode, 0)
			for _, message := range processed {
				codes = append(codes, message.Type())
			}

			data, err := P.Encode(processed...)
			if err != nil {
				log.Error().Err(err).Msgf("failed to encode message")
				continue
			}

			filteredData, err := P.Encode(filtered...)
			if err != nil {
				log.Error().Err(err).Msgf("failed to encode message")
				continue
			}

			user.RawTo.Publish(io.RawPacket{
				Data:    filteredData,
				Channel: channel,
			})

			ack := user.Connection.Send(io.RawPacket{
				Channel: uint8(packet.Channel),
				Data:    data,
			})

			go func() {
				select {
				case result := <-ack:
					done <- result
				case <-ctx.Done():
					return
				}

			}()

		case request := <-commands:
			command := request.Command
			outChannel := request.Response

			go func() {
				err := c.runCommandWithTimeout(ctx, user, command)
				outChannel <- ingress.CommandResult{
					Err: err,
				}
			}()

		case <-disconnect:
			// This is triggered when the user changes servers,
			// too, it's not the same thing as leaving the cluster.
			c.NotifyClientChange(ctx, user, false)
			user.DisconnectFromServer()
		}
	}
}
