package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/messages"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

type DuelResult struct {
	Winner       *clients.Client
	Loser        *clients.Client
	Type         string
	IsDraw       bool
	Disconnected bool
}

type DuelQueue struct {
	Client *clients.Client
	Type   string
}

type QueuedClient struct {
	JoinTime time.Time
	Client   *clients.Client
	Type     string
	// Valid for the duration of the client being in the queue
	Context context.Context
	Cancel  context.CancelFunc
}

type Matchmaker struct {
	duelTypes  []config.DuelType
	manager    *servers.ServerManager
	clients    *clients.ClientManager
	queue      []*QueuedClient
	queueEvent chan bool
	results    chan DuelResult
	queues     chan DuelQueue
	mutex      sync.Mutex
}

func NewMatchmaker(manager *servers.ServerManager, clients *clients.ClientManager, duelTypes []config.DuelType) *Matchmaker {
	return &Matchmaker{
		duelTypes:  duelTypes,
		queue:      make([]*QueuedClient, 0),
		queueEvent: make(chan bool, 0),
		results:    make(chan DuelResult, 10),
		queues:     make(chan DuelQueue, 10),
		manager:    manager,
		clients:    clients,
	}
}

func (m *Matchmaker) ReceiveResults() <-chan DuelResult {
	return m.results
}

func (m *Matchmaker) ReceiveQueues() <-chan DuelQueue {
	return m.queues
}

func (m *Matchmaker) FindDuelType(name string) opt.Option[config.DuelType] {
	for _, duelType := range m.duelTypes {
		if duelType.Name == name || (len(name) == 0 && duelType.Default) {
			return opt.Some[config.DuelType](duelType)
		}
	}

	return opt.None[config.DuelType]()
}

// Inform the client regularly as to how long they've been in the queue.
func (m *Matchmaker) NotifyProgress(queued *QueuedClient) {
	tick := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-tick.C:
			since := time.Now().Sub(queued.JoinTime).Round(time.Second)
			queued.Client.SendServerMessage(fmt.Sprintf("you have been queued for %s for %s", queued.Type, since))
		case <-queued.Context.Done():
			return
		}
	}
}

func (m *Matchmaker) Queue(client *clients.Client, typeName string) error {
	duelType := m.FindDuelType(typeName)

	if opt.IsNone(duelType) {
		return fmt.Errorf("failed to find duel type")
	}

	m.mutex.Lock()
	for _, queued := range m.queue {
		if queued.Client == client && queued.Type == typeName {
			client.SendServerMessage(fmt.Sprintf("you are already in the queue for %s", typeName))
			return nil
		}
	}
	m.mutex.Unlock()

	m.mutex.Lock()
	context, cancel := context.WithCancel(client.Connection.SessionContext())
	queued := QueuedClient{
		Type:     duelType.Value.Name,
		Context:  context,
		Cancel:   cancel,
		Client:   client,
		JoinTime: time.Now(),
	}
	go m.NotifyProgress(&queued)
	m.queue = append(m.queue, &queued)
	m.mutex.Unlock()
	m.queueEvent <- true
	log.Info().Uint16("client", client.Id).Str("type", queued.Type).Msg("queued for dueling")
	client.SendServerMessage(fmt.Sprintf("you are now in the queue for %s", queued.Type))

	m.queues <- DuelQueue{
		Client: client,
		Type:   duelType.Value.Name,
	}

	return nil
}

func (m *Matchmaker) Dequeue(client *clients.Client) {
	m.mutex.Lock()
	cleaned := make([]*QueuedClient, 0)
	for _, queued := range m.queue {
		if queued.Client == client {
			log.Info().Uint16("client", client.Id).Str("type", queued.Type).Msg("left duel queue")
			client.SendServerMessage(fmt.Sprintf("you left the queue for %s", queued.Type))
			queued.Cancel()
			continue
		}
		cleaned = append(cleaned, queued)
	}
	m.queue = cleaned
	m.mutex.Unlock()
}

