package entities

import (
	"fmt"
	"reflect"

	C "github.com/cfoust/sour/pkg/game/constants"
)

type Attributes []int16

var Empty = fmt.Errorf("value was missing, check default")

func (a *Attributes) Get() (int16, error) {
	if len(*a) < 1 {
		return 0, fmt.Errorf("no attributes remaining")
	}

	value := (*a)[0]
	*a = (*a)[1:]
	return value, nil
}

func (a *Attributes) Put(value int16) {
	*a = append(*a, value)
}

type Decodable interface {
	Decode(*Attributes) error
}

type Encodable interface {
	Encode(*Attributes) error
}

// just like banks
type Default[T any] struct {
	isEmpty bool
	value   T
}

func (d *Default[T]) Set(value T) {
	d.value = value
	d.isEmpty = false
}

// Indicate that this should use the default value.
func (d *Default[T]) Clear() {
	d.isEmpty = true
}

func (d *Default[T]) IsEmpty() bool {
	return d.isEmpty
}

func (d *Default[T]) Get() T {
	return d.value
}

func NewDefault[T any](value T) Default[T] {
	return Default[T]{value: value}
}

type Defaultable interface {
	Defaults() Defaultable
}

func decodeValue(a *Attributes, type_ reflect.Type, valuePtr reflect.Value) error {
	if valuePtr.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot decode into non-pointer value")
	}

	if u, ok := valuePtr.Interface().(Decodable); ok {
		return u.Decode(a)
	}

	value := valuePtr.Elem()

	switch type_.Kind() {
	case reflect.Int16:
		readValue, err := a.Get()
		if err != nil {
			return err
		}
		if readValue == 0 {
			return Empty
		}
		value.SetInt(int64(readValue))
	case reflect.Float32:
		readValue, err := a.Get()
		if err != nil {
			return err
		}
		if readValue == 0 {
			return Empty
		}
		value.SetFloat(float64(readValue) / 100.0)
	case reflect.Uint8:
		readValue, err := a.Get()
		if err != nil {
			return err
		}
		if readValue == 0 {
			return Empty
		}
		value.SetUint(uint64(readValue))
	case reflect.Struct:
		err := decodeStruct(a, type_, value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func decodeStruct(a *Attributes, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot decode non-struct: %+v", value)
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		err := decodeValue(a, field.Type, fieldValue.Addr())
		if err == Empty {
			if d, ok := value.Interface().(Defaultable); ok {
				defaultValue := reflect.ValueOf(d.Defaults())
				fieldValue.Set(defaultValue.Field(i))
			}
			err = nil
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func Decode(entityType C.EntityType, a *Attributes) (EntityInfo, error) {
	type_, ok := ENTITY_TYPE_MAP[entityType]
	if !ok {
		return nil, fmt.Errorf("unknown entity type %s", entityType.String())
	}

	decodedType := reflect.TypeOf(type_)
	decoded := reflect.New(decodedType.Elem())
	err := decodeValue(a, decodedType.Elem(), decoded)
	if err != nil {
		return nil, err
	}

	if value, ok := decoded.Interface().(EntityInfo); ok {
		return value, nil
	}

	return nil, fmt.Errorf("failed to decode entity")
}

func encodeStruct(a *Attributes, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot encode non-struct")
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		err := encodeValue(a, field.Type, fieldValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func encodeValue(a *Attributes, type_ reflect.Type, value reflect.Value) error {
	if u, ok := value.Interface().(Encodable); ok {
		return u.Encode(a)
	}

	if value.Kind() == reflect.Pointer {
		value = value.Elem()
		type_ = type_.Elem()
	}

	switch type_.Kind() {
	case reflect.Int16:
		a.Put(int16(value.Int()))
	case reflect.Float32:
		a.Put(int16(value.Float() * 100.0))
	case reflect.Uint8:
		a.Put(int16(value.Uint()))
	case reflect.Struct:
		err := encodeStruct(a, type_, value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unimplemented type: %s", type_.String())
	}

	return nil
}

func Encode(info EntityInfo) (*Attributes, error) {
	a := Attributes{}
	err := encodeValue(&a, reflect.TypeOf(info), reflect.ValueOf(info))
	if err != nil {
		return nil, err
	}

	for len(a) < 5 {
		a.Put(0)
	}

	return &a, nil
}
