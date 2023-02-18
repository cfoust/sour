package servers

import (
	"context"
	"time"

	"github.com/cfoust/sour/svc/cluster/utils"

	"github.com/sasha-s/go-deadlock"
)

type DeploymentEvent struct {
	Old  *GameServer
	New  *GameServer
	done chan bool
}

type Migrator chan DeploymentEvent

func (d *DeploymentEvent) Done() {
	d.done <- true
}

// Represents the deployment of a single server.
type ServerDeployment struct {
	utils.Session
	server      *GameServer
	configurers []Migrator
	preset      string
	isVirtualOk bool

	orchestrator *DeploymentOrchestrator

	mutex deadlock.Mutex
}

func (s *ServerDeployment) GetServer() *GameServer {
	s.mutex.Lock()
	server := s.server
	s.mutex.Unlock()
	return server
}

func handleDeployment(ctx context.Context, handler chan DeploymentEvent, oldServer *GameServer, newServer *GameServer) error {
	done := make(chan bool)

	handler <- DeploymentEvent{
		Old:  oldServer,
		New:  newServer,
		done: done,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	select {
	case <-done:
		return nil
	case <-timeoutCtx.Done():
		return timeoutCtx.Err()
	}
}

func (s *ServerDeployment) Configure() <-chan DeploymentEvent {
	configurer := make(chan DeploymentEvent)
	s.mutex.Lock()
	s.configurers = append(s.configurers, configurer)
	s.mutex.Unlock()
	return configurer
}

func (s *ServerDeployment) startServer(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	oldServer := s.server

	server, err := s.orchestrator.servers.NewServer(
		s.Ctx(),
		s.preset,
		s.isVirtualOk,
	)

	if err != nil {
		return err
	}

	err = server.StartAndWait(ctx)
	if err != nil {
		return err
	}

	for _, configurer := range s.configurers {
		err = handleDeployment(
			ctx,
			configurer,
			oldServer,
			server,
		)
		if err != nil {
			return err
		}
	}

	// Let the cluster do anything it needs to with this server
	err = handleDeployment(
		ctx,
		s.orchestrator.migrator,
		oldServer,
		server,
	)
	if err != nil {
		return err
	}

	s.server = server

	return nil
}

func (s *ServerDeployment) watch(ctx context.Context) {
	for {
		server := s.GetServer()

		select {
		case <-ctx.Done():
			return
		case <-s.Ctx().Done():
			return
		case <-server.Ctx().Done():
			err := s.startServer(ctx)
			if err != nil {
				return
			}
		}
	}
}

func (s *ServerDeployment) Start(ctx context.Context) error {
	err := s.startServer(ctx)
	if err != nil {
		return err
	}

	go s.watch(ctx)

	return nil
}

type DeploymentOrchestrator struct {
	migrator chan DeploymentEvent
	servers  *ServerManager
}

func NewDeploymentOrchestrator(servers *ServerManager) *DeploymentOrchestrator {
	return &DeploymentOrchestrator{
		migrator: make(chan DeploymentEvent),
		servers:  servers,
	}
}

// Whenever a server fails, this gives the cluster an opportunity to do
// something with the old and new servers.
func (s *DeploymentOrchestrator) ReceiveMigrations() <-chan DeploymentEvent {
	return s.migrator
}

func (s *DeploymentOrchestrator) NewDeployment(ctx context.Context, presetName string, isVirtualOk bool) *ServerDeployment {
	return &ServerDeployment{
		Session:      utils.NewSession(ctx),
		preset:       presetName,
		isVirtualOk:  isVirtualOk,
		orchestrator: s,
		configurers:  make([]Migrator, 0),
	}
}
