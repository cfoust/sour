package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/ingress"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Client struct {
	id   uint16
	host string

	server     *servers.GameServer
	sendPacket chan clients.GamePacket
	closeSlow  func()
}

const (
	CREATE_SERVER_COOLDOWN = time.Duration(10 * time.Second)
)

type Cluster struct {
	clientMutex sync.Mutex
	clients     map[*Client]struct{}

	createMutex sync.Mutex
	// host -> time a client from that host last created a server. We
	// REALLY don't want clients to be able to DDOS us
	lastCreate map[string]time.Time
	// host -> the server created by that host
	// each host can only have one server at once
	hostServers map[string]*servers.GameServer

	manager       *servers.ServerManager
	serverCtx     context.Context
	serverMessage chan []byte

	clientManager *clients.ClientManager
}

func NewCluster(ctx context.Context, serverPath string) *Cluster {
	server := &Cluster{
		serverCtx:     ctx,
		hostServers:   make(map[string]*servers.GameServer),
		lastCreate:    make(map[string]time.Time),
		clients:       make(map[*Client]struct{}),
		clientManager: clients.NewClientManager(),
		serverMessage: make(chan []byte, 1),
		manager: servers.NewServerManager(
			serverPath,
			50000,
			51000,
		),
	}

	return server
}


func (server *Cluster) StartPresetServer(ctx context.Context) (*servers.GameServer, error) {
	// Default in development
	configPath := "../server/config/server-init.cfg"

	if envPath, ok := os.LookupEnv("QSERV_LOBBY_CONFIG"); ok {
		configPath = envPath
	}

	gameServer, err := server.manager.NewServer(ctx, configPath)

	return gameServer, err
}

func (server *Cluster) StartServers(ctx context.Context) {
	gameServer, err := server.StartPresetServer(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	gameServer.Alias = "lobby"

	go gameServer.Start(ctx, server.serverMessage)
	go server.manager.PruneServers(ctx)
}

func (server *Cluster) PollMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-server.serverMessage:
			p := game.Packet(msg)

			for len(p) > 0 {
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

				server.clientMutex.Lock()
				for client, _ := range server.clients {
					if client.id != uint16(id) {
						continue
					}

					packet := clients.GamePacket{
						Channel: uint8(chan_),
						Data:    data,
					}

					client.sendPacket <- packet

					break
				}
				server.clientMutex.Unlock()
			}
		}
	}
}

func (server *Cluster) MoveClient(ctx context.Context, client *Client, targetServer *servers.GameServer) error {
	if targetServer.Status != servers.ServerOK {
		return errors.New("Server is not available")
	}

	if targetServer == client.server {
		return nil
	}

	log.Info().Msgf("swapping from %s to %s", client.server.Id, targetServer.Id)

	// We have 'em!
	client.server.SendDisconnect(client.id)
	targetServer.SendConnect(client.id)
	client.server = targetServer

	return nil
}

func (server *Cluster) RunCommand(ctx context.Context, client *Client, command string) (string, error) {
	logger := log.With().Uint16("clientId", client.id).Logger()
	logger.Info().Msgf("running sour command '%s'", command)

	args := strings.Split(command, " ")

	if len(args) == 0 {
		return "", errors.New("invalid command")
	}

	switch args[0] {
	case "creategame":

		server.createMutex.Lock()
		defer server.createMutex.Unlock()

		lastCreate, hasLastCreate := server.lastCreate[client.host]
		if hasLastCreate && (time.Now().Sub(lastCreate)) < CREATE_SERVER_COOLDOWN {
			return "", errors.New("too soon since last server create")
		}

		existingServer, hasExistingServer := server.hostServers[client.host]
		if hasExistingServer {
			server.manager.RemoveServer(existingServer)
		}

		logger.Info().Msg("starting server")

		gameServer, err := server.StartPresetServer(server.serverCtx)
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to create server")
			return "", errors.New("failed to create server")
		}

		logger = logger.With().Str("server", gameServer.Reference()).Logger()

		go gameServer.Start(server.serverCtx, server.serverMessage)

		tick := time.NewTicker(250 * time.Millisecond)
		for {
			status := gameServer.GetStatus()
			if status == servers.ServerOK {
				logger.Info().Msg("server ok")
				break
			}

			select {
			case <-ctx.Done():
				return "", errors.New("server start timed out")
			case <-tick.C:
				continue
			}
		}

		server.lastCreate[client.host] = time.Now()
		server.hostServers[client.host] = gameServer

		return gameServer.Id, nil
	}

	return "", nil
}



func (server *Cluster) AddClient(s *Client) {
	server.clientMutex.Lock()
	server.clients[s] = struct{}{}
	server.clientMutex.Unlock()
}

func (server *Cluster) RemoveClient(client *Client) {
	if client.server != nil {
		client.server.SendDisconnect(client.id)
	}

	server.clientMutex.Lock()
	delete(server.clients, client)
	server.clientMutex.Unlock()
}

func (server *Cluster) Shutdown() {
	server.manager.Shutdown()
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	serverPath := "../server/qserv"
	if envPath, ok := os.LookupEnv("QSERV_PATH"); ok {
		serverPath = envPath
	}

	clientManager := clients.NewClientManager()

	wsIngress := ingress.NewWSIngress(clientManager)

	enetIngress := ingress.NewENetIngress(clientManager)
	enetIngress.Serve(28785)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := NewCluster(ctx, serverPath)

	go enetIngress.Poll(ctx)

	go cluster.StartServers(ctx)
	go cluster.PollMessages(ctx)

	errc := make(chan error, 1)
	go func() {
		errc <- wsIngress.Serve(ctx, 29999)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, os.Kill)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	wsIngress.Shutdown(ctx)
	enetIngress.Shutdown()
	cluster.Shutdown()
}
