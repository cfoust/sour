package game

import (
	"reflect"
	"fmt"
	"log"
)

type Message interface {
	Type() MessageCode
	Data() interface{}
}

type RawMessage struct {
	code MessageCode
	data interface{}
}

func (m RawMessage) Type() MessageCode {
	return m.code
}

func (m RawMessage) Data() interface{} {
	return m.data
}

func Unmarshal(p Packet, code MessageCode, message interface{}) (Message, error) {
	raw := RawMessage{
		code: code,
		data: message,
	}

	type_ := reflect.TypeOf(message)
	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		log.Print(field.Name)
	}

	return raw, nil
}

func Read(b []byte) (*[]Message, error) {
	messages := make([]Message, 0)
	p := Packet(b)

	for len(p) > 0 {
		type_, ok := p.GetInt()
		if !ok {
			return nil, fmt.Errorf("failed to read message")
		}

		code := MessageCode(type_)

		var message Message
		var err error
		switch code {
		case N_WELCOME:
			message, err = Unmarshal(p, N_WELCOME, Welcome{})
		case N_MAPCHANGE:
			message, err = Unmarshal(p, N_MAPCHANGE, MapChange{})
		default:
			return nil, fmt.Errorf("unhandled code %s", code.String())
		}

		if err != nil {
			return nil, err
		}

		messages = append(messages, message)
	}

	return &messages, nil
}
