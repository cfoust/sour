package servers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

// Sauerbraten servers assign each client a number.
// NOT the same thing as ClientID.
type ClientNum int32

const (
	// How long we wait before pruning an unused server
	SERVER_MAX_IDLE_TIME = time.Duration(10 * time.Minute)
)

type MapRequest struct {
	Map  string
	Mode int32
}

type ClientPacket struct {
	Client   ingress.ClientID
	Messages []protocol.Message
	Server   *GameServer
}

type ClientJoin struct {
	Client ingress.ClientID
	Num    ClientNum
}

type ClientKick struct {
	Client ingress.ClientID
	Reason int32
	Text   string
}

type ClientLeave struct {
	Client ingress.ClientID
	Num    ClientNum
}

type ClientName struct {
	Client ingress.ClientID
	Name   string
}

type ServerManager struct {
	Servers []*GameServer
	Receive chan []byte
	Mutex   sync.Mutex

	presets []config.ServerPreset
	Maps    *assets.AssetFetcher

	serverDescription string

	packets chan ClientPacket
}

func (manager *ServerManager) ReceivePackets() <-chan ClientPacket {
	return manager.packets
}

func (manager *ServerManager) GetServerInfo() *ServerInfo {
	info := ServerInfo{}

	manager.Mutex.Lock()
	for _, server := range manager.Servers {
		serverInfo := server.GetServerInfo()
		info.NumClients += serverInfo.NumClients
	}
	manager.Mutex.Unlock()

	return &info
}

func NewServerManager(maps *assets.AssetFetcher, serverDescription string, presets []config.ServerPreset) *ServerManager {
	return &ServerManager{
		Servers:           make([]*GameServer, 0),
		Maps:              maps,
		serverDescription: serverDescription,
		presets:           presets,
		packets:           make(chan ClientPacket, 100),
	}
}

func (manager *ServerManager) Start() error {
	return nil
}

func (manager *ServerManager) Shutdown() {
	for _, server := range manager.Servers {
		server.Shutdown()
	}
}

type Identity struct {
	Hash string
	Path string
}

func FindIdentity() Identity {
	generate := func() Identity {
		number, _ := rand.Int(rand.Reader, big.NewInt(1000))
		bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
		hash := strings.ToUpper(fmt.Sprintf("%x", bytes)[:4])
		return Identity{
			Hash: hash,
			Path: filepath.Join("/tmp", fmt.Sprintf("qserv_%s.sock", hash)),
		}
	}

	for {
		identity := generate()

		if _, err := os.Stat(identity.Path); !os.IsNotExist(err) {
			continue
		}

		return identity
	}
}

func (manager *ServerManager) RemoveServer(server *GameServer) {
	server.Shutdown()

	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	manager.Servers = fp.Filter(func(v *GameServer) bool { return v.Id != server.Id })(manager.Servers)
}

func (manager *ServerManager) PruneServers(ctx context.Context) {
	interval := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-interval.C:
			manager.Mutex.Lock()

			toPrune := make([]*GameServer, 0)

			for _, server := range manager.Servers {
				server.Mutex.RLock()
				lastEvent := server.LastEvent
				numClients := server.NumClients()
				server.Mutex.RUnlock()
				if (time.Now().Sub(lastEvent)) < SERVER_MAX_IDLE_TIME || numClients > 0 || server.Alias != "" {
					continue
				}
				toPrune = append(toPrune, server)
			}

			manager.Mutex.Unlock()

			for _, server := range toPrune {
				logger := server.Logger()
				logger.Info().Msg("server was pruned")
				server.Cancel()
			}

			continue
		case <-ctx.Done():
			return
		}
	}
}

func (manager *ServerManager) ReadEntities(ctx context.Context, server *GameServer, data []byte) error {
	map_, err := maps.BasicsFromGZ(data)
	if err != nil {
		log.Error().Err(err).Msgf("could not read map entities")
		return err
	}

	server.Mutex.Lock()
	server.Entities = map_.Entities
	server.Mutex.Unlock()
	return nil
}

func (manager *ServerManager) PollMapRequests(ctx context.Context, server *GameServer) {
	requests := server.ReceiveMapRequests()

	for {
		select {
		case request := <-requests:
			logger := log.With().Str("map", request.Map).Int32("mode", request.Mode).Logger()

			if request.Map == "" {
				server.SendMapResponse(request.Map, request.Mode, "", false)
				continue
			}

			server.SetStatus(ServerLoadingMap)
			data, err := manager.Maps.FetchMapBytes(ctx, request.Map)
			if err != nil {
				logger.Error().Err(err).Msg("failed to download map")
				server.SendMapResponse(request.Map, request.Mode, "", false)
				continue
			}

			path := filepath.Join(manager.workingDir, fmt.Sprintf("packages/base/%s.ogz", request.Map))
			err = assets.WriteBytes(data, path)
			if err != nil {
				logger.Error().Err(err).Msg("failed to download map")
				server.SendMapResponse(request.Map, request.Mode, "", false)
				continue
			}

			server.SendMapResponse(request.Map, request.Mode, path, true)

			go manager.ReadEntities(ctx, server, data)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (manager *ServerManager) FindPreset(presetName string, isVirtualOk bool) opt.Option[config.ServerPreset] {
	for _, preset := range manager.presets {
		if (preset.Name == presetName || (len(presetName) == 0 && preset.Default)) && (isVirtualOk || !preset.Virtual) {
			return opt.Some(preset)
		}
	}

	return opt.None[config.ServerPreset]()
}

func (manager *ServerManager) ComputeConfig(preset config.ServerPreset) (string, error) {
	if len(preset.Inherit) != 0 {
		found := manager.FindPreset(preset.Inherit, true)
		if opt.IsNone(found) {
			return "", fmt.Errorf("preset inherited from nonexistent preset: %s", preset.Inherit)
		}

		computed, err := manager.ComputeConfig(found.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s\n%s", computed, preset.Config), nil
	}

	return preset.Config, nil
}

func (manager *ServerManager) NewServer(ctx context.Context, presetName string, isVirtualOk bool) (*GameServer, error) {
	found := manager.FindPreset(presetName, isVirtualOk)

	if opt.IsNone(found) {
		return nil, fmt.Errorf("failed to find server preset %s and there is no default", presetName)
	}

	//preset := found.Value

	// TODO configs

	server := GameServer{
		Alias:     "",
		LastEvent: time.Now(),
		Entities:  make([]maps.Entity, 0),
		Hidden:    false,
	}

	identity := FindIdentity()

	server.Id = identity.Hash

	manager.Servers = append(manager.Servers, &server)

	// Remove the server when it exits for any reason
	go func() {
		<-server.Ctx().Done()
		manager.RemoveServer(&server)
	}()

	go manager.PollMapRequests(ctx, &server)

	return &server, nil
}
