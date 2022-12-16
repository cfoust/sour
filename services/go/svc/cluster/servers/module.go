package servers

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"syscall"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/assets"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServerStatus byte

const (
	ServerStarting ServerStatus = iota
	ServerOK
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
)

const (
	SERVER_EVENT_PACKET uint32 = iota
	SERVER_EVENT_DISCONNECT
	SERVER_EVENT_REQUEST_MAP
)

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
}

type GameServer struct {
	Status ServerStatus
	Id     string
	// Another way for the client to refer to this server
	Alias      string
	NumClients int
	LastEvent  time.Time
	Mutex      sync.Mutex

	// The path of the socket
	path string
	// The working directory of the server
	wdir    string
	socket  *net.Conn
	command *exec.Cmd
	exit    chan bool
	send    chan []byte

	disconnects chan ForceDisconnect
	mapRequests chan MapRequest
	packets     chan ClientPacket
}

func (server *GameServer) ReceiveDisconnects() <-chan ForceDisconnect {
	return server.disconnects
}

func (server *GameServer) ReceiveMapRequests() <-chan MapRequest {
	return server.mapRequests
}

func (server *GameServer) ReceivePackets() <-chan ClientPacket {
	return server.packets
}

func (server *GameServer) sendMessage(data []byte) {
	p := game.Packet{}
	p.PutUint(uint32(len(data)))
	p = append(p, data...)
	server.send <- p
}

func (server *GameServer) SendData(clientId uint16, channel uint32, data []byte) {
	p := game.Packet{}
	p.PutUint(SOCKET_EVENT_RECEIVE)
	p.PutUint(uint32(clientId))
	p.PutUint(uint32(channel))
	p = append(p, data...)

	server.sendMessage(p)
}

func (server *GameServer) SendConnect(clientId uint16) {
	p := game.Packet{}
	p.PutUint(SOCKET_EVENT_CONNECT)
	p.PutUint(uint32(clientId))
	server.sendMessage(p)
}

func (server *GameServer) SendDisconnect(clientId uint16) {
	p := game.Packet{}
	p.PutUint(SOCKET_EVENT_DISCONNECT)
	p.PutUint(uint32(clientId))
	server.sendMessage(p)
}

func (server *GameServer) SendCommand(command string) {
	p := game.Packet{}
	p.PutUint(SOCKET_EVENT_COMMAND)
	p.PutString(command)
	server.sendMessage(p)
}

func (server *GameServer) SendMapResponse(mapName string, mode int32, succeeded int32) {
	p := game.Packet{}
	p.PutUint(SOCKET_EVENT_RESPOND_MAP)
	p.PutString(mapName)
	p.PutInt(mode)
	p.PutInt(succeeded)
	server.sendMessage(p)
}

func (server *GameServer) GetStatus() ServerStatus {
	server.Mutex.Lock()
	defer server.Mutex.Unlock()
	return server.Status
}

// Whether this string is a reference to this server (either an alias or an id).
func (server *GameServer) IsReference(reference string) bool {
	return server.Id == reference || server.Alias == reference
}

func (server *GameServer) Reference() string {
	if server.Alias != "" {
		return fmt.Sprintf("%s-%s", server.Alias, server.Id)
	}
	return server.Id
}

func Connect(path string) (*net.Conn, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}

	return &conn, nil
}

func (server *GameServer) Log() zerolog.Logger {
	return log.With().Str("server", server.Reference()).Logger()
}

func (server *GameServer) Shutdown() {
	status := server.GetStatus()

	if status == ServerOK {
		server.command.Process.Kill()
	}

	if server.socket != nil {
		(*server.socket).Close()
	}

	// Remove the socket if it's there
	if _, err := os.Stat(server.path); !os.IsNotExist(err) {
		os.Remove(server.path)
	}
}

