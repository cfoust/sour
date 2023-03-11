package protocol

import (
	"fmt"
	"reflect"

	"github.com/cfoust/sour/pkg/game/io"
	"github.com/rs/zerolog/log"
)

func getMessageType(code MessageCode, fromClient bool) Message {
	if fromClient {
		message, ok := CLIENT_MESSAGES[code]
		if !ok {
			return nil
		}

		return message
	}

	message, ok := SERVER_MESSAGES[code]
	if !ok {
		return nil
	}

	return message
}

func Decode(b []byte, fromClient bool) ([]Message, error) {
	messages := make([]Message, 0)
	p := io.Packet(b)

	for len(p) > 0 {
		type_, ok := p.GetInt()
		if !ok {
			return nil, fmt.Errorf("failed to read message")
		}

		code := MessageCode(type_)

		if code >= NUMMSG {
			return nil, fmt.Errorf("code %d is not in range of messages", code)
		}

		if !IsSpammyMessage(code) {
			log.Debug().Msgf("decoding %s", code.String())
		}

		messageType := getMessageType(code, fromClient)
		if messageType == nil {
			return nil, fmt.Errorf("code %d did not correspond to a message type", code)
		}

		resultType := reflect.TypeOf(messageType).Elem()
		resultValue := reflect.New(resultType)
		err := io.UnmarshalValue(&p, resultType, resultValue)
		if err != nil {
			return nil, err
		}

		message := resultValue.Elem().Interface().(Message)
		messages = append(messages, message)
	}

	return messages, nil
}

func Encode(messages ...Message) ([]byte, error) {
	p := io.Packet{}

	// N_CLIENT has a field indicating the number of bytes to follow.
	if len(messages) > 0 && messages[0].Type() == N_CLIENT {
		client := messages[0].(ClientPacket)
		rest, err := Encode(messages[1:]...)
		if err != nil {
			return nil, err
		}

		err = p.Put(N_CLIENT, ClientPacket{
			Client: client.Client,
			Length: int32(len(rest)),
		})
		if err != nil {
			return nil, err
		}

		return append(p, rest...), nil
	}

	for _, message := range messages {
		err := p.Put(message.Type(), message)

		if err != nil {
			return nil, err
		}
	}

	return p, nil
}
