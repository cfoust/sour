package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/mmr"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type DuelPhase byte

const (
	DuelPhaseWarmup = iota
	DuelPhaseBattle
	DuelPhaseOvertime
	DuelPhaseDone
)

type DuelResult struct {
	Winner       *User
	Loser        *User
	Type         string
	IsDraw       bool
	Disconnected bool
}

type DuelDone struct {
	Duel   *Duel
	Result DuelResult
}

type Duel struct {
	Mutex sync.Mutex
	Phase DuelPhase
	Type  config.DuelType

	A *User
	B *User

	// The servers A and B were on prior to joining the duel
	oldAServer *servers.GameServer
	oldBServer *servers.GameServer

	scoreA   int
	scoreB   int
	Manager  *servers.ServerManager
	Finished chan DuelDone
	server   *servers.GameServer
}

func (d *Duel) Logger() zerolog.Logger {
	logger := log.With().
		Str("nameA", d.A.Reference()).
		Str("nameB", d.B.Reference()).
		Logger()

	if d.server != nil {
		logger = logger.With().Str("server", d.server.Reference()).Logger()
	}

	return logger
}

func (d *Duel) broadcast(message string) {
	d.A.Message(message)
	d.B.Message(message)
}

// Do a period of uninterrupted gameplay, like the warmup or main "struggle" sections.
func (d *Duel) runPhase(ctx context.Context, numSeconds uint, title string) {
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

	d.server.BroadcastTime(int(numSeconds))
	d.server.SetDescription(fmt.Sprintf("serverdesc \"Sour %s\"", title))

	for {
		select {
		case <-tick.C:
			remaining := uint(endTime.Sub(time.Now()).Round(time.Second) / time.Second)
			if announceIndex < len(announceThresholds) && remaining <= announceThresholds[announceIndex] {
				d.broadcast(fmt.Sprintf("%s %d seconds remaining", title, announceThresholds[announceIndex]))
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

func (d *Duel) doCountdown(ctx context.Context, seconds int) {
	logger := d.Logger()
	tick := time.NewTicker(1 * time.Second)
	count := seconds

	for {
		select {
		case <-tick.C:
			if count == 0 {
				return
			}
			d.broadcast(fmt.Sprintf("%d", count))
			count--
		case <-ctx.Done():
			logger.Info().Msg("countdown context canceled")
			return
		}
	}
}

func (d *Duel) getLeaveWinner(user *User) DuelResult {
	d.Mutex.Lock()
	phase := d.Phase
	d.Mutex.Unlock()

	result := DuelResult{
		Type:         d.Type.Name,
		IsDraw:       false,
		Disconnected: true,
	}

	if phase == DuelPhaseWarmup {
		result.IsDraw = true
	}

	if user == d.A {
		result.Winner = d.B
		result.Loser = d.A
		return result
	}

	result.Winner = d.A
	result.Loser = d.B
	return result
}

func abs(x, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

func (d *Duel) finish(result DuelResult) {
	d.Finished <- DuelDone{
		Duel:   d,
		Result: result,
	}
}

func (d *Duel) setPhase(phase DuelPhase) {
	d.Mutex.Lock()
	d.Phase = phase
	d.Mutex.Unlock()
}

func (d *Duel) Respawn(ctx context.Context, user *User) {
	if d.Type.ForceRespawn == config.RespawnTypeAll {
		d.server.ForceRespawn(nil)
	} else if d.Type.ForceRespawn == config.RespawnTypeDead {
		d.server.ForceRespawn(user.ServerClient)
	}

	if d.Type.PauseOnDeath {
		d.server.Pause()
		d.doCountdown(ctx, 1)
		d.server.Resume()
	}
}

func (d *Duel) PollDeaths(ctx context.Context) {
	broadcasts := d.server.Broadcasts.Subscribe()
	defer broadcasts.Done()

	for {
		select {
		case messages := <-broadcasts.Recv():
			for _, message := range messages {
				if message.Type() != P.N_DIED {
					continue
				}
				died := message.(P.Died)

				var killed *User

				numA := int(d.A.GetClientNum())
				numB := int(d.B.GetClientNum())

				if died.Client == numA {
					killed = d.A
				} else if died.Client == numB {
					killed = d.B
				}

				d.Respawn(ctx, killed)

				d.Mutex.Lock()
				if died.Client == died.Killer {
					if killed == d.A {
						d.scoreA = died.KillerFrags
					} else if killed == d.B {
						d.scoreB = died.KillerFrags
					}
				} else {
					if killed == d.A {
						d.scoreB = died.KillerFrags
					} else if killed == d.B {
						d.scoreA = died.KillerFrags
					}
				}
				d.Mutex.Unlock()
			}
		case <-ctx.Done():
			return
		}
	}
}

// Free up resources and move clients back to their original servers
func (d *Duel) Cleanup() {
	if d.A.GetServer() == d.server && d.oldAServer != nil {
		d.A.Connect(d.oldAServer)
	}

	if d.B.GetServer() == d.server && d.oldBServer != nil {
		d.B.Connect(d.oldBServer)
	}

	d.Manager.RemoveServer(d.server)
}

func (d *Duel) MonitorClient(
	ctx context.Context,
	user *User,
	oldServer *servers.GameServer,
	cancelMatch context.CancelFunc,
	matchResult chan DuelResult,
) {
	logger := d.Logger()

	select {
	case <-ctx.Done():
		return
	case <-user.ServerSessionContext().Done():
		logger.Info().Msgf("client %d disconnected from server, ending match", user.Id)
		matchResult <- d.getLeaveWinner(user)
		cancelMatch()
		return
	}
}

func (d *Duel) Run(ctx context.Context) {
	logger := d.Logger()
	logger.Info().Str("type", d.Type.Name).Msg("initiating duel")

	matchContext, cancelMatch := context.WithCancel(ctx)
	defer cancelMatch()

	d.oldAServer = d.A.Server
	d.oldBServer = d.B.Server

	matchResult := make(chan DuelResult, 1)

	go func() {
		<-matchContext.Done()

		// Take the first result we get (one disconnect could trigger multiple)
		result := <-matchResult
		d.Cleanup()
		d.finish(result)
	}()

	// If any client disconnects from the CLUSTER, end the match
	for _, user := range []*User{d.A, d.B} {
		go func(user *User) {
			select {
			case <-matchContext.Done():
				return
			case <-user.Ctx().Done():
				logger.Info().Msgf("user %s disconnected from cluster, ending match", user.Reference())
				matchResult <- d.getLeaveWinner(user)
				cancelMatch()
				return
			}
		}(user)
	}

	failure := func() {
		d.broadcast(game.Red("error starting match server"))
		cancelMatch()
	}

	d.broadcast(game.Green("Found a match!"))
	d.broadcast("starting match server")

	gameServer, err := d.Manager.NewServer(ctx, d.Type.Preset, true)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		failure()
		return
	}

	gameServer.Hidden = true

	d.server = gameServer

	// So we get the server in the log context
	logger = d.Logger()

	go func() {
		select {
		case <-gameServer.Ctx().Done():
			cancelMatch()
		case <-matchContext.Done():
			return
		}
	}()

	logger = logger.With().Str("server", gameServer.Reference()).Logger()

	gameServer.Pause()
	gameServer.SetDescription(fmt.Sprintf("serverdesc \"Sour %s\"", game.Red("Duel")))
	// Lock down master regardless of the user's settings
	// TODO gameServer.SendCommand("publicserver 1")

	if matchContext.Err() != nil {
		return
	}

	// Move the clients to the new server
	for _, user := range []*User{d.A, d.B} {
		// Store previous server
		oldServer := user.GetServer()

		connected, err := user.ConnectToServer(gameServer, "", true, false)
		result := <-connected
		if result == false || err != nil {
			logger.Error().Err(err).Msg("client failed to connect")
			failure()
			return
		}

		go d.MonitorClient(matchContext, user, oldServer, cancelMatch, matchResult)
	}

	if matchContext.Err() != nil {
		return
	}

	d.server.Resume()
	d.broadcast(fmt.Sprintf("Duel: You must win by at least %d frags. You are respawned automatically. Disconnecting counts as a loss.", d.Type.WinThreshold))

	// Start with a warmup
	d.broadcast(game.Blue("Warmup"))
	d.broadcast("Leaving the match during the warmup does not count as a loss.")
	d.runPhase(matchContext, d.Type.WarmupSeconds, game.Blue("Warmup"))
	gameServer.ResetPlayers(true)
	gameServer.ForceRespawn(nil)

	if matchContext.Err() != nil {
		return
	}

	go d.PollDeaths(matchContext)

	d.setPhase(DuelPhaseBattle)

	d.broadcast(game.Red("Get ready!"))
	gameServer.Pause()
	d.doCountdown(matchContext, 5)
	gameServer.Resume()
	d.broadcast(game.Green("GO!"))

	if matchContext.Err() != nil {
		return
	}

	d.runPhase(matchContext, d.Type.GameSeconds, game.Red("Duel"))

	if matchContext.Err() != nil {
		return
	}

	// You have to win by three points from where overtime started
	for {
		d.Mutex.Lock()
		overtimeA := d.scoreA
		overtimeB := d.scoreB
		d.Mutex.Unlock()

		if abs(overtimeA, overtimeB) >= int(d.Type.WinThreshold) {
			break
		}

		d.setPhase(DuelPhaseOvertime)

		d.broadcast(game.Red("Overtime"))
		gameServer.ResetPlayers(false)

		gameServer.Pause()
		d.doCountdown(matchContext, 5)
		gameServer.Resume()

		d.broadcast(game.Red("GO!"))
		d.runPhase(matchContext, d.Type.OvertimeSeconds, game.Red("Overtime"))

		if matchContext.Err() != nil {
			return
		}
	}

	d.setPhase(DuelPhaseDone)

	d.Mutex.Lock()
	logger.Info().Msgf("match ended %d:%d", d.scoreA, d.scoreB)
	d.Mutex.Unlock()

	result := DuelResult{
		Type:   d.Type.Name,
		Winner: d.A,
		Loser:  d.B,
		IsDraw: false,
	}

	if d.scoreA == d.scoreB {
		result.IsDraw = true
	} else if d.scoreB > d.scoreA {
		result.Winner = d.B
		result.Loser = d.A
	}

	matchResult <- result
}

type DuelQueue struct {
	User *User
	Type string
}

type QueuedClient struct {
	JoinTime time.Time
	User     *User
	Type     string
	// Valid for the duration of the client being in the queue
	Context context.Context
	Cancel  context.CancelFunc
}

type Matchmaker struct {
	duelTypes  []config.DuelType
	manager    *servers.ServerManager
	duels      []*Duel
	queue      []*QueuedClient
	queueEvent chan bool
	results    chan DuelResult
	queues     chan DuelQueue
	mutex      sync.Mutex
}

func NewMatchmaker(manager *servers.ServerManager, duelTypes []config.DuelType) *Matchmaker {
	return &Matchmaker{
		duelTypes:  duelTypes,
		queue:      make([]*QueuedClient, 0),
		duels:      make([]*Duel, 0),
		queueEvent: make(chan bool, 0),
		results:    make(chan DuelResult, 10),
		queues:     make(chan DuelQueue, 10),
		manager:    manager,
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
			return opt.Some(duelType)
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
			queued.User.Message(fmt.Sprintf("you have been queued for %s for %s", queued.Type, since))
		case <-queued.Context.Done():
			return
		}
	}
}

func (m *Matchmaker) Queue(user *User, typeName string) error {
	duelType := m.FindDuelType(typeName)

	if opt.IsNone(duelType) {
		return fmt.Errorf("failed to find duel type")
	}

	m.mutex.Lock()
	for _, queued := range m.queue {
		if queued.User == user && queued.Type == typeName {
			user.Message(fmt.Sprintf("you are already in the queue for %s", typeName))
			return nil
		}
	}
	m.mutex.Unlock()

	m.mutex.Lock()
	context, cancel := context.WithCancel(user.Ctx())
	queued := QueuedClient{
		Type:     duelType.Value.Name,
		Context:  context,
		Cancel:   cancel,
		User:     user,
		JoinTime: time.Now(),
	}
	go m.NotifyProgress(&queued)
	m.queue = append(m.queue, &queued)
	m.mutex.Unlock()
	log.Info().Str("user", user.Reference()).Str("type", queued.Type).Msg("queued for dueling")
	user.Message(fmt.Sprintf("you are now in the queue for %s", queued.Type))

	m.queues <- DuelQueue{
		User: user,
		Type: duelType.Value.Name,
	}

	m.queueEvent <- true

	return nil
}

func (m *Matchmaker) Dequeue(user *User) {
	m.mutex.Lock()
	cleaned := make([]*QueuedClient, 0)
	for _, queued := range m.queue {
		if queued.User == user {
			log.Info().Str("user", user.Reference()).Str("type", queued.Type).Msg("left duel queue")
			user.Message(fmt.Sprintf("you left the queue for %s", queued.Type))
			queued.Cancel()
			continue
		}
		cleaned = append(cleaned, queued)
	}
	m.queue = cleaned
	m.mutex.Unlock()
}

func (m *Matchmaker) Poll(ctx context.Context) {
	finished := make(chan DuelDone)

	for {
		select {
		case <-ctx.Done():
			return
		case done := <-finished:
			m.results <- done.Result
			m.mutex.Lock()
			duels := make([]*Duel, 0)
			for _, duel := range m.duels {
				if duel == done.Duel {
					continue
				}
				duels = append(duels, duel)
			}
			m.duels = duels
			m.mutex.Unlock()
		case <-m.queueEvent:
			// Check to see if there are any matches we can arrange
			m.mutex.Lock()

			// First prune the list of any clients that are gone
			cleaned := make([]*QueuedClient, 0)
			for _, queued := range m.queue {
				if queued.User.Connection.NetworkStatus() == ingress.NetworkStatusDisconnected {
					logger := queued.User.Logger()
					logger.Info().Msg("pruning disconnected client")
					continue
				}
				cleaned = append(cleaned, queued)
			}
			m.queue = cleaned

			// Then look to see if we can make any matches
			matched := make(map[*User]bool, 0)
			for _, queuedA := range m.queue {
				// We may have already matched this queued
				// note: can this actually occur?
				if _, ok := matched[queuedA.User]; ok {
					continue
				}

				for _, queuedB := range m.queue {
					// Same here
					if _, ok := matched[queuedB.User]; ok {
						continue
					}
					if queuedA.User == queuedB.User || queuedA.Type != queuedB.Type {
						continue
					}

					queuedA.Cancel()
					queuedB.Cancel()

					matched[queuedA.User] = true
					matched[queuedB.User] = true

					duelType := m.FindDuelType(queuedA.Type)

					// This should never happen; we check on queueing
					if opt.IsNone(duelType) {
						continue
					}

					duel := Duel{
						Type:     duelType.Value,
						Phase:    DuelPhaseWarmup,
						A:        queuedA.User,
						B:        queuedB.User,
						Manager:  m.manager,
						Finished: finished,
					}

					m.duels = append(m.duels, &duel)

					go duel.Run(ctx)
				}
			}

			// Remove the matches we made from the queue
			cleaned = make([]*QueuedClient, 0)
			for _, queued := range m.queue {
				if _, ok := matched[queued.User]; ok {
					continue
				}
				cleaned = append(cleaned, queued)
			}
			m.queue = cleaned

			m.mutex.Unlock()
		}
	}
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
				winner.Message(message)
				loser.Message(message)
				continue
			}

			winner.Message(
				game.Green("you won! ") + winnerOutcome.String(),
			)
			loser.Message(
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

			server.Users.Mutex.RLock()
			for _, user := range server.Users.Users {
				if user == winner || user == loser {
					continue
				}
				user.Message(message)
			}
			server.Users.Mutex.RUnlock()
		case queue := <-queues:
			server.Users.Mutex.RLock()
			for _, client := range server.Users.Users {
				if client == queue.User {
					continue
				}

				client.Message(fmt.Sprintf(
					"%s queued for %s",
					queue.User.Reference(),
					queue.Type,
				))
			}
			server.Users.Mutex.RUnlock()
		case <-ctx.Done():
			return
		}
	}
}
