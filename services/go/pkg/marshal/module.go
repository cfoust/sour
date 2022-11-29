package marshal

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type ServerStatus byte

const (
	ServerStarting ServerStatus = iota
	ServerOK
	ServerFailure
	ServerExited
)

type GameServer struct {
	// The UDP port of the server
	Port   uint16
	Status ServerStatus
	Id     string

	// The path of the socket
	path    string
	socket  *net.Conn
	command *exec.Cmd
	mutex   sync.Mutex
	exit    chan bool
}

func (server *GameServer) GetStatus() ServerStatus {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	return server.Status
}

func Connect(path string) (*net.Conn, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}

	return &conn, nil
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

func (server *GameServer) Wait() {
	state, err := server.command.Process.Wait()

	defer func() {
		server.exit <- true
	}()

	exitCode := state.ExitCode()
	if exitCode != 0 || err != nil {
		server.mutex.Lock()
		server.Status = ServerFailure
		server.mutex.Unlock()
		log.Printf("[%s] failed", server.Id)
		if err != nil {
			log.Print(err)
		}
		return
	}

	server.mutex.Lock()
	server.Status = ServerExited
	server.mutex.Unlock()

	log.Printf("[%s] exited", server.Id)
}

func (server *GameServer) Monitor(ctx context.Context) {
	tick := time.NewTicker(250 * time.Millisecond)
	exitChannel := make(chan bool)

	go server.Wait()

	for {
		status := server.GetStatus()

		// Check to see whether the socket is there
		if status == ServerStarting {
			conn, err := Connect(server.path)

			if err == nil {
				log.Printf("[%s] connected", server.Id)
				server.mutex.Lock()
				server.Status = ServerOK
				status = ServerOK
				server.socket = conn
				server.mutex.Unlock()
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

type Marshaller struct {
	minPort    uint16
	maxPort    uint16
	serverPath string
	Servers    []*GameServer
	mutex      sync.Mutex
}

func NewMarshaller(serverPath string, minPort uint16, maxPort uint16) *Marshaller {
	marshal := Marshaller{}
	marshal.Servers = make([]*GameServer, 0)
	marshal.serverPath = serverPath
	marshal.minPort = minPort
	marshal.maxPort = maxPort
	return &marshal
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

func (marshal *Marshaller) FindPort() (uint16, error) {
	// Qserv uses port and port + 1
	for port := marshal.minPort; port < marshal.maxPort; port += 2 {
		occupied := false
		for _, server := range marshal.Servers {
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

func (marshal *Marshaller) Shutdown() {
	marshal.mutex.Lock()
	defer marshal.mutex.Unlock()

	for _, server := range marshal.Servers {
		server.Shutdown()
	}
}

type Identity struct {
	Hash string
	Path string
}

func FindIdentity(port uint16) Identity {
	generate := func() Identity {
		hash := fmt.Sprintf("%x", sha256.Sum256(
			[]byte(fmt.Sprintf("%d-%d", port, rand.Intn(1000))),
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

func (marshal *Marshaller) NewServer(ctx context.Context) (*GameServer, error) {
	server := GameServer{}

	// We don't want other servers to start while this one is being started
	// because of port contention
	marshal.mutex.Lock()
	defer marshal.mutex.Unlock()

	port, err := marshal.FindPort()
	if err != nil {
		return nil, err
	}

	server.Port = port

	identity := FindIdentity(port)

	server.Id = identity.Hash

	cmd := exec.CommandContext(
		ctx,
		marshal.serverPath,
		fmt.Sprintf("-S%s", identity.Path),
		"-C../server/config/server-init.cfg",
		fmt.Sprintf("-j%d", port),
	)

	server.command = cmd
	server.path = identity.Path
	server.exit = make(chan bool)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	log.Printf("[%s] started on port %d", server.Id, server.Port)

	go server.Monitor(ctx)

	marshal.Servers = append(marshal.Servers, &server)

	return &server, nil
}
