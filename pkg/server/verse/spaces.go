package verse

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/config"
	"github.com/cfoust/sour/pkg/game/commands"
	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/gameserver"
	gameServers "github.com/cfoust/sour/pkg/server/servers"
	"github.com/cfoust/sour/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sasha-s/go-deadlock"
)

type SpaceInstance struct {
	utils.Session

	id          string
	PresetSpace *config.PresetSpace
	Server      *gameServers.GameServer

	links []config.SpaceLink
}

func (s *SpaceInstance) GetID() string {
	return s.id
}

func (s *SpaceInstance) GetLinks(ctx context.Context) ([]config.SpaceLink, error) {
	return s.links, nil
}

type SpaceManager struct {
	utils.Session

	// space id -> instance
	instances map[string]*SpaceInstance
	servers   *gameServers.ServerManager
	mutex     deadlock.RWMutex
	maps      *assets.AssetFetcher
}

func NewSpaceManager(servers *gameServers.ServerManager, maps *assets.AssetFetcher) *SpaceManager {
	return &SpaceManager{
		Session:   utils.NewSession(context.Background()),
		servers:   servers,
		instances: make(map[string]*SpaceInstance),
		maps:      maps,
	}
}

func (s *SpaceManager) Logger() zerolog.Logger {
	return log.With().Str("service", "spaces").Logger()
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

func (s *SpaceManager) WatchInstance(ctx context.Context, space *SpaceInstance) {
	select {
	case <-ctx.Done():
		return
	case <-space.Ctx().Done():
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

func (s *SpaceManager) DoExploreMode(ctx context.Context, gameServer *gameServers.GameServer, skipRoot string) {
	maps := s.maps.GetMaps(skipRoot)

	skips := make(map[*gameserver.Client]struct{})

	cycleMap := func() {
		var name string
		for {
			index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(maps))))
			map_ := maps[index.Int64()]

			gameServer.Mutex.RLock()
			currentMap := gameServer.Map
			gameServer.Mutex.RUnlock()

			name = map_.Name
			if name == "" || name == currentMap || strings.Contains(name, ".") || strings.Contains(name, " ") || map_.HasCFG {
				continue
			}

			break
		}

		gameServer.ChangeMap(C.MODE_COOP, name)
		skips = make(map[*gameserver.Client]struct{})
	}

	err := gameServer.Commands.Register(
		commands.Command{
			Name:        "skip",
			Description: "vote to skip to the next map",
			Callback: func(client *gameserver.Client) {
				if _, ok := skips[client]; ok {
					client.Message("you have already voted to skip")
					return
				}

				name := gameServer.Clients.UniqueName(client)
				gameServer.Message(fmt.Sprintf("%s voted to skip to the next map (say #skip to vote)", name))

				skips[client] = struct{}{}

				numClients := gameServer.Clients.GetNumClients()
				if len(skips) > numClients/2 || (numClients == 1 && len(skips) == 1) {
					cycleMap()
				}
			},
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("could not register explore command")
	}

	tick := time.NewTicker(3 * time.Minute)

	cycleMap()

	for {
		select {
		case <-gameServer.Session.Ctx().Done():
			return
		case <-tick.C:
			cycleMap()
			continue
		}
	}
}

func (s *SpaceManager) StartPresetSpace(ctx context.Context, presetSpace config.PresetSpace) (*SpaceInstance, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	c := presetSpace.Config
	id := c.Alias

	links := make([]config.SpaceLink, 0)
	for _, link := range c.Links {
		links = append(links, config.SpaceLink{
			Teleport:    link.Teleport,
			Teledest:    link.Teledest,
			Destination: link.Destination,
		})
	}

	logger := s.Logger()

	gameServer, err := s.servers.NewServer(ctx, presetSpace.Preset, true)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server for preset")
		return nil, err
	}

	gameServer.Alias = c.Alias

	if c.Description != "" {
		gameServer.SetDescription(c.Description)
	} else {
		gameServer.SetDescription(fmt.Sprintf("Sour [%s]", c.Alias))
	}

	logger.Info().Msgf("started space %s", c.Alias)

	if presetSpace.ExploreMode {
		go s.DoExploreMode(ctx, gameServer, presetSpace.ExploreModeSkip)
	}

	instance := SpaceInstance{
		Session:     utils.NewSession(s.Ctx()),
		Server:      gameServer,
		PresetSpace: &presetSpace,
	}

	go s.WatchInstance(ctx, &instance)

	instance.id = id
	s.instances[id] = &instance

	return &instance, nil
}
