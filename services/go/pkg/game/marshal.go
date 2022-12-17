package game

import (
	"reflect"
	"fmt"

	"github.com/rs/zerolog/log"
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

func unmarshalStruct(p Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)
		ref := fmt.Sprintf("%s.%s", type_.Name(), field.Name)

		log.Debug().Str("ref", ref).Msg("parsing field")

		tag := field.Tag
		if len(tag) > 0 {
			return fmt.Errorf("unhandled tag %s: %s", ref, tag)
		}

		switch field.Type.Kind() {
		case reflect.Int:
			readValue, ok := p.GetInt()
			if !ok {
				return fmt.Errorf("error reading int %s", ref)
			}
			fieldValue.SetInt(int64(readValue))
		case reflect.String:
			readValue, ok := p.GetString()
			if !ok {
				return fmt.Errorf("error reading string %s", ref)
			}
			fieldValue.SetString(readValue)
		case reflect.Struct:
			err := unmarshalStruct(p, field.Type, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Unmarshal(p Packet, code MessageCode, message interface{}) (Message, error) {
	raw := RawMessage{
		code: code,
		data: message,
	}

	type_ := reflect.TypeOf(message)
	value := reflect.ValueOf(message)

	if value.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("can't unmarshal non-pointer")
	}

	log.Debug().Str("type", type_.Elem().Name()).Msg("parsing message")
	err := unmarshalStruct(p, type_.Elem(), value.Elem())

	return raw, err
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
			message, err = Unmarshal(p, N_WELCOME, &Welcome{})
		case N_MAPCHANGE:
			message, err = Unmarshal(p, N_MAPCHANGE, &MapChange{})
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
