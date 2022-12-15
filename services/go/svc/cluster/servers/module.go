package servers

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/repeale/fp-go"

	"github.com/cfoust/sour/pkg/game"
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
)

const (
	// How long we wait before pruning an unused server
	SERVER_MAX_IDLE_TIME = time.Duration(10 * time.Minute)
)

type GameServer struct {
	// The UDP port of the server
	Port   uint16
	Status ServerStatus
	Id     string
	// Another way for the client to refer to this server
	Alias      string
	NumClients int
	LastEvent  time.Time
	Mutex   sync.Mutex

	// The path of the socket
	path    string
	socket  *net.Conn
	command *exec.Cmd
	exit    chan bool
	send    chan []byte
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

	logger.Info().Uint("port", uint(server.Port)).Msg("server started")

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

func (server *GameServer) Start(ctx context.Context, readChannel chan []byte) {
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

				go server.PollWrites(ctx)
				go server.PollReads(ctx, readChannel)
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

	minPort    uint16
	maxPort    uint16
	serverPath string
	mutex      sync.Mutex
}

func NewServerManager(serverPath string, minPort uint16, maxPort uint16) *ServerManager {
	manager := ServerManager{}
	manager.Servers = make([]*GameServer, 0)
	manager.serverPath = serverPath
	manager.minPort = minPort
	manager.maxPort = maxPort
	return &manager
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

func (manager *ServerManager) FindPort() (uint16, error) {
	// Qserv uses port and port + 1
	for port := manager.minPort; port < manager.maxPort; port += 2 {
		occupied := false
		for _, server := range manager.Servers {
			if server.Port == port {
				occupied = true
			}
		}
		if occupied {
			continue
		}

		available, err := IsPortAvailable(port)
		if available {
			return port, nil
		}

		if err != nil {
			continue
		}
	}

	return 0, errors.New("Failed to find port in range")
}

func (manager *ServerManager) Shutdown() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	for _, server := range manager.Servers {
		server.Shutdown()
	}
}

type Identity struct {
	Hash string
	Path string
}

func FindIdentity(port uint16) Identity {
	generate := func() Identity {
		number, _ := rand.Int(rand.Reader, big.NewInt(1000))
		hash := fmt.Sprintf("%x", sha256.Sum256(
			[]byte(fmt.Sprintf("%d-%d", port, number)),
		))[:8]
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

func (manager *ServerManager) NewServer(ctx context.Context, configPath string) (*GameServer, error) {
	server := GameServer{
		send:       make(chan []byte, 1),
		NumClients: 0,
		LastEvent:  time.Now(),
	}

	// We don't want other servers to start while this one is being started
	// because of port contention
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	port, err := manager.FindPort()
	if err != nil {
		return nil, err
	}

	server.Port = port

	identity := FindIdentity(port)

	server.Id = identity.Hash

	cmd := exec.CommandContext(
		ctx,
		manager.serverPath,
		fmt.Sprintf("-S%s", identity.Path),
		fmt.Sprintf("-C%s", configPath),
		fmt.Sprintf("-j%d", port),
	)

	server.command = cmd
	server.path = identity.Path
	server.exit = make(chan bool, 1)

	manager.Servers = append(manager.Servers, &server)

	return &server, nil
}
