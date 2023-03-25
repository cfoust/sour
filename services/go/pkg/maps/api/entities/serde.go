package entities

import (
	"fmt"
	"log"
	"reflect"

	C "github.com/cfoust/sour/pkg/game/constants"
)

type Attributes []int16

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

func decodeValue(a *Attributes, type_ reflect.Type, valuePtr reflect.Value) error {
	if valuePtr.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot decode into non-pointer value")
	}

	if u, ok := valuePtr.Interface().(Decodable); ok {
		return u.Decode(a)
	}

	value := valuePtr.Elem()
	log.Printf("decodeValue %+v %+v %+v", type_, valuePtr, value)

	switch type_.Kind() {
	case reflect.Int16, reflect.Uint8:
		readValue, err := a.Get()
		if err != nil {
			return err
		}
		value.SetInt(int64(readValue))
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
	log.Printf("%+v %+v", type_, value)
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot decode non-struct: %+v", value)
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		err := decodeValue(a, field.Type, fieldValue.Addr())
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
