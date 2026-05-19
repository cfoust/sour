package servers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/config"
	C "github.com/cfoust/sour/pkg/game/constants"
	P "github.com/cfoust/sour/pkg/game/protocol"
	"github.com/cfoust/sour/pkg/gameserver"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/utils"

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
	Channel  uint8
	Messages []P.Message
	Server   *GameServer
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

	presets []config.Preset
	Maps    *assets.AssetFetcher

	serverDescription string

	kicks   chan ClientKick
	packets chan ClientPacket
}

func (manager *ServerManager) ReceivePackets() <-chan ClientPacket {
	return manager.packets
}

func (manager *ServerManager) ReceiveKicks() <-chan ClientKick {
	return manager.kicks
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

func NewServerManager(
	maps *assets.AssetFetcher,
	serverDescription string,
	presets []config.Preset,
) *ServerManager {
	return &ServerManager{
		Servers:           make([]*GameServer, 0),
		Maps:              maps,
		serverDescription: serverDescription,
		presets:           presets,
		kicks:             make(chan ClientKick, 100),
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

// TODO this is bad, server identifiers can collide
func FindIdentity() string {
	generate := func() string {
		number, _ := rand.Int(rand.Reader, big.NewInt(1000))
		bytes := sha256.Sum256([]byte(fmt.Sprintf("%d", number)))
		hash := strings.ToUpper(fmt.Sprintf("%x", bytes)[:4])
		return hash
	}

	return generate()
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

func (manager *ServerManager) handleMapChange(ctx context.Context, server *GameServer, mapName string) {
	if mapName == "" {
		return
	}

	logger := log.With().Str("map", mapName).Logger()
	data, err := manager.Maps.FetchMapBytes(ctx, mapName)
	if err != nil {
		logger.Error().Err(err).Msg("failed to download map")
		return
	}

	go manager.ReadEntities(ctx, server, data)
}

// runStepLoop drives a GameServer via Step() on a fixed ticker. It drains
// incoming packets, steps the server, routes output, and handles map changes.
func (manager *ServerManager) runStepLoop(server *GameServer, tickRate time.Duration) {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	dtMs := int64(tickRate / time.Millisecond)

	for {
		select {
		case <-server.Ctx():
			return
		case <-ticker.C:
			var batch []gameserver.InputPacket
			for {
				select {
				case pkt := <-server.incoming:
					batch = append(batch, pkt)
				default:
					goto drained
				}
			}
		drained:

			outputs := server.Server.Step(dtMs, batch)

			for _, pkt := range outputs {
				server.packets <- ClientPacket{
					Client:   ingress.ClientID(pkt.Session),
					Channel:  pkt.Channel,
					Messages: pkt.Messages,
					Server:   server,
				}
			}

			if mapName := server.Server.PendingMap(); mapName != "" {
				go manager.handleMapChange(server.Session.Ctx(), server, mapName)
			}
		}
	}
}

func (manager *ServerManager) FindPreset(presetName string, isVirtualOk bool) opt.Option[config.Preset] {
	for _, preset := range manager.presets {
		if (preset.Name == presetName || (len(presetName) == 0 && preset.Default)) && (isVirtualOk || !preset.Virtual) {
			return opt.Some(preset)
		}
	}

	return opt.None[config.Preset]()
}

func (manager *ServerManager) NewServer(ctx context.Context, presetName string, isVirtualOk bool) (*GameServer, error) {
	found := manager.FindPreset(presetName, isVirtualOk)

	if opt.IsNone(found) {
		return nil, fmt.Errorf("failed to find server preset %s and there is no default", presetName)
	}

	config := found.Value.Config

	server := GameServer{
		Server:    gameserver.New(&config),
		Session:   utils.NewSession(ctx),
		Alias:     "",
		LastEvent: time.Now(),
		Entities:  make([]maps.Entity, 0),
		Hidden:    false,
		Started:   time.Now(),
		kicks:     manager.kicks,
		packets:   manager.packets,
		Id:        FindIdentity(),
		incoming:  make(chan gameserver.InputPacket, 256),

		From: P.NewMessageProxy(false),
		To:   P.NewMessageProxy(true),
	}

	server.SetDescription(
		strings.ReplaceAll(manager.serverDescription, "#id", server.Id),
	)

	mode := C.GetModeNumber(config.DefaultMode)
	if opt.IsNone(mode) {
		return nil, fmt.Errorf("invalid mode name: %s", config.DefaultMode)
	}

	server.ChangeMap(int32(mode.Value), config.DefaultMap)

	go manager.runStepLoop(&server, time.Duration(config.GetTickRateMs())*time.Millisecond)

	manager.Servers = append(manager.Servers, &server)

	// Remove the server when it exits for any reason
	go func() {
		<-server.Ctx()
		manager.RemoveServer(&server)
	}()

	return &server, nil
}
