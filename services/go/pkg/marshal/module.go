package marshal

import (
	"context"
	"fmt"
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

	// The path of the socket
	path    string
	socket  *net.Conn
	command *exec.Cmd
	mutex   sync.Mutex
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
	if server.socket != nil {
		(*server.socket).Close()
	}

	// Remove the socket if it's there
	if _, err := os.Stat(server.path); !os.IsNotExist(err) {
		os.Remove(server.path)
	}
}

func (server *GameServer) Wait(exitChannel chan bool) {
	state, err := server.command.Process.Wait()

	defer func() {
		exitChannel <- true
	}()

	exitCode := state.ExitCode()
	if exitCode != 0 || err != nil {
		server.mutex.Lock()
		server.Status = ServerFailure
		server.mutex.Unlock()
		return
	}

	server.mutex.Lock()
	server.Status = ServerExited
	server.mutex.Unlock()
}

func (server *GameServer) Monitor(ctx context.Context) {
	tick := time.NewTicker(250 * time.Millisecond)
	exitChannel := make(chan bool)

	go server.Wait(exitChannel)

	defer server.Shutdown()

	for {
		status := server.GetStatus()

		// Check to see whether the socket is there
		if status == ServerStarting {
			conn, err := Connect(server.path)

			if err == nil {
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
	var nextPort uint16 = marshal.minPort

	if len(marshal.Servers) > 0 {
		nextPort = marshal.Servers[len(marshal.Servers)-1].Port + 1
	}

	for {
		available, err := IsPortAvailable(nextPort)
		if available {
			break
		}

		if err != nil {
			return 0, err
		}

		nextPort++
	}

	return nextPort, nil
}

func (marshal *Marshaller) NewServer(ctx context.Context) (*GameServer, error) {
	server := GameServer{}

	port, err := marshal.FindPort()
	if err != nil {
		return nil, err
	}

	path := filepath.Join("/tmp", fmt.Sprintf("qserv_%d.sock", port))

	cmd := exec.CommandContext(
		ctx,
		marshal.serverPath,
		fmt.Sprintf("-S%s", path),
		fmt.Sprintf("-j%d", port),
	)

	server.command = cmd
	server.path = path

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	go server.Monitor(ctx)

	return &server, nil
}
