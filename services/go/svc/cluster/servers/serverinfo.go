package servers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/enet"
	"github.com/cfoust/sour/pkg/game"

	"github.com/rs/zerolog/log"
)

type ENetDatagram struct {
	socket *enet.Socket
}

func NewENetDatagram() *ENetDatagram {
	return &ENetDatagram{}
}

func (i *ENetDatagram) Serve(port int) error {
	socket, err := enet.NewDatagramSocket(port)
	if err != nil {
		return err
	}
	i.socket = socket
	return nil
}

func (i *ENetDatagram) Shutdown() {
	i.socket.DestroySocket()
}

type PingEvent struct {
	Request  []byte
	Response chan []byte
}

func (i *ENetDatagram) Poll(ctx context.Context) <-chan PingEvent {
	out := make(chan PingEvent)

	go func() {
		events := i.socket.Service()
		for {
			select {
			case event := <-events:
				go func(msg enet.SocketMessage) {
					ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

					defer cancel()

					response := make(chan []byte)

					out <- PingEvent{
						Request:  event.Data,
						Response: response,
					}

					select {
					case data := <-response:
						i.socket.SendDatagram(
							event.Address,
							data,
						)
					case <-ctx.Done():
						return
					}
				}(event)
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

const (
	EXT_ACK                    = -1
	EXT_VERSION                = 105
	EXT_NO_ERROR               = 0
	EXT_ERROR                  = 1
	EXT_PLAYERSTATS_RESP_IDS   = -10
	EXT_PLAYERSTATS_RESP_STATS = -11
	EXT_UPTIME                 = 0
	EXT_PLAYERSTATS            = 1
	EXT_TEAMSCORE              = 2
)

type ClientExtInfo struct {
	Client    int
	Ping      int
	Name      string
	Team      string
	Frags     int
	Flags     int
	Deaths    int
	TeamKills int
	Damage    int
	Health    int
	Armour    int
	GunSelect int
	Privilege int
	State     int
	Ip0       byte
	Ip1       byte
	Ip2       byte
}

func DecodeClientInfo(p game.Packet) (*ClientExtInfo, error) {
	client := ClientExtInfo{}
	err := p.Get(&client)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

type ServerUptime struct {
	TimeUp int
}

func DecodeServerUptime(p game.Packet) (*ServerUptime, error) {
	uptime := ServerUptime{}
	err := p.Get(&uptime)
	if err != nil {
		return nil, err
	}
	return &uptime, nil
}

type ServerInfo struct {
	NumClients int32
	GamePaused bool
	GameMode   int32
	// Seconds
	TimeLeft     int32
	MaxClients   int32
	PasswordMode int32
	GameSpeed    int32
	Map          string
	Description  string
}

func DecodeServerInfo(p game.Packet) (*ServerInfo, error) {
	info := ServerInfo{}

	var protocol int
	var numAttributes int
	err := p.Get(
		&info.NumClients,
		&numAttributes,
		&protocol,
		&info.GameMode,
		&info.TimeLeft,
		&info.MaxClients,
		&info.PasswordMode,
	)
	if err != nil {
		return nil, err
	}

	if numAttributes == 7 {
		err = p.Get(
			&info.GamePaused,
			&info.GameSpeed,
		)
	} else {
		info.GamePaused = false
		info.GameSpeed = 100
	}

	err = p.Get(
		&info.Map,
		&info.Description,
	)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

type InfoProvider interface {
	GetServerInfo() *ServerInfo
	GetClientInfo() []*ClientExtInfo
	GetUptime() int // seconds
}

type ServerInfoService struct {
	provider InfoProvider
	datagram *ENetDatagram
}

func NewServerInfoService(provider InfoProvider) *ServerInfoService {
	return &ServerInfoService{
		provider: provider,
		datagram: NewENetDatagram(),
	}
}

func (s *ServerInfoService) Handle(request *game.Packet, response *game.Packet) error {
	millis, ok := request.GetInt()
	if !ok {
		return fmt.Errorf("invalid request")
	}

	// TODO Other kinds of server info
	if millis == 0 {
		return fmt.Errorf("not yet implemented")
	}

	info := s.provider.GetServerInfo()

	response.PutInt(info.NumClients)

	// The number of attributes following
	if info.GameSpeed != 100 || info.GamePaused {
		response.PutInt(7)
	} else {
		response.PutInt(5)
	}

	response.PutInt(PROTOCOL_VERSION)
	response.PutInt(info.GameMode)
	response.PutInt(info.TimeLeft)
	response.PutInt(info.MaxClients)
	response.PutInt(info.PasswordMode)

	if info.GameSpeed != 100 || info.GamePaused {
		if info.GamePaused {
			response.PutInt(1)
		} else {
			response.PutInt(0)
		}

		response.PutInt(info.GameSpeed)
	}

	response.PutString(info.Map)
	response.PutString(info.Description)

	return nil
}

func (s *ServerInfoService) UpdateMaster(port int) error {
	socket, err := enet.NewConnectSocket("master.sauerbraten.org", 28787)
	defer socket.DestroySocket()

	if err != nil {
		return fmt.Errorf("error creating socket")
	}

	err = socket.SendString(fmt.Sprintf("regserv %d\n", port))
	if err != nil {
		return fmt.Errorf("error registering server")
	}

	output, length := socket.Receive()
	if length < 0 {
		return fmt.Errorf("failed to receive master response")
	}

	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "failreg") {
			return fmt.Errorf("master rejected registration: %s", line)
		} else if strings.HasPrefix(line, "succreg") {
			log.Info().Msg("registered with master")
			return nil
		}
	}

	return fmt.Errorf("failed to register")
}

func (s *ServerInfoService) PollMaster(ctx context.Context, port int) {
	tick := time.NewTicker(1 * time.Hour)

	for {
		err := s.UpdateMaster(port)
		if err != nil {
			log.Error().Err(err).Msg("failed to register with master")
		}

		select {
		case <-tick.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (s *ServerInfoService) Serve(ctx context.Context, port int, registerMaster bool) error {
	err := s.datagram.Serve(port)

	if err != nil {
		return err
	}

	if registerMaster {
		// You register a game server with master
		go s.PollMaster(ctx, port-1)
	}

	events := s.datagram.Poll(ctx)

	go func() {
		for {
			select {
			case event := <-events:
				request := game.Packet(event.Request)
				// The response includes the entirety of the
				// request since they use it to calculate ping
				// time
				response := game.Packet(event.Request)

				err := s.Handle(&request, &response)
				if err != nil {
					log.Warn().Err(err).Msg("error handling server info")
					continue
				}

				event.Response <- response
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (s *ServerInfoService) Shutdown() {
	s.datagram.Shutdown()
}
