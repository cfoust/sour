package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/messages"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/rs/zerolog/log"
)

type QueuedClient struct {
	joinTime time.Time
	client   *clients.Client
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

func (m *Matchmaker) Queue(client *clients.Client) {
	m.mutex.Lock()
	for _, queued := range m.queue {
		if queued.client == client {
			return
		}
	}
	m.mutex.Unlock()

	log.Info().Uint16("client", client.Id).Msg("queued for dueling")
	m.mutex.Lock()
	m.queue = append(m.queue, QueuedClient{
		client:   client,
		joinTime: time.Now(),
	})
	m.mutex.Unlock()
	m.queueEvent <- true
	client.SendServerMessage("you are now queued for dueling")
}

func (m *Matchmaker) Dequeue(client *clients.Client) {
	log.Info().Uint16("client", client.Id).Msg("left duel queue")
	m.mutex.Lock()
	cleaned := make([]QueuedClient, 0)
	for _, queued := range m.queue {
		if queued.client == client {
			continue
		}
		cleaned = append(cleaned, queued)
	}
	m.queue = cleaned
	m.mutex.Unlock()
	client.SendServerMessage("you are no longer queued")
}

func (m *Matchmaker) Poll(ctx context.Context) {
	updateTicker := time.NewTicker(10 * time.Second)

	for {
		// Check to see if there are any matches we can arrange
		m.mutex.Lock()

		// First prune the list of any clients that are gone
		cleaned := make([]QueuedClient, 0)
		for _, queued := range m.queue {
			if queued.client.Connection.NetworkStatus() == clients.ClientNetworkStatusDisconnected {
				log.Info().Uint16("client", queued.client.Id).Msg("pruning disconnected client")
				continue
			}
			cleaned = append(cleaned, queued)
		}
		m.queue = cleaned

		// Then look to see if we can make any matches
		matched := make(map[*clients.Client]bool, 0)
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
				go m.Duel(ctx, queuedA.client, queuedB.client)
			}
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

// Do a period of uninterrupted gameplay, like the warmup or main "struggle" sections.
func (m *Matchmaker) DoSession(ctx context.Context, numSeconds int, announce bool, message func(string)) {
	tick := time.NewTicker(50 * time.Millisecond)

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(numSeconds) * time.Second)

	sessionCtx, cancelSession := context.WithDeadline(ctx, endTime)
	defer cancelSession()

	announceThresholds := []int{
		120,
		60,
		30,
		15,
		10,
		9,
		8,
		7,
		6,
		5,
		4,
		3,
		2,
		1,
	}

	announceIndex := 0

	for i, announce := range announceThresholds {
		if numSeconds > announce {
			break
		}

		announceIndex = i
	}

	for {
		select {
		case <-tick.C:
			remaining := int(endTime.Sub(time.Now()))
			if announceIndex < len(announceThresholds) && remaining <= announceThresholds[announceIndex] {
				message(fmt.Sprintf("%d seconds remaining", announceThresholds[announceIndex]))
				announceIndex++
			}
		case <-ctx.Done():
			cancelSession()
			return
		case <-sessionCtx.Done():
			return
		}
	}
}

func (m *Matchmaker) DoCountdown(ctx context.Context, seconds int, message func(string)) {
	tick := time.NewTicker(1 * time.Second)
	count := seconds

	for {
		select {
		case <-tick.C:
			if count == 0 {
				return
			}
			message(fmt.Sprintf("%d", count))
			count--
		case <-ctx.Done():
			log.Info().Msg("countdown context canceled")
			return
		}
	}
}

func abs(x, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func (m *Matchmaker) Duel(ctx context.Context, clientA *clients.Client, clientB *clients.Client) {
	logger := log.With().Uint16("clientA", clientA.Id).Uint16("clientB", clientB.Id).Logger()

	logger.Info().Msg("initiating 1v1")

	matchContext, cancelMatch := context.WithCancel(ctx)
	defer cancelMatch()

	// If any client disconnects from the CLUSTER, end the match
	for _, client := range []*clients.Client{clientA, clientB} {
		go func(client *clients.Client) {
			select {
			case <-matchContext.Done():
				return
			case <-client.Connection.SessionContext().Done():
				logger.Info().Msgf("client %d disconnected from cluster, ending match", client.Id)
				cancelMatch()
				return
			}
		}(client)
	}

	broadcast := func(text string) {
		clientA.SendServerMessage(text)
		clientB.SendServerMessage(text)
	}

	failure := func() {
		broadcast(game.Red("error starting match server"))
	}

	broadcast(game.Green("Found a match!"))
	broadcast("starting match server")

	gameServer, err := m.manager.NewServer(ctx, "1v1")
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		failure()
		return
	}

	go func() {
		select {
		case <-matchContext.Done():
			m.manager.RemoveServer(gameServer)
			return
		}
	}()

	logger = logger.With().Str("server", gameServer.Reference()).Logger()

	err = gameServer.StartAndWait(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("server failed to start")
		failure()
		return
	}

	gameServer.SendCommand("pausegame 1")
	gameServer.SendCommand(fmt.Sprintf("serverdesc \"Sour %s\"", game.Red("DUEL")))

	if matchContext.Err() != nil {
		return
	}

	// Move the clients to the new server
	for _, client := range []*clients.Client{clientA, clientB} {
		// Store previous server
		oldServer := client.GetServer()

		go func(client *clients.Client, oldServer *servers.GameServer) {
			select {
			case <-matchContext.Done():
				// When the match is done (regardless of result) attempt to move
				client.ConnectToServer(oldServer)
				return
			case <-client.ServerSessionContext().Done():
				logger.Info().Msgf("client %d disconnected from server, ending match", client.Id)
				// If any client disconnects from the SERVER, end the match
				cancelMatch()
				return
			}
		}(client, oldServer)

		connected, err := client.ConnectToServer(gameServer)
		result := <-connected
		if result == false || err != nil {
			logger.Error().Err(err).Msg("client failed to connect")
			failure()
			return
		}

	}

	if matchContext.Err() != nil {
		return
	}

	gameServer.SendCommand("pausegame 0")
	broadcast("Duel: You must win by at least three frags. You are respawned automatically. Disconnecting counts as a loss.")

	// Start with a warmup
	broadcast(game.Blue("WARMUP"))
	m.DoSession(matchContext, 30, true, broadcast)
	gameServer.SendCommand("resetplayers 1")
	gameServer.SendCommand("forcerespawn")

	if matchContext.Err() != nil {
		return
	}

	broadcasts := gameServer.BroadcastSubscribe()
	defer gameServer.BroadcastUnsubscribe(broadcasts)

	scoreA := 0
	scoreB := 0
	var scoreMutex sync.Mutex

	go func() {
		for {
			select {
			case msg := <-broadcasts:
				if msg.Type() == game.N_DIED {
					died := msg.Contents().(*messages.Died)

					if died.Client == died.Killer {
						continue
					}

					scoreMutex.Lock()
					// should be A?
					if died.Client == 0 {
						logger.Info().Err(err).Msg("client B killed A")
						scoreB = died.Frags
					} else if died.Client == 1 {
						logger.Info().Err(err).Msg("client A killed B")
						scoreA = died.Frags
					}
					scoreMutex.Unlock()

					gameServer.SendCommand("forcerespawn")
					gameServer.SendCommand("pausegame 1")
					m.DoCountdown(matchContext, 1, broadcast)
					gameServer.SendCommand("pausegame 0")
				}
			case <-matchContext.Done():
				return
			}
		}
	}()

	broadcast(game.Red("GET READY"))
	gameServer.SendCommand("pausegame 1")
	m.DoCountdown(matchContext, 5, broadcast)
	gameServer.SendCommand("pausegame 0")
	broadcast(game.Green("GO"))

	if matchContext.Err() != nil {
		return
	}

	m.DoSession(matchContext, 180, true, broadcast)

	if matchContext.Err() != nil {
		return
	}

	// You have to win by three points from where overtime started
	for {
		scoreMutex.Lock()
		overtimeA := scoreA
		overtimeB := scoreB
		scoreMutex.Unlock()

		if abs(overtimeA, overtimeB) >= 3 {
			break
		}

		broadcast(game.Red("OVERTIME"))
		gameServer.SendCommand("resetplayers 0")

		gameServer.SendCommand("pausegame 1")
		m.DoCountdown(matchContext, 5, broadcast)
		gameServer.SendCommand("pausegame 0")

		broadcast(game.Red("GO"))
		m.DoSession(matchContext, 60, true, broadcast)

		if matchContext.Err() != nil {
			return
		}
	}

	logger.Info().Msgf("match ended %d:%d", scoreA, scoreB)
}
