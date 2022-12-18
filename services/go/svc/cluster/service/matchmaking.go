package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/rs/zerolog/log"
)

type QueuedClient struct {
	joinTime time.Time
	client   clients.Client
}

type Matchmaker struct {
	manager    *servers.ServerManager
	clients    *clients.ClientManager
	queue      []QueuedClient
	queueEvent chan bool
	mutex      sync.Mutex
}

func NewMatchmaker(manager *servers.ServerManager, clients *clients.ClientManager) *Matchmaker {
	return &Matchmaker{
		queue:      make([]QueuedClient, 0),
		queueEvent: make(chan bool, 0),
		manager:    manager,
		clients:    clients,
	}
}

func (m *Matchmaker) Queue(client clients.Client) {
	log.Info().Uint16("client", client.Id()).Msg("queued for dueling")
	clients.SendServerMessage(client, "you are now queued for dueling")
	m.mutex.Lock()
	m.queue = append(m.queue, QueuedClient{
		client:   client,
		joinTime: time.Now(),
	})
	m.mutex.Unlock()
	m.queueEvent <- true
}

func (m *Matchmaker) Poll(ctx context.Context) {
	updateTicker := time.NewTicker(10 * time.Second)

	for {
		// Check to see if there are any matches we can arrange
		m.mutex.Lock()

		// First prune the list of any clients that are gone
		cleaned := make([]QueuedClient, 0)
		for _, queued := range m.queue {
			if queued.client.NetworkStatus() == clients.ClientNetworkStatusDisconnected {
				log.Info().Uint16("client", queued.client.Id()).Msg("pruning disconnected client")
				continue
			}
			cleaned = append(cleaned, queued)
		}
		m.queue = cleaned


		// Then look to see if we can make any matches
		matched := make(map[clients.Client]bool, 0)
		for _, queuedA := range m.queue {
			// We may have already matched this queued
			// note: can this actually occur?
			if _, ok := matched[queuedA.client]; ok {
				continue
			}

			for _, queuedB := range m.queue {
				// Same here
				if _, ok := matched[queuedB.client]; ok {
					continue
				}
				if queuedA == queuedB {
					continue
				}

				matched[queuedA.client] = true
				matched[queuedB.client] = true

				// We have a match!
				log.Info().Msg("starting duel")
				go m.Duel(ctx, queuedA.client, queuedB.client)
			}

			since := time.Now().Sub(queuedA.joinTime)
			clients.SendServerMessage(queuedA.client, fmt.Sprintf("You have been queued for %s. Say #leavequeue to leave.", since.String()))
		}

		// Remove the matches we made from the queue
		cleaned = make([]QueuedClient, 0)
		for _, queued := range m.queue {
			if _, ok := matched[queued.client]; ok {
				continue
			}
			cleaned = append(cleaned, queued)
		}
		m.queue = cleaned

		m.mutex.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-m.queueEvent:
		case <-updateTicker.C:
		}
	}
}

func (m *Matchmaker) Duel(ctx context.Context, clientA clients.Client, clientB clients.Client) {
	logger := log.With().Uint16("clientA", clientA.Id()).Uint16("clientB", clientB.Id()).Logger()

	logger.Info().Msg("initiating 1v1")

	broadcast := func(text string) {
		clients.SendServerMessage(clientA, text)
		clients.SendServerMessage(clientB, text)
	}

	failure := func() {
		broadcast(game.Red("error starting match server"))
	}

	broadcast(game.Green("Found a match!"))
	broadcast("starting match server")

	gameServer, err := m.manager.NewServer(ctx, "1v1")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
		failure()
		return
	}

	logger = logger.With().Str("server", gameServer.Reference()).Logger()

	err = gameServer.StartAndWait(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("server failed to start")
		failure()
		return
	}

	// Move the clients to the new server
	for _, client := range []clients.Client{clientA, clientB} {
		m.clients.ConnectClient(gameServer, client)
		err = m.clients.WaitUntilConnected(ctx, client)
		if err != nil {
			logger.Fatal().Err(err).Msg("client failed to connect")
			failure()
		}
	}
}