func (m *Matchmaker) Poll(ctx context.Context) {
	updateTicker := time.NewTicker(10 * time.Second)

	for {
		// Check to see if there are any matches we can arrange
		m.mutex.Lock()

		// First prune the list of any clients that are gone
		cleaned := make([]*QueuedClient, 0)
		for _, queued := range m.queue {
			if queued.Client.Connection.NetworkStatus() == clients.ClientNetworkStatusDisconnected {
				log.Info().Uint16("client", queued.Client.Id).Msg("pruning disconnected client")
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
			if _, ok := matched[queuedA.Client]; ok {
				continue
			}

			for _, queuedB := range m.queue {
				// Same here
				if _, ok := matched[queuedB.Client]; ok {
					continue
				}
				if queuedA.Client == queuedB.Client || queuedA.Type != queuedB.Type {
					continue
				}

				queuedA.Cancel()
				queuedB.Cancel()

				matched[queuedA.Client] = true
				matched[queuedB.Client] = true

				// We have a match!
				go m.Duel(ctx, queuedA.Client, queuedB.Client, queuedA.Type)
			}
		}

		// Remove the matches we made from the queue
		cleaned = make([]*QueuedClient, 0)
		for _, queued := range m.queue {
			if _, ok := matched[queued.Client]; ok {
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
func (m *Matchmaker) DoSession(ctx context.Context, numSeconds uint, title string, message func(string)) {
	tick := time.NewTicker(50 * time.Millisecond)

	startTime := time.Now()
	endTime := startTime.Add(time.Duration(numSeconds) * time.Second)

	sessionCtx, cancelSession := context.WithDeadline(ctx, endTime)
	defer cancelSession()

	announceThresholds := []uint{
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
			remaining := uint(endTime.Sub(time.Now()).Round(time.Second) / time.Second)
			if announceIndex < len(announceThresholds) && remaining <= announceThresholds[announceIndex] {
				message(fmt.Sprintf("%s %d seconds remaining", title, announceThresholds[announceIndex]))
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

func (m *Matchmaker) Duel(ctx context.Context, clientA *clients.Client, clientB *clients.Client, typeName string) {
	logger := log.With().Uint16("clientA", clientA.Id).Uint16("clientB", clientB.Id).Logger()

	found := m.FindDuelType(typeName)

	if opt.IsNone(found) {
		log.Fatal().Msgf("a duel was started with nonexistent duel type %s", typeName)
		return
	}

	duelType := found.Value

	logger.Info().Msg("initiating 1v1")

	matchContext, cancelMatch := context.WithCancel(ctx)
	defer cancelMatch()

	// If client is client A, B is the winner, and vice versa.
	getLeaveWinner := func(client *clients.Client) DuelResult {
		result := DuelResult{
			Type:         typeName,
			IsDraw:       false,
			Disconnected: true,
		}
		if client == clientA {
			result.Winner = clientB
			result.Loser = clientA
			return result
		}

		result.Winner = clientA
		result.Loser = clientB
		return result
	}

	outResult := make(chan DuelResult, 1)

	// Take the first result we get (one disconnect could trigger multiple)
	go func() {
		result := <-outResult
		m.results <- result
	}()

	// If any client disconnects from the CLUSTER, end the match
	for _, client := range []*clients.Client{clientA, clientB} {
		go func(client *clients.Client) {
			select {
			case <-matchContext.Done():
				return
			case <-client.Connection.SessionContext().Done():
				outResult <- getLeaveWinner(client)
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

	gameServer, err := m.manager.NewServer(ctx, duelType.Preset, true)
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
	gameServer.SendCommand(fmt.Sprintf("serverdesc \"Sour %s\"", game.Red("Duel")))

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
				outResult <- getLeaveWinner(client)
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
	broadcast(fmt.Sprintf("Duel: You must win by at least %d frags. You are respawned automatically. Disconnecting counts as a loss.", duelType.WinThreshold))

	// Start with a warmup
	broadcast(game.Blue("Warmup"))
	m.DoSession(matchContext, duelType.WarmupSeconds, game.Blue("Warmup"), broadcast)
	gameServer.SendCommand("resetplayers 1")
	gameServer.SendCommand("forcerespawn -1")

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

					killed := clientB

					clientA.Mutex.Lock()
					if int(clientB.ClientNum) == died.Killer {
						killed = clientA
					}
					clientA.Mutex.Unlock()

					scoreMutex.Lock()
					// should be A?
					if killed == clientA {
						logger.Info().Err(err).Msg("client B killed A")
						scoreB = died.Frags
					} else if killed == clientB {
						logger.Info().Err(err).Msg("client A killed B")
						scoreA = died.Frags
					}
					scoreMutex.Unlock()

					if duelType.ForceRespawn == config.RespawnTypeAll {
						gameServer.SendCommand("forcerespawn -1")
					} else if duelType.ForceRespawn == config.RespawnTypeDead {
						gameServer.SendCommand(fmt.Sprintf("forcerespawn %d", killed.ClientNum))
					}

					if duelType.PauseOnDeath {
						gameServer.SendCommand("pausegame 1")
						m.DoCountdown(matchContext, 1, broadcast)
						gameServer.SendCommand("pausegame 0")
					}
				}
			case <-matchContext.Done():
				return
			}
		}
	}()

	broadcast(game.Red("Get ready!"))
	gameServer.SendCommand("pausegame 1")
	m.DoCountdown(matchContext, 5, broadcast)
	gameServer.SendCommand("pausegame 0")
	broadcast(game.Green("GO!"))

	if matchContext.Err() != nil {
		return
	}

	m.DoSession(matchContext, duelType.GameSeconds, game.Red("Duel"), broadcast)

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

		broadcast(game.Red("Overtime"))
		gameServer.SendCommand("resetplayers 0")

		gameServer.SendCommand("pausegame 1")
		m.DoCountdown(matchContext, 5, broadcast)
		gameServer.SendCommand("pausegame 0")

		broadcast(game.Red("GO!"))
		m.DoSession(matchContext, duelType.OvertimeSeconds, game.Red("Overtime"), broadcast)

		if matchContext.Err() != nil {
			return
		}
	}

	logger.Info().Msgf("match ended %d:%d", scoreA, scoreB)

	result := DuelResult{
		Type:   typeName,
		Winner: clientA,
		Loser:  clientB,
		IsDraw: false,
	}
	if scoreA == scoreB {
		result.IsDraw = true
	} else if scoreB > scoreA {
		result.Winner = clientB
		result.Loser = clientA
	}

	m.results <- result
}
