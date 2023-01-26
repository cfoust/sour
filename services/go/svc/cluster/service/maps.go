package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/maps"

	"github.com/rs/zerolog/log"
)

func MakeDownloadMap(demoName string) ([]byte, error) {
	gameMap, err := maps.NewMap()
	if err != nil {
		return nil, err
	}
	gameMap.Vars["cloudlayer"] = game.StringVariable("")
	gameMap.Vars["skyboxcolour"] = game.IntVariable(0)

	// First, request the "demo" in its entirety.
	fileName := demoName[:20]
	script := fmt.Sprintf(`
can_teleport_1 = [
demodir sour
getdemo 0 %s
can_teleport_1 = []
]
can_teleport_2 = [
addzip sour/%s.dmo
demodir demo
can_teleport_2 = []
]
say a
`, fileName, fileName)

	log.Warn().Msgf("maptitle len=%d", len(script))
	gameMap.Vars["maptitle"] = game.StringVariable(script)

	gameMap.Entities = append(gameMap.Entities,
		maps.Entity{
			Type:  game.EntityTypeTeleport,
			Attr3: 1,
			Position: maps.Vector{
				X: 512 + 10,
				Y: 512 + 10,
				Z: 512,
			},
		},
		maps.Entity{
			Type:  game.EntityTypeTeleport,
			Attr3: 2,
			Position: maps.Vector{
				X: 512 - 10,
				Y: 512 - 10,
				Z: 512,
			},
		},
	)

	mapBytes, err := gameMap.EncodeOGZ()
	if err != nil {
		return mapBytes, err
	}

	return mapBytes, nil
}

type SendState struct {
	Mutex  sync.Mutex
	User *User
	Maps   *assets.AssetFetcher
	Sender *MapSender
	Path   string
	Map    string

	userAccepted  chan bool
	demoRequested chan int
}

func (s *SendState) SendClient(data []byte, channel int) <-chan bool {
	return s.User.Send(game.GamePacket{
		Channel: uint8(channel),
		Data:    data,
	})
}

func (s *SendState) SendClientSync(data []byte, channel int) error {
	if !<-s.SendClient(data, channel) {
		return fmt.Errorf("client never acknowledged message")
	}
	return nil
}

func (s *SendState) MoveClient(x float64, y float64) error {
	p := game.Packet{}
	err := p.Put(
		game.N_POS,
		uint(s.User.GetClientNum()),
		game.PhysicsState{
			LifeSequence: s.User.GetLifeSequence(),
			O: game.Vec{
				X: x,
				Y: y,
				Z: 512 + 14,
			},
		},
	)
	if err != nil {
		return err
	}
	s.SendClient(p, 0)
	return nil
}

func (s *SendState) SendPause(state bool) error {
	p := game.Packet{}
	p.Put(
		game.N_PAUSEGAME,
		state,
		s.User.GetClientNum(),
	)
	s.SendClient(p, 1)
	return nil
}

func (s *SendState) SendDemo(tag int) {
	s.demoRequested <- tag
}

func (s *SendState) TriggerSend() {
	s.userAccepted <- true
}

