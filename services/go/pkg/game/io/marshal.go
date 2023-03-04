package io

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/rs/zerolog/log"
)

type Unmarshalable interface {
	Unmarshal(p *Packet) error
}

type Marshalable interface {
	Marshal(p *Packet) error
}

func unmarshalStruct(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
	}

	if type_ == reflect.TypeOf(PhysicsState{}) {
		state := readPhysics(p)
		value.Set(reflect.ValueOf(state))
		return nil
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		switch field.Type.Kind() {
		case reflect.Slice:
			element := field.Type.Elem()
			tag := field.Tag
			if len(tag) == 0 {
				return fmt.Errorf("all arrays must specify tag")
			}

			endType, haveType := field.Tag.Lookup("type")
			if !haveType {
				return fmt.Errorf("all arrays must specify a type in the tag")
			}

			slice := reflect.MakeSlice(field.Type, 0, 0)

			switch endType {
			// There is some condition that indicates the array is done
			case "term":
				cmp, haveCmp := field.Tag.Lookup("cmp")
				if !haveCmp {
					return fmt.Errorf("term tags must specify end condition")
				}

				for {
					peekable := Packet(*p)

					done := false
					switch cmp {
					case "gez":
						endValue, ok := peekable.GetInt()
						if !ok {
							return fmt.Errorf("failed to read int condition")
						}

						if endValue < 0 {
							p.GetInt()
							done = true
							break
						}
					case "len":
						endValue, ok := peekable.GetString()
						if !ok {
							return fmt.Errorf("failed to read string condition")
						}

						if len(endValue) == 0 {
							p.GetString()
							done = true
							break
						}
					}

					if done {
						break
					}

					entry := reflect.New(element)
					err := unmarshalStruct(p, element, entry.Elem())
					if err != nil {
						return err
					}

					reflect.Append(slice, entry.Elem())
				}
			case "count":
				number, haveConst := field.Tag.Lookup("const")
				var numElements int
				if haveConst {
					numElements, _ = strconv.Atoi(number)
				} else {
					readElements, ok := p.GetInt()
					if !ok {
						return fmt.Errorf("failed to read number of elements")
					}
					numElements = int(readElements)
				}

				for i := 0; i < numElements; i++ {
					entry := reflect.New(element)
					err := unmarshalStruct(p, element, entry.Elem())
					if err != nil {
						return err
					}

					reflect.Append(slice, entry.Elem())
				}
				break
			default:
				return fmt.Errorf("unhandled end type: %s", endType)
			}

			fieldValue.Set(slice)

		case reflect.Struct:
			err := unmarshalStruct(p, field.Type, fieldValue)
			if err != nil {
				return err
			}
		default:
			err := unmarshalValue(p, field.Type, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func unmarshalValue(p *Packet, type_ reflect.Type, value reflect.Value) error {
	switch type_.Kind() {
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		readValue, ok := p.GetInt()
		if !ok {
			return fmt.Errorf("error reading int")
		}
		value.SetInt(int64(readValue))
	case reflect.Uint8:
		readValue, ok := p.GetByte()
		if !ok {
			return fmt.Errorf("error reading byte")
		}
		value.SetUint(uint64(readValue))
	case reflect.Bool:
		readValue, ok := p.GetInt()
		if !ok {
			return fmt.Errorf("error reading bool")
		}
		if readValue == 1 {
			value.SetBool(true)
		} else {
			value.SetBool(false)
		}
	case reflect.Float32:
		readValue, ok := p.GetFloat()
		if !ok {
			return fmt.Errorf("error reading float")
		}
		value.SetFloat(float64(readValue))
	case reflect.Uint:
		readValue, ok := p.GetUint()
		if !ok {
			return fmt.Errorf("error reading uint")
		}
		value.SetUint(uint64(readValue))
	case reflect.String:
		readValue, ok := p.GetString()
		if !ok {
			return fmt.Errorf("error reading string")
		}
		value.SetString(readValue)
	case reflect.Struct:
		err := unmarshalStruct(p, type_, value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func Unmarshal(p *Packet, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece).Elem()
		pointer := reflect.ValueOf(piece)
		value := pointer.Elem()

		if u, ok := pointer.Interface().(Unmarshalable); ok {
			err := u.Unmarshal(p)
			if err != nil {
				return err
			}
			continue
		}

		err := unmarshalValue(p, type_, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func marshalValue(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if type_ == reflect.TypeOf(PhysicsState{}) {
		return writePhysics(p, value.Interface().(PhysicsState))
	}

	if u, ok := value.Interface().(Marshalable); ok {
		return u.Marshal(p)
	}

	switch type_.Kind() {
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		p.PutInt(int32(value.Int()))
	case reflect.Uint8:
		p.PutByte(byte(value.Uint()))
	case reflect.Float32:
		p.PutFloat(float32(value.Float()))
	case reflect.Bool:
		boolean := value.Bool()
		if boolean {
			p.PutInt(1)
		} else {
			p.PutInt(0)
		}
	case reflect.Uint32:
		fallthrough
	case reflect.Uint:
		p.PutUint(uint32(value.Uint()))
	case reflect.String:
		p.PutString(value.String())
	case reflect.Struct:
		for i := 0; i < type_.NumField(); i++ {
			field := type_.Field(i)
			fieldValue := value.Field(i)
			marshalValue(p, field.Type, fieldValue)
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func Marshal(p *Packet, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece)
		value := reflect.ValueOf(piece)

		err := marshalValue(p, type_, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func UnmarshalMessage(p *Packet, code MessageCode, message Message) error {
	// Throw away the type information (we got it already)
	p.GetInt()

	type_ := reflect.TypeOf(message)
	value := reflect.ValueOf(message)

	if value.Kind() != reflect.Ptr {
		return fmt.Errorf("can't unmarshal non-pointer")
	}

	var err error
	if u, ok := value.Interface().(Unmarshalable); ok {
		err = u.Unmarshal(p)
	} else {
		err = unmarshalStruct(p, type_.Elem(), value.Elem())
	}

	return err
}

// Peek the first byte to determine the message type but don't deserialize the
// rest of the packet.
func Peek(b []byte) (Message, error) {
	p := Packet(b)
	type_, ok := p.GetInt()
	if !ok {
		return nil, fmt.Errorf("failed to read message type")
	}

	code := MessageCode(type_)

	if code >= NUMMSG {
		return nil, fmt.Errorf("code %d is not in range of messages", code)
	}

	raw := RawMessage{
		code:    code,
		message: nil,
	}

	raw.data = b

	return raw, nil
}
