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
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/game/messages"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/config"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

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
	}

	return ""
}

const (
	// How long we wait before pruning an unused server
	SERVER_MAX_IDLE_TIME = time.Duration(10 * time.Minute)
)

type ForceDisconnect struct {
	Client uint32
	Reason int32
	Text   string
}

type MapRequest struct {
	Map  string
	Mode int32
}

type ClientPacket struct {
	Client uint32
	Packet game.GamePacket
	Server *GameServer
}

type ServerManager struct {
	Servers []*GameServer
	Receive chan []byte

	presets []config.ServerPreset
	maps    *assets.MapFetcher

	serverDescription string
	serverPath        string
	mutex             sync.Mutex
	// The working directory of all of the servers
	workingDir string

	connects    chan uint32
	disconnects chan ForceDisconnect
	packets     chan ClientPacket
}

func (manager *ServerManager) ReceiveDisconnects() <-chan ForceDisconnect {
	return manager.disconnects
}

func (manager *ServerManager) ReceivePackets() <-chan ClientPacket {
	return manager.packets
}

func (manager *ServerManager) ReceiveConnects() <-chan uint32 {
	return manager.connects
}

func NewServerManager(maps *assets.MapFetcher, serverDescription string, presets []config.ServerPreset) *ServerManager {
	return &ServerManager{
		Servers:           make([]*GameServer, 0),
		maps:              maps,
		serverDescription: serverDescription,
		presets:           presets,
		packets:           make(chan ClientPacket, 100),
		disconnects:       make(chan ForceDisconnect, 100),
		connects:          make(chan uint32, 100),
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
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

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

	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	manager.Servers = fp.Filter(func(v *GameServer) bool { return v.Id != server.Id })(manager.Servers)

	return nil
}

func (manager *ServerManager) PruneServers(ctx context.Context) {
	interval := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-interval.C:
			manager.mutex.Lock()

			toPrune := make([]*GameServer, 0)

			for _, server := range manager.Servers {
				if (time.Now().Sub(server.LastEvent)) < SERVER_MAX_IDLE_TIME || server.Alias != "" {
					continue
				}
				toPrune = append(toPrune, server)
			}

			manager.mutex.Unlock()

			for _, server := range toPrune {
				logger := server.Log()
				logger.Info().Msg("server was pruned")
				manager.RemoveServer(server)
			}

			continue
		case <-ctx.Done():
			return
		}
	}
}

func DownloadMap(url string, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	//Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (manager *ServerManager) PollMapRequests(ctx context.Context, server *GameServer) {
	requests := server.ReceiveMapRequests()

	for {
		select {
		case request := <-requests:
			server.SetStatus(ServerLoadingMap)
			url := manager.maps.FindMapURL(request.Map)

			if opt.IsNone(url) {
				server.SendMapResponse(request.Map, request.Mode, 0)
				continue
			}

			logger := log.With().Str("map", request.Map).Int32("mode", request.Mode).Logger()

			logger.Info().Str("url", url.Value).Msg("downloading map")
			path := filepath.Join(manager.workingDir, fmt.Sprintf("packages/base/%s.ogz", request.Map))
			err := DownloadMap(url.Value, path)
			if err != nil {
				logger.Error().Err(err).Msg("failed to download map")
				server.SendMapResponse(request.Map, request.Mode, 0)
				continue
			}

			logger.Info().Str("destination", path).Msg("downloaded map")
			server.SendMapResponse(request.Map, request.Mode, 1)
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (manager *ServerManager) FindPreset(presetName string) opt.Option[config.ServerPreset] {
	for _, preset := range manager.presets {
		if preset.Name == presetName || (len(presetName) == 0 && preset.Default) {
			return opt.Some[config.ServerPreset](preset)
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

func (manager *ServerManager) NewServer(ctx context.Context, presetName string) (*GameServer, error) {
	found := manager.FindPreset(presetName)

	if opt.IsNone(found) {
		return nil, fmt.Errorf("failed to find server preset %s and there is no default", presetName)
	}

	preset := found.Value

	resolvedConfig, err := manager.ResolveConfig(preset.Config)
	if err != nil {
		return nil, err
	}
	log.Info().Msgf("using config %s", resolvedConfig)

	server := GameServer{
		Alias:         "",
		Connecting:    make(chan bool, 1),
		LastEvent:     time.Now(),
		NumClients:    0,
		broadcasts:    make(chan messages.Message, 10),
		connects:      manager.connects,
		disconnects:   manager.disconnects,
		mapRequests:   make(chan MapRequest, 10),
		packets:       manager.packets,
		rawBroadcasts: make(chan game.GamePacket, 10),
		send:          make(chan []byte, 1),
		subscribers:   make([]chan messages.Message, 0),
	}

	// We don't want other servers to start while this one is being started
	// because of port contention
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

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
