package verse

import (
	"context"
	"fmt"

	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/config"
	gameServers "github.com/cfoust/sour/svc/cluster/servers"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type SpaceInstance struct {
	SpaceConfig

	id      string
	Space   *UserSpace
	Editing *EditingState
	Server  *gameServers.GameServer
	// Lasts for the lifetime of the instance, it's copied from the game
	// server's
	Context context.Context
}

func (s *SpaceInstance) IsOpenEdit() bool {
	if s.Editing == nil {
		return false
	}
	return s.Editing.IsOpenEdit()
}

func (s *SpaceInstance) GetID() string {
	return s.id
}

func (s *SpaceInstance) GetOwner(ctx context.Context) (string, error) {
	if s.Space != nil {
		return s.Space.GetOwner(ctx)
	}
	return s.Owner, nil
}

func (s *SpaceInstance) GetDescription(ctx context.Context) (string, error) {
	if s.Space != nil {
		return s.Space.GetDescription(ctx)
	}
	return s.Description, nil
}

func (s *SpaceInstance) GetAlias(ctx context.Context) (string, error) {
	if s.Space != nil {
		alias, err := s.Space.GetAlias(ctx)
		if err != nil {
			return "", err
		}
		s.Alias = alias
		return alias, err
	}
	return s.Alias, nil
}

func (s *SpaceInstance) GetMap(ctx context.Context) (string, error) {
	if s.Space != nil {
		map_, err := s.Space.GetMap(ctx)
		if err != nil {
			return "", err
		}
		s.Map = map_.GetID()
		return map_.GetID(), nil
	}
	return s.Map, nil
}

func (s *SpaceInstance) GetLinks(ctx context.Context) ([]Link, error) {
	if s.Space != nil {
		links, err := s.Space.GetLinks(ctx)
		if err != nil {
			return nil, err
		}
		s.Links = links
		return links, nil
	}
	return s.Links, nil
}

func (s *SpaceInstance) PollEdits(ctx context.Context) {
	edits := s.Server.ReceiveMapEdits()
	for {
		select {
		case <-s.Context.Done():
			return
		case edit := <-edits:
			if s.Editing == nil {
				continue
			}
			s.Editing.Process(edit.Client, edit.Message)
			continue
		}
	}
}

type SpaceManager struct {
	// space id -> instance
	instances map[string]*SpaceInstance
	verse     *Verse
	servers   *gameServers.ServerManager
	mutex     deadlock.RWMutex
	maps      *assets.AssetFetcher
}

func NewSpaceManager(verse *Verse, servers *gameServers.ServerManager, maps *assets.AssetFetcher) *SpaceManager {
	return &SpaceManager{
		verse:     verse,
		servers:   servers,
		instances: make(map[string]*SpaceInstance),
		maps:      maps,
	}
}

func (s *SpaceManager) Logger() zerolog.Logger {
	return log.With().Str("service", "spaces").Logger()
}

func (s *SpaceManager) SearchSpace(ctx context.Context, id string) (*UserSpace, error) {
	// Search for a user's space matching this ID
	space, _ := s.verse.FindSpace(ctx, id)
	if space != nil {
		return space, nil
	}

	// We don't care if that errored, search the maps (which are implicitly spaces)
	found := s.maps.FindMap(id)
	if opt.IsNone(found) {
		return nil, fmt.Errorf("ambiguous reference")
	}

	// TODO support game maps
	return nil, fmt.Errorf("found map, but unsupported")
}

func (s *SpaceManager) FindInstance(server *gameServers.GameServer) *SpaceInstance {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, instance := range s.instances {
		if instance.Server == server {
			return instance
		}
	}

	return nil
}

func (s *SpaceManager) WatchServer(ctx context.Context, space *SpaceInstance, server *gameServers.GameServer) {

	select {
	case <-ctx.Done():
		return
	case <-server.Context.Done():
		space.Editing.Checkpoint(ctx)

		s.mutex.Lock()

		deleteId := ""
		for id, instance := range s.instances {
			if instance.Server == server {
				deleteId = id
			}
		}

		if deleteId != "" {
			delete(s.instances, deleteId)
		}

		s.mutex.Unlock()
		return
	}
}

func (s *SpaceManager) StartSpace(ctx context.Context, id string) (*SpaceInstance, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger := s.Logger()

	space, err := s.SearchSpace(ctx, id)
	if err != nil {
		logger.Error().Err(err).Msgf("could not find space %s", id)
		return nil, err
	}

	if instance, ok := s.instances[space.GetID()]; ok {
		return instance, nil
	}

	serverCtx := context.Background()
	gameServer, err := s.servers.NewServer(serverCtx, "", true)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server for space")
		return nil, err
	}

	err = gameServer.StartAndWait(serverCtx)
	if err != nil {
		return nil, err
	}

	desc, err := space.GetDescription(ctx)
	if err != nil {
		return nil, err
	}

	gameServer.SendCommand(fmt.Sprintf("serverdesc \"%s\"", desc))
	gameServer.SendCommand("publicserver 1")
	gameServer.SendCommand("emptymap")

	verseMap, err := space.GetMap(ctx)
	if err != nil {
		return nil, err
	}

	map_, err := verseMap.LoadGameMap(ctx)
	if err != nil {
		return nil, err
	}

	editing := NewEditingState(s.verse, space, verseMap)
	err = editing.LoadMap(map_)
	if err != nil {
		return nil, err
	}

	go editing.SavePeriodically(gameServer.Context)

	instance := SpaceInstance{
		Space:   space,
		Editing: editing,
		Server:  gameServer,
		Context: gameServer.Context,
	}

	instance.id = space.GetID()

	go s.WatchServer(ctx, &instance, gameServer)

	go instance.PollEdits(gameServer.Context)

	s.instances[space.GetID()] = &instance

	return &instance, nil
}

func (s *SpaceManager) StartPresetSpace(ctx context.Context, presetSpace config.PresetSpace) (*SpaceInstance, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger := s.Logger()

	serverCtx := context.Background()
	gameServer, err := s.servers.NewServer(serverCtx, presetSpace.Preset, true)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server for preset")
		return nil, err
	}

	err = gameServer.StartAndWait(serverCtx)
	if err != nil {
		return nil, err
	}

	config := presetSpace.Config
	id := config.Alias
	gameServer.Alias = id

	instance := SpaceInstance{
		Server:  gameServer,
		Context: gameServer.Context,
	}

	instance.id = id

	links := make([]Link, 0)
	for _, link := range config.Links {
		links = append(links, Link{
			ID:          link.ID,
			Destination: link.Destination,
		})
	}
	instance.Links = links

	if config.Description != "" {
		gameServer.SendCommand(fmt.Sprintf("serverdesc \"%s\"", config.Description))
	}

	go s.WatchServer(ctx, &instance, gameServer)

	s.instances[id] = &instance

	logger.Info().Msgf("started preset %s", id)

	return &instance, nil
}