func (s *SendState) Send() error {
	//user := s.User
	//logger := user.Logger()
	//ctx := user.ServerSessionContext()

	//logger.Info().Msg("sending map to client")

	//if ctx.Err() != nil {
		//return ctx.Err()
	//}

	//s.SendPause(true)

	//p := game.Packet{}
	//p.Put(
		//game.N_MAPCHANGE,
		//game.MapChange{
			//Name:     "sending",
			//Mode:     int(game.MODE_COOP),
			//HasItems: 0,
		//},
	//)
	//s.SendClient(p, 1)

	//if ctx.Err() != nil {
		//return ctx.Err()
	//}

	//map_ := s.Maps.FindMap(s.Map)
	//if opt.IsNone(map_) {
		//// How?
		//return fmt.Errorf("could not find map")
	//}

	//logger = user.Logger().With().Str("map", map_.Value.Map.Name).Logger()

	//fakeMap, err := MakeDownloadMap(map_.Value.Map.Bundle)
	//if err != nil {
		//logger.Error().Err(err).Msgf("failed to make map")
		//return err
	//}

	//time.Sleep(1 * time.Second)
	//p = game.Packet{}
	//p.Put(game.N_SENDMAP)
	//p = append(p, fakeMap...)
	//err = s.SendClientSync(p, 2)
	//if err != nil {
		//return err
	//}

	//desktopURL := map_.GetDesktopURL()
	//if opt.IsNone(desktopURL) {
		//return fmt.Errorf("no desktop bundle for map %s", s.Map)
	//}

	//mapPath := filepath.Join(s.Sender.workingDir, assets.GetURLBase(desktopURL.Value))
	//s.Path = mapPath
	//err = assets.DownloadFile(
		//desktopURL.Value,
		//mapPath,
	//)
	//if err != nil {
		//return err
	//}

	//if ctx.Err() != nil {
		//return ctx.Err()
	//}

	//user.SendServerMessage("You are missing this map. Please run '/do $maptitle' to download it.")

	//select {
	//case <-s.userAccepted:
	//case <-ctx.Done():
		//return ctx.Err()
	//}

	//logger.Info().Msg("user accepted download")

	//s.User.GetServer().SendCommand(fmt.Sprintf("forcerespawn %d", s.User.GetClientNum()))
	//time.Sleep(1 * time.Second)
	//s.MoveClient(512+10, 512+10)
	//time.Sleep(1 * time.Second)
	//// so physics runs
	//s.SendPause(false)

	//var tag int
	//select {
	//case request := <-s.demoRequested:
		//tag = request
	//case <-ctx.Done():
		//return ctx.Err()
	//}

	//logger.Info().Msg("user requested demo")

	//file, err := os.Open(s.Path)
	//defer file.Close()
	//if err != nil {
		//return err
	//}

	//buffer, err := io.ReadAll(file)
	//if err != nil {
		//return err
	//}

	//p = game.Packet{}
	//p.Put(
		//game.N_SENDDEMO,
		//tag,
	//)
	//p = append(p, buffer...)
	//err = s.SendClientSync(p, 2)
	//if err != nil {
		//return err
	//}
	//logger.Info().Msg("demo downloaded")

	//time.Sleep(500 * time.Millisecond)

	//if ctx.Err() != nil {
		//return ctx.Err()
	//}

	//// Then load the demo
	//s.SendPause(true)
	//s.MoveClient(512-10, 512-10)
	//time.Sleep(500 * time.Millisecond)

	//if ctx.Err() != nil {
		//return ctx.Err()
	//}

	//s.SendPause(false)

	//logger.Info().Msg("download complete")

	return nil
}

type MapSender struct {
	Users    map[*User]*SendState
	Maps       *assets.AssetFetcher
	Mutex      sync.Mutex
	workingDir string
}

func NewMapSender(maps *assets.AssetFetcher) *MapSender {
	return &MapSender{
		Users: make(map[*User]*SendState),
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
func (m *MapSender) IsHandling(user *User) bool {
	m.Mutex.Lock()
	_, handling := m.Users[user]
	m.Mutex.Unlock()
	return handling
}

func (m *MapSender) SendDemo(ctx context.Context, user *User, tag int) {
	m.Mutex.Lock()
	state, handling := m.Users[user]
	m.Mutex.Unlock()

	if !handling {
		return
	}

	state.SendDemo(tag)
}

func (m *MapSender) TriggerSend(ctx context.Context, user *User) {
	m.Mutex.Lock()
	state, handling := m.Users[user]
	m.Mutex.Unlock()

	if !handling {
		return
	}

	state.TriggerSend()
}

func (m *MapSender) SendMap(ctx context.Context, user *User, mapName string) {
	logger := user.Logger()
	logger.Info().Str("map", mapName).Msg("sending map")
	state := &SendState{
		User:        user,
		Map:           mapName,
		Maps:          m.Maps,
		Sender:        m,
		userAccepted:  make(chan bool, 1),
		demoRequested: make(chan int, 1),
	}

	m.Mutex.Lock()
	m.Users[user] = state
	m.Mutex.Unlock()
	server := user.GetServer()

	out := make(chan error)
	go func() {
		out <- state.Send()
	}()
	go func() {
		select {
		case <-user.ServerSessionContext().Done():
			return
		case err := <-out:
			if err != nil {
				logger.Error().Err(err).Msg("failed to download map")
				return
			}

			m.Mutex.Lock()
			delete(m.Users, user)
			m.Mutex.Unlock()

			// Now we can reconnect the user to their server
			user.DisconnectFromServer()
			user.Connect(server)
		}
	}()
}

func (m *MapSender) Shutdown() {
	os.RemoveAll(m.workingDir)
}

func (c *Cluster) SendMap(ctx context.Context, user *User, name string) error {
	server := user.GetServer()
	instance := c.spaces.FindInstance(server)

	if instance != nil && instance.Editing != nil {
		e := instance.Editing
		err := e.Checkpoint(ctx)
		if err != nil {
			return err
		}

		data, err := e.Map.LoadMapData(ctx)
		if err != nil {
			return err
		}

		p := game.Packet{}
		p.Put(game.N_SENDMAP)
		p = append(p, data...)

		user.Send(game.GamePacket{
			Channel: 2,
			Data:    p,
		})

		return nil
	}

	data, err := c.assets.FetchMapBytes(ctx, name)
	if err != nil {
		return err
	}

	p := game.Packet{}
	p.Put(game.N_SENDMAP)
	p = append(p, data...)
	user.Send(game.GamePacket{
		Channel: 2,
		Data:    p,
	})

	log.Info().Msgf("Sent map %s (%d) to client", name, len(data))

	return nil
}
