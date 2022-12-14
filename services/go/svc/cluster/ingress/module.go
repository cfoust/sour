package ingress

import (
	"context"

	"github.com/cfoust/sour/pkg/enet"
	"github.com/cfoust/sour/svc/cluster/clients"
)

type ENetClient struct {
	id   uint16
	peer *enet.Peer
}

func (c *ENetClient) Send(packet clients.GamePacket) {
}

type ENetIngress struct {
	clients map[*ENetClient]struct{}
	host    *enet.Host
}

func (server *ENetIngress) Poll(ctx context.Context, host *enet.Host) {
	events := host.Service()

	for {
		select {
		case event := <-events:
			switch event.Type {
			case enet.EventTypeConnect:
				client, err := server.EmptyClient()
				if err != nil {
					log.Error().Err(err).Msg("failed to accept enet client")
				}

				client.peer = event.Peer

				logger := log.With().Uint16("clientId", client.id).Logger()
				logger.Info().Msg("client joined (desktop)")

				gamePacketChannel := make(chan clients.GamePacket, clients.CLIENT_MESSAGE_LIMIT)
				client.sendPacket = gamePacketChannel

				go func() {
					for {
						select {
						case packet := <-gamePacketChannel:
							client.peer.Send(packet.Channel, packet.Data)
							continue
						case <-ctx.Done():
							return
						}
					}
				}()

				client.server = server.manager.Servers[0]
				client.server.SendConnect(client.id)

				server.AddClient(client)
				break
			case enet.EventTypeReceive:
				peer := event.Peer

				var target *Client = nil
				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.peer == nil || peer.CPeer != client.peer.CPeer {
						continue
					}

					target = client
					break
				}
				server.clientMutex.Unlock()
				if target == nil || target.server == nil {
					break
				}

				target.server.SendData(
					target.id,
					uint32(event.ChannelID),
					event.Packet.Data,
				)

				break
			case enet.EventTypeDisconnect:
				peer := event.Peer

				var target *Client = nil
				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.peer == nil || peer.CPeer != client.peer.CPeer {
						continue
					}

					target = client
					break
				}
				server.clientMutex.Unlock()
				if target == nil {
					break
				}

				if target.server != nil {
					target.server.SendDisconnect(target.id)
				}

				server.RemoveClient(target)
				break
			}
		case <-ctx.Done():
			return
		}
	}

}
