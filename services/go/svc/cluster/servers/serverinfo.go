package servers

import (
	"context"
	"fmt"
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
	socket, err := enet.NewSocket("", port, enet.ENET_SOCKET_TYPE_DATAGRAM, false)
	if err != nil {
		return err
	}
	i.socket = socket
	return nil
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

type ServerInfo struct {
	NumClients int32
	GamePaused bool
	GameMode   int32
	// Seconds
	TimeRemaining int32
	MaxClients    int32
	PasswordMode  int32
	GameSpeed     int32
	Map           string
	Description   string
}

type InfoProvider interface {
	GetServerInfo() *ServerInfo
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

func (s *ServerInfoService) Serve(ctx context.Context, port int) error {
	err := s.datagram.Serve(port)

	if err != nil {
		return err
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
