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

func (s *ServerInfoService) Handle(request *game.Packet, out chan []byte) error {
	// The response includes the entirety of the
	// request since they use it to calculate ping
	// time
	response := game.Packet(*request)

	millis, ok := request.GetInt()
	if !ok {
		return fmt.Errorf("invalid request")
	}

	if millis == 0 {
		extCmd, ok := request.GetInt()
		if !ok {
			return fmt.Errorf("missing cmd")
		}

		switch extCmd {
		case EXT_UPTIME:
			response.PutInt(int32(s.provider.GetUptime()))
			out <- response
			return nil
		case EXT_PLAYERSTATS:
			clientNum, ok := request.GetInt()
			if !ok {
				return fmt.Errorf("missing client")
			}

			if clientNum >= 0 {
				clients := s.provider.GetClientInfo()

				found := false
				for _, client := range clients {
					if client.Client == int(clientNum) {
						found = true
						break
					}
				}

				if !found {
					response.PutInt(EXT_ERROR)
					out <- response
					return nil
				}
			}

			response.PutInt(EXT_NO_ERROR)

			// Remember position
			q := game.Packet(response)
			q.PutInt(EXT_PLAYERSTATS_RESP_IDS)
			if clientNum >= 0 {
				q.PutInt(clientNum)
			} else {
				clients := s.provider.GetClientInfo()

				for _, client := range clients {
					q.PutInt(int32(client.Client))
				}
			}
			out <- q

			clients := s.provider.GetClientInfo()

			for _, client := range clients {
				if clientNum < 0 || client.Client != int(clientNum) {
					break
				}
				q = game.Packet(response)
				q.PutInt(EXT_PLAYERSTATS_RESP_STATS)
				q.Put(client)
				out <- q
			}
		default:
			return fmt.Errorf("unsupported extinfo command: %d", extCmd)
		}
		return nil
	}

	info := s.provider.GetServerInfo()

	response.Put(info.NumClients)

	// The number of attributes following
	if info.GameSpeed != 100 || info.GamePaused {
		response.Put(7)
	} else {
		response.Put(5)
	}

	err := response.Put(
		PROTOCOL_VERSION,
		info.GameMode,
		info.TimeLeft,
		info.MaxClients,
		info.PasswordMode,
	)
	if err != nil {
		return err
	}

	if info.GameSpeed != 100 || info.GamePaused {
		response.Put(
			info.GamePaused,
			info.GameSpeed,
		)
	}

	err = response.Put(
		info.Map,
		info.Description,
	)
	if err != nil {
		return err
	}

	out <- response

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

				err := s.Handle(&request, event.Response)
				if err != nil {
					log.Warn().Err(err).Msg("error handling server info")
					continue
				}

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
