package verse

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/config"
	gameServers "github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type SpaceInstance struct {
	utils.Session

	SpaceConfig

	id          string
	Space       *UserSpace
	PresetSpace *config.PresetSpace
	Editing     *EditingState
	Deployment  *gameServers.ServerDeployment
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
	edits := s.Deployment.GetServer().ReceiveMapEdits()
	for {
		select {
		case <-s.Ctx().Done():
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
	utils.Session

	// space id -> instance
	instances   map[string]*SpaceInstance
	verse       *Verse
	deployments *gameServers.DeploymentOrchestrator
	mutex       deadlock.RWMutex
	maps        *assets.AssetFetcher
}

func NewSpaceManager(verse *Verse, deployments *gameServers.DeploymentOrchestrator, maps *assets.AssetFetcher) *SpaceManager {
	return &SpaceManager{
		Session:     utils.NewSession(context.Background()),
		verse:       verse,
		deployments: deployments,
		instances:   make(map[string]*SpaceInstance),
		maps:        maps,
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
	if found == nil {
		return nil, fmt.Errorf("ambiguous reference")
	}

	// TODO support game maps
	return nil, fmt.Errorf("found map, but unsupported")
}

func (s *SpaceManager) FindInstance(server *gameServers.GameServer) *SpaceInstance {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, instance := range s.instances {
		if instance.Deployment.GetServer() == server {
			return instance
		}
	}

	return nil
}

func (s *SpaceManager) WatchInstance(ctx context.Context, space *SpaceInstance) {
	select {
	case <-ctx.Done():
		return
	case <-space.Ctx().Done():
		if space.Editing != nil {
			space.Editing.Checkpoint(ctx)
		}

		s.mutex.Lock()

		deleteId := ""
		for id, instance := range s.instances {
			if instance == space {
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

	config, err := space.GetConfig(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to fetch config for space")
		return nil, err
	}

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

	instance := SpaceInstance{
		Session:     utils.NewSession(ctx),
		Space:       space,
		Editing:     editing,
		SpaceConfig: *config,
	}

	instance.id = space.GetID()

	go editing.SavePeriodically(instance.Ctx())

	deployment := s.deployments.NewDeployment(context.Background(), "", true)
	migrations := deployment.ReceiveMigrations()

	go func() {
		for {
			select {
			case <-deployment.Ctx().Done():
				return
			case migration := <-migrations:
				if instance.Editing != nil {
					instance.Editing.Checkpoint(ctx)
				}

				gameServer := migration.New
				gameServer.SendCommand(fmt.Sprintf("serverdesc \"%s\"", config.Description))
				gameServer.SendCommand("publicserver 1")
				gameServer.SendCommand("emptymap")

				migration.Done()
			}
		}
	}()

	err = deployment.Start(ctx)
	if err != nil {
		return nil, err
	}

	instance.Deployment = deployment

	go s.WatchInstance(ctx, &instance)

	go instance.PollEdits(instance.Ctx())

	s.instances[space.GetID()] = &instance

	return &instance, nil
}

func (s *SpaceManager) DoExploreMode(ctx context.Context, deployment *gameServers.ServerDeployment, skipRoot string) {
	maps := s.maps.GetMaps(skipRoot)

	cycleMap := func() {
		gameServer := deployment.GetServer()

		var name string
		for {
			index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(maps))))
			map_ := maps[index.Int64()]

			gameServer.Mutex.RLock()
			currentMap := gameServer.Map
			gameServer.Mutex.RUnlock()

			name = map_.Name
			if name == "" || name == currentMap || strings.Contains(name, ".") || strings.Contains(name, " ") {
				continue
			}

			break
		}

		gameServer.SendCommand(fmt.Sprintf("changemap %s %d", name, game.MODE_FFA))
	}

	tick := time.NewTicker(3 * time.Minute)

	cycleMap()

	migrations := deployment.ReceiveMigrations()

	for {
		select {
		case <-deployment.Ctx().Done():
			return
		case migration := <-migrations:
			migration.Done()
			cycleMap()
		case <-tick.C:
			cycleMap()
			continue
		}
	}
}

func (s *SpaceManager) StartPresetSpace(ctx context.Context, presetSpace config.PresetSpace) (*SpaceInstance, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	config := presetSpace.Config
	id := config.Alias

	links := make([]Link, 0)
	for _, link := range config.Links {
		links = append(links, Link{
			ID:          link.ID,
			Destination: link.Destination,
		})
	}

	deployment := s.deployments.NewDeployment(ctx, presetSpace.Preset, true)
	migrations := deployment.ReceiveMigrations()
	logger := s.Logger()

	go func() {
		for {
			select {
			case <-deployment.Ctx().Done():
				return
			case migration := <-migrations:
				migration.New.Alias = config.Alias

				if config.Description != "" {
					migration.New.SendCommand(fmt.Sprintf("serverdesc \"%s\"", config.Description))
				} else {
					migration.New.SendCommand(fmt.Sprintf("serverdesc \"Sour [%s]\"", config.Alias))
				}

				migration.Done()
			}
		}
	}()

	err := deployment.Start(ctx)
	if err != nil {
		return nil, err
	}

	logger.Info().Msgf("started space %s", config.Alias)

	if presetSpace.ExploreMode {
		go s.DoExploreMode(ctx, deployment, presetSpace.ExploreModeSkip)
	}

	instance := SpaceInstance{
		Session:     utils.NewSession(s.Ctx()),
		Deployment:  deployment,
		PresetSpace: &presetSpace,
		SpaceConfig: SpaceConfig{
			Alias:       config.Alias,
			Description: config.Description,
			Links:       links,
			Owner:       "cluster",
			Map:         "",
		},
	}

	go s.WatchInstance(ctx, &instance)

	instance.id = id
	s.instances[id] = &instance

	return &instance, nil
}