func (server *GameServer) PollWrites(ctx context.Context) {
	for {
		select {
		case msg := <-server.send:
			if server.socket != nil {
				(*server.socket).Write(msg)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (server *GameServer) PollReads(ctx context.Context, out chan []byte) {
	buffer := make([]byte, 5242880)
	for {
		if ctx.Err() != nil {
			return
		}

		numBytes, err := (*server.socket).Read(buffer)
		if err != nil {
			continue
		}

		if numBytes == 0 {
			continue
		}

		result := make([]byte, numBytes)
		copy(result, buffer[:numBytes])
		out <- result
	}
}

func (server *GameServer) PollEvents(ctx context.Context) {
	socketWrites := make(chan []byte, 16)

	go server.PollReads(ctx, socketWrites)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-socketWrites:
			p := game.Packet(msg)

			for len(p) > 0 {
				type_, ok := p.GetUint()
				if !ok {
					break
				}

				if type_ == SERVER_EVENT_REQUEST_MAP {
					mapName, ok := p.GetString()
					if !ok {
						break
					}

					mode, ok := p.GetInt()
					if !ok {
						break
					}

					server.mapRequests <- MapRequest{
						Map:  mapName,
						Mode: mode,
					}
					break
				}

				if type_ == SERVER_EVENT_DISCONNECT {
					id, ok := p.GetUint()
					if !ok {
						break
					}

					reason, ok := p.GetInt()
					if !ok {
						break
					}

					reasonText, ok := p.GetString()
					if !ok {
						break
					}

					server.disconnects <- ForceDisconnect{
						Client: id,
						Reason: reason,
						Text:   reasonText,
					}

				}

				numBytes, ok := p.GetUint()
				if !ok {
					break
				}
				id, ok := p.GetUint()
				if !ok {
					break
				}
				chan_, ok := p.GetUint()
				if !ok {
					break
				}

				data := p[:numBytes]
				p = p[len(data):]

				server.packets <- ClientPacket{
					Client: id,
					Packet: game.GamePacket{
						Data:    data,
						Channel: uint8(chan_),
					},
				}
			}
		}
	}
}

func (server *GameServer) Wait() {
	logger := server.Log()

	tailPipe := func(pipe io.ReadCloser, done chan bool) {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			logger.Info().Msg(scanner.Text())
		}
		done <- true
	}

	stdout, _ := server.command.StdoutPipe()
	stderr, _ := server.command.StderrPipe()

	err := server.command.Start()
	if err != nil {
		logger.Error().Err(err).Msg("failed to start server")
		return
	}

	logger.Info().Msg("server started")

	stdoutEOF := make(chan bool, 1)
	stderrEOF := make(chan bool, 1)

	go func(pipe io.ReadCloser, done chan bool) {
		scanner := bufio.NewScanner(pipe)

		for scanner.Scan() {
			message := scanner.Text()

			if strings.HasPrefix(message, "Join:") {
				server.Mutex.Lock()
				server.NumClients++
				server.LastEvent = time.Now()
				server.Mutex.Unlock()
			}

			if strings.HasPrefix(message, "Leave:") {
				server.Mutex.Lock()
				server.NumClients--
				server.LastEvent = time.Now()

				if server.NumClients < 0 {
					server.NumClients = 0
				}

				server.Mutex.Unlock()
			}

			logger.Info().Msg(message)
		}
		done <- true
	}(stdout, stdoutEOF)

	go tailPipe(stderr, stderrEOF)

	<-stdoutEOF
	<-stderrEOF

	state, err := server.command.Process.Wait()

	defer func() {
		server.exit <- true
	}()

	exitCode := state.ExitCode()
	if exitCode != 0 || err != nil {
		server.Mutex.Lock()
		server.Status = ServerFailure
		server.Mutex.Unlock()

		unixStatus := state.Sys().(syscall.WaitStatus)

		logger.Error().
			Err(err).
			Bool("continued", unixStatus.Continued()).
			Bool("coreDump", unixStatus.CoreDump()).
			Int("exitStatus", unixStatus.ExitStatus()).
			Bool("exited", unixStatus.Exited()).
			Bool("stopped", unixStatus.Stopped()).
			Str("stopSignal", unixStatus.StopSignal().String()).
			Str("signal", unixStatus.Signal().String()).
			Bool("signaled", unixStatus.Signaled()).
			Int("trapCause", unixStatus.TrapCause()).
			Msgf("[%s] exited with code %d", server.Reference(), exitCode)
		return
	}

	server.Mutex.Lock()
	server.Status = ServerExited
	server.Mutex.Unlock()

	logger.Info().Msg("exited")
}

func (server *GameServer) Start(ctx context.Context) {
	logger := server.Log()
	tick := time.NewTicker(250 * time.Millisecond)
	exitChannel := make(chan bool, 1)

	go server.Wait()

	for {
		status := server.GetStatus()

		// Check to see whether the socket is there
		if status == ServerStarting {
			conn, err := Connect(server.path)

			if err == nil {
				logger.Info().Msg("connected")
				server.Mutex.Lock()
				server.Status = ServerOK
				status = ServerOK
				server.socket = conn
				server.Mutex.Unlock()

				name := server.Id
				if len(server.Alias) > 0 {
					name = server.Alias
				}

				go server.SendCommand(fmt.Sprintf("serverdesc \"Sour [%s]\"", name))
				go server.PollWrites(ctx)
				go server.PollEvents(ctx)
			}
		}

		select {
		case <-exitChannel:
		case <-ctx.Done():
			return
		case <-tick.C:
			continue
		}
	}
}

type ServerManager struct {
	Servers []*GameServer
	Receive chan []byte

	maps       *assets.MapFetcher
	serverPath string
	mutex      sync.Mutex
	// The working directory of all of the servers
	workingDir string
}

func NewServerManager(serverPath string, maps *assets.MapFetcher) *ServerManager {
	return &ServerManager{
		Servers:    make([]*GameServer, 0),
		serverPath: serverPath,
		maps:       maps,
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
			url := manager.maps.FindMapURL(request.Map)

			if opt.IsNone(url) {
				server.SendMapResponse(request.Map, request.Mode, 0)
				continue
			}

			logger := log.With().Str("map", request.Map).Int32("mode", request.Mode).Logger()

			logger.Info().Msg("downloading map")
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

func (manager *ServerManager) NewServer(ctx context.Context, configPath string) (*GameServer, error) {
	server := GameServer{
		send:        make(chan []byte, 1),
		NumClients:  0,
		LastEvent:   time.Now(),
		disconnects: make(chan ForceDisconnect, 10),
		mapRequests: make(chan MapRequest, 10),
		packets:     make(chan ClientPacket, 10),
	}

	// We don't want other servers to start while this one is being started
	// because of port contention
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	identity := FindIdentity()

	server.Id = identity.Hash

	cmd := exec.CommandContext(
		ctx,
		manager.serverPath,
		fmt.Sprintf("-S%s", identity.Path),
		fmt.Sprintf("-C%s", configPath),
	)

	cmd.Dir = manager.workingDir

	server.command = cmd
	server.path = identity.Path
	server.exit = make(chan bool, 1)

	manager.Servers = append(manager.Servers, &server)

	go manager.PollMapRequests(ctx, &server)

	return &server, nil
}
