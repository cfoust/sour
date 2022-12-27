package service

import (
	"context"
	"sync"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/clients"
	//"github.com/rs/zerolog/log"
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
	Map    string
}

func (s *SendState) SetStatus(status SendStatus) {
	s.Mutex.Lock()
	s.Status = status
	s.Mutex.Unlock()
}

func (s *SendState) SendClient(data []byte) {
	s.Client.Connection.Send(game.GamePacket{
		Channel: 1,
		Data:    data,
	})
}

func (s *SendState) Send() {
	client := s.Client
	ctx := client.ServerSessionContext()

	if ctx.Err() != nil {
		return
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
	s.SendClient(p)
}

type MapSender struct {
	Clients map[*clients.Client]*SendState
	Maps    *assets.MapFetcher
	Mutex   sync.Mutex
}

func NewMapSender(maps *assets.MapFetcher) *MapSender {
	return &MapSender{
		Clients: make(map[*clients.Client]*SendState),
		Maps:    maps,
	}
}

// Whether a map is being sent to this client.
func (m *MapSender) IsHandling(client *clients.Client) bool {
	m.Mutex.Lock()
	_, handling := m.Clients[client]
	m.Mutex.Unlock()
	return handling
}

func (m *MapSender) SendMap(ctx context.Context, client *clients.Client, mapName string) {
	logger := client.Logger()
	logger.Info().Str("map", mapName).Msg("sending map")
	state := &SendState{
		Status: SendStatusInitialized,
		Client: client,
		Map:    mapName,
		Maps:   m.Maps,
	}

	m.Mutex.Lock()
	m.Clients[client] = state
	m.Mutex.Unlock()

	go state.Send()
}
