package service

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/clients"
	"github.com/rs/zerolog/log"

	"github.com/repeale/fp-go/option"
)

type SendStatus byte

const (
	SendStatusInitialized SendStatus = iota
	SendStatusDownloading
	SendStatusMoved
)

type SendState struct {
	Status SendStatus
	Mutex  sync.Mutex
	Client *clients.Client
	Maps   *assets.MapFetcher
	Sender *MapSender
	Path   string
	Map    string
}

func (s *SendState) SetStatus(status SendStatus) {
	s.Mutex.Lock()
	s.Status = status
	s.Mutex.Unlock()
}

func (s *SendState) SendClient(data []byte, channel int) {
	s.Client.Connection.Send(game.GamePacket{
		Channel: uint8(channel),
		Data:    data,
	})
}

func (s *SendState) SendDemo() error {
	file, err := os.Open(s.Path)
	defer file.Close()
	if err != nil {
		return err
	}

	buffer, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	p := game.Packet{}
	p.Put(
		game.N_SENDDEMO,
		int(0),
		len(buffer),
	)
	p = append(p, buffer...)
	s.SendClient(p, 2)
	return nil
}

func (s *SendState) Send() error {
	client := s.Client
	logger := client.Logger()
	ctx := client.ServerSessionContext()

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// First we send a dummy map
	s.SetStatus(SendStatusDownloading)

	client.SendServerMessage("downloading map")
	p := game.Packet{}
	p.Put(
		game.N_MAPCHANGE,
		game.MapChange{
			Name:     "sending",
			Mode:     game.MODE_COOP,
			HasItems: 0,
		},
	)
	s.SendClient(p, 1)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	desktopURL := s.Maps.FindDesktopURL(s.Map)
	if opt.IsNone(desktopURL) {
		// How?
		return fmt.Errorf("could not find map URL")
	}

	log.Info().Msg(desktopURL.Value)

	mapPath := filepath.Join(s.Sender.workingDir, assets.GetURLBase(desktopURL.Value))

	s.Path = mapPath

	err := assets.DownloadFile(
		desktopURL.Value,
		mapPath,
	)
	if err != nil {
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	logger.Info().Msgf("downloaded desktop map to %s", mapPath)

	client.SendServerMessage("downloaded map")

	fakeMap, err := maps.MakeMap("getdemo 0")
	if err != nil {
		logger.Info().Err(err).Msgf("failed to make map")
		return err
	}

	p = game.Packet{}
	p.Put(game.N_SENDMAP)
	p = append(p, fakeMap...)
	s.SendClient(p, 2)

	return nil
}

type MapSender struct {
	Clients    map[*clients.Client]*SendState
	Maps       *assets.MapFetcher
	Mutex      sync.Mutex
	workingDir string
}

func NewMapSender(maps *assets.MapFetcher) *MapSender {
	return &MapSender{
		Clients: make(map[*clients.Client]*SendState),
		Maps:    maps,
	}
}

func (m *MapSender) Start() error {
	tempDir, err := ioutil.TempDir("", "maps")
	if err != nil {
		return err
	}

	m.workingDir = tempDir

	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return err
	}

	return nil
}

// Whether a map is being sent to this client.
func (m *MapSender) IsHandling(client *clients.Client) bool {
	m.Mutex.Lock()
	_, handling := m.Clients[client]
	m.Mutex.Unlock()
	return handling
}

func (m *MapSender) SendDemo(ctx context.Context, client *clients.Client) {
	m.Mutex.Lock()
	state, handling := m.Clients[client]
	m.Mutex.Unlock()

	if !handling {
		return
	}

	err := state.SendDemo()
	if err != nil {
		log.Info().Err(err).Msg("error sending demo")
	}
}

func (m *MapSender) SendMap(ctx context.Context, client *clients.Client, mapName string) {
	logger := client.Logger()
	logger.Info().Str("map", mapName).Msg("sending map")
	state := &SendState{
		Status: SendStatusInitialized,
		Client: client,
		Map:    mapName,
		Maps:   m.Maps,
		Sender: m,
	}

	m.Mutex.Lock()
	m.Clients[client] = state
	m.Mutex.Unlock()

	go state.Send()
}

func (m *MapSender) Shutdown() {
	os.RemoveAll(m.workingDir)
}
