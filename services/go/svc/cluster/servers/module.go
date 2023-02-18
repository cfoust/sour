package servers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/utils"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

// Sauerbraten servers assign each client a number.
// NOT the same thing as ClientID.
type ClientNum int32

//go:embed qserv/qserv
var QSERV_EXECUTABLE []byte

type ServerStatus byte

const (
	ServerStarting ServerStatus = iota
	ServerStarted
	ServerLoadingMap
	ServerHealthy
	ServerFailure
	ServerExited
)

// From the enum in services/server/socket/socket.h
const (
	SOCKET_EVENT_CONNECT uint32 = iota
	SOCKET_EVENT_RECEIVE
	SOCKET_EVENT_DISCONNECT
	SOCKET_EVENT_COMMAND
	SOCKET_EVENT_RESPOND_MAP
	SOCKET_EVENT_PING
	SOCKET_EVENT_SERVER_INFO_REQUEST
)

type ServerEvent uint32

const (
	// The server is sending a packet to a particular client
	SERVER_EVENT_PACKET ServerEvent = iota
	// The server is broadcasting a packet to all clients
	// We don't want to have to infer this
	SERVER_EVENT_BROADCAST
	// The server finished connecting a client
	SERVER_EVENT_CONNECT
	// The server forces a client to disconnect
	SERVER_EVENT_DISCONNECT
	// The server is requesting a map URL
	SERVER_EVENT_REQUEST_MAP
	// When the server is ready to accept connections (after the map loads)
	SERVER_EVENT_HEALTHY
	SERVER_EVENT_PONG
	// When the server becomes aware of a client's name
	SERVER_EVENT_NAME
	SERVER_EVENT_SERVER_INFO_REPLY
	SERVER_EVENT_EDIT
)

func (e ServerEvent) String() string {
	switch e {
	case SERVER_EVENT_PACKET:
		return "SERVER_EVENT_PACKET"
	case SERVER_EVENT_BROADCAST:
		return "SERVER_EVENT_BROADCAST"
	case SERVER_EVENT_CONNECT:
		return "SERVER_EVENT_CONNECT"
	case SERVER_EVENT_DISCONNECT:
		return "SERVER_EVENT_DISCONNECT"
	case SERVER_EVENT_REQUEST_MAP:
		return "SERVER_EVENT_REQUEST_MAP"
	case SERVER_EVENT_PONG:
		return "SERVER_EVENT_PONG"
	case SERVER_EVENT_NAME:
		return "SERVER_EVENT_NAME"
	case SERVER_EVENT_SERVER_INFO_REPLY:
		return "SERVER_EVENT_SERVER_INFO_REPLY"
	}

	return ""
}

const (
	// How long we wait before pruning an unused server
	SERVER_MAX_IDLE_TIME = time.Duration(10 * time.Minute)
)

type MapRequest struct {
	Map  string
	Mode int32
}

type ClientPacket struct {
	Client ingress.ClientID
	Packet game.GamePacket
	Server *GameServer
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
	serverPath        string
	// The working directory of all of the servers
	workingDir string

	connects chan ClientJoin
	names    chan ClientName
	kicks    chan ClientKick
	packets  chan ClientPacket
}

func (manager *ServerManager) ReceiveKicks() <-chan ClientKick {
	return manager.kicks
}

func (manager *ServerManager) ReceivePackets() <-chan ClientPacket {
	return manager.packets
}

func (manager *ServerManager) ReceiveConnects() <-chan ClientJoin {
	return manager.connects
}

func (manager *ServerManager) ReceiveNames() <-chan ClientName {
	return manager.names
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
		kicks:             make(chan ClientKick, 100),
		names:             make(chan ClientName, 100),
		connects:          make(chan ClientJoin, 100),
	}
}

func IsPortAvailable(port uint16) (bool, error) {
	addr := net.UDPAddr{
		Port: int(port),
		IP:   net.ParseIP("127.0.0.1"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return false, err
	}

	defer conn.Close()

	return true, nil
}

func (manager *ServerManager) Start() error {
	tempDir, err := ioutil.TempDir("", "qserv")
	if err != nil {
		return err
	}

	manager.workingDir = tempDir

	err = os.MkdirAll(filepath.Join(tempDir, "packages/base"), 0755)
	if err != nil {
		return err
	}

	qservPath := filepath.Join(tempDir, "qserv")

	// Copy the qserv executable out
	out, err := os.Create(qservPath)
	if err != nil {
		return err
	}
	defer out.Close()

	reader := bytes.NewReader(QSERV_EXECUTABLE)

	_, err = io.Copy(out, reader)
	if err != nil {
		return err
	}

	err = out.Chmod(0774)
	if err != nil {
		return err
	}

	manager.serverPath = qservPath

	return nil
}

func (manager *ServerManager) Shutdown() {
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	for _, server := range manager.Servers {
		server.Shutdown()
	}

	os.RemoveAll(manager.workingDir)
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

func (manager *ServerManager) RemoveServer(server *GameServer) error {
	server.Shutdown()

	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	manager.Servers = fp.Filter(func(v *GameServer) bool { return v.Id != server.Id })(manager.Servers)

	return nil
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
				numClients := server.NumClients
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
				manager.RemoveServer(server)
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

// Resolve a config string either to a file on the filesystem, or write one.
func (manager *ServerManager) ResolveConfig(config string) (filepath string, err error) {
	// If it exists, just resolve to that file path.
	if _, err := os.Stat(config); err == nil {
		return config, nil
	}

	temp, err := ioutil.TempFile(manager.workingDir, "server-config")
	if err != nil {
		return "", err
	}

	temp.Write([]byte(config))

	return temp.Name(), nil
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

	preset := found.Value

	config, err := manager.ComputeConfig(preset)

	resolvedConfig, err := manager.ResolveConfig(config)
	if err != nil {
		return nil, err
	}

	server := GameServer{
		Alias:         "",
		Session:       utils.NewSession(ctx),
		Connecting:    make(chan bool, 1),
		LastEvent:     time.Now(),
		ClientInfo:    make(map[ingress.ClientID]*ClientExtInfo),
		NumClients:    0,
		Entities:      make([]maps.Entity, 0),
		broadcasts:    make(chan game.Message, 10),
		connects:      manager.connects,
		kicks:         manager.kicks,
		mapRequests:   make(chan MapRequest),
		packets:       manager.packets,
		rawBroadcasts: make(chan game.GamePacket),
		pongs:         make(chan time.Time),
		rawEdits:      make(chan RawEdit),
		mapEdits:      make(chan MapEdit, 10),
		send:          make(chan []byte, 100),
		subscribers:   make([]chan game.Message, 0),
		names:         manager.names,
		Hidden:        false,
	}

	// We don't want other servers to start while this one is being started
	// because of port contention
	manager.Mutex.Lock()
	defer manager.Mutex.Unlock()

	identity := FindIdentity()

	server.Id = identity.Hash
	server.configFile = resolvedConfig

	cmd := exec.CommandContext(
		ctx,
		//"valgrind",
		//"--leak-check=full",
		manager.serverPath,
		fmt.Sprintf("-S%s", identity.Path),
		fmt.Sprintf("-C%s", server.configFile),
	)

	cmd.Dir = manager.workingDir

	server.description = manager.serverDescription
	server.command = cmd
	server.path = identity.Path
	server.exit = make(chan bool, 1)

	manager.Servers = append(manager.Servers, &server)

	go manager.PollMapRequests(ctx, &server)

	return &server, nil
}
