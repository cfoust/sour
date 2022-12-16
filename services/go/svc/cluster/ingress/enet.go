package ingress

import (
	"context"
	"fmt"
	"sync"

	"github.com/cfoust/sour/pkg/enet"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/pkg/game"

	"github.com/rs/zerolog/log"
)

type ENetClient struct {
	id         uint16
	peer       *enet.Peer
	host       *enet.Host
	cancel     context.CancelFunc
	toClient   chan game.GamePacket
	toServer   chan game.GamePacket
	commands   chan clients.ClusterCommand
	disconnect chan bool
}

func NewENetClient(cancel context.CancelFunc, host *enet.Host) *ENetClient {
	return &ENetClient{
		cancel:     cancel,
		toClient:   make(chan game.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		toServer:   make(chan game.GamePacket, clients.CLIENT_MESSAGE_LIMIT),
		host:       host,
		commands:   make(chan clients.ClusterCommand, clients.CLIENT_MESSAGE_LIMIT),
		disconnect: make(chan bool, 1),
	}
}

func (c *ENetClient) Id() uint16 {
	return c.id
}

func (c *ENetClient) Host() string {
	return ""
}

func (c *ENetClient) Connect() {
}

func (c *ENetClient) Type() clients.ClientType {
	return clients.ClientTypeENet
}

func (c *ENetClient) Reference() string {
	return fmt.Sprintf("enet:%d", c.id)
}

func (c *ENetClient) SetId(id uint16) {
	c.id = id
}

func (c *ENetClient) Send(packet game.GamePacket) {
	c.toClient <- packet
}

func (c *ENetClient) ReceivePackets() <-chan game.GamePacket {
	return c.toServer
}

func (c *ENetClient) ReceiveCommands() <-chan clients.ClusterCommand {
	return c.commands
}

func (c *ENetClient) ReceiveDisconnect() <-chan bool {
	return c.disconnect
}

func (c *ENetClient) Poll(ctx context.Context) {
	for {
		select {
		case packet := <-c.toClient:
			c.peer.Send(packet.Channel, packet.Data)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (c *ENetClient) Disconnect(reason int, message string) {
	c.cancel()
	c.host.Disconnect(c.peer, enet.ID(reason))
}

type ENetIngress struct {
	// Run when a client joins
	InitialCommand string
	manager        *clients.ClientManager
	clients        map[*ENetClient]struct{}
	host           *enet.Host
	mutex          sync.Mutex
}

func NewENetIngress(manager *clients.ClientManager) *ENetIngress {
	return &ENetIngress{
		manager: manager,
		clients: make(map[*ENetClient]struct{}),
	}
}

func (server *ENetIngress) Serve(port int) error {
	host, err := enet.NewHost("", port)
	if err != nil {
		return err
	}
	server.host = host
	return nil
}

func (server *ENetIngress) FindClientForPeer(peer *enet.Peer) *ENetClient {
	var target *ENetClient = nil

	server.mutex.Lock()
	for client, _ := range server.clients {
		if client.peer == nil || peer.CPeer != client.peer.CPeer {
			continue
		}

		target = client
		break
	}
	server.mutex.Unlock()

	return target
}

func (server *ENetIngress) AddClient(s *ENetClient) {
	server.mutex.Lock()
	server.clients[s] = struct{}{}
	server.mutex.Unlock()
}

func (server *ENetIngress) RemoveClient(client *ENetClient) {
	server.mutex.Lock()
	delete(server.clients, client)
	server.mutex.Unlock()
}

func (server *ENetIngress) Poll(ctx context.Context) {
	events := server.host.Service()

	for {
		select {
		case event := <-events:
			switch event.Type {
			case enet.EventTypeConnect:
				ctx, cancel := context.WithCancel(ctx)

				client := NewENetClient(cancel, server.host)
				client.peer = event.Peer

				err := server.manager.AddClient(client)
				if err != nil {
					log.Error().Err(err).Msg("failed to accept enet client")
				}

				server.AddClient(client)

				logger := log.With().Uint16("clientId", client.id).Logger()
				logger.Info().Msg("client joined (desktop)")

				if len(server.InitialCommand) > 0 {
					client.commands <- clients.ClusterCommand{
						Command: server.InitialCommand,
					}
				}

				go client.Poll(ctx)

				// TODO
				//client.server = server.manager.Servers[0]
				//client.server.SendConnect(client.id)
				break

			case enet.EventTypeReceive:
				target := server.FindClientForPeer(event.Peer)

				if target == nil {
					continue
				}

				target.toServer <- game.GamePacket{
					Channel: event.ChannelID,
					Data:    event.Packet.Data,
				}

				break
			case enet.EventTypeDisconnect:
				target := server.FindClientForPeer(event.Peer)

				if target == nil {
					continue
				}

				server.RemoveClient(target)
				target.disconnect <- true

				server.manager.RemoveClient(target)
				break
			}
		case <-ctx.Done():
			return
		}
	}

}

func (server *ENetIngress) Shutdown() {
	server.host.Shutdown()
}
