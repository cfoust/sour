package io

import (
	"fmt"
	"reflect"

	"github.com/cfoust/sour/pkg/game/constants"
)

type Marshalable interface {
	Marshal(p *Packet) error
}

type Unmarshalable interface {
	Unmarshal(p *Packet) error
}

var MARSHALABLE = reflect.TypeOf((*Marshalable)(nil)).Elem()
var UNMARSHALABLE = reflect.TypeOf((*Unmarshalable)(nil)).Elem()

// There are some mistakes that can't be checked automatically at compile time.
// This is an issue with golang: to cast to an interface, the receivers of all
// of that interface's methods must be of the same kind (pointer or value) in
// order for the casting statement to return "ok".
//
// We specifically want Unmarshable to have a pointer receiver and Marshalable
// to have a value receiver, because that's how the serde library works.
func checkInterfaces(type_ reflect.Type, value reflect.Value) error {
	if _, ok := value.Interface().(Marshalable); ok {
		if type_.Kind() == reflect.Pointer {
			return fmt.Errorf("Implementation of Marshalable for %s should have value receiver", type_.String())
		}

		pointerType := reflect.PointerTo(type_)
		if !pointerType.Implements(UNMARSHALABLE) {
			return fmt.Errorf("implementation of Unmarshalable missing for %s", type_.String())
		}
	}

	if _, ok := value.Interface().(Unmarshalable); ok {
		if value.Kind() != reflect.Pointer {
			return fmt.Errorf("implementation of Unmarshalable for %s should have pointer receiver", type_.String())
		}

		valueType := value.Elem().Type()
		if !valueType.Implements(MARSHALABLE) {
			return fmt.Errorf("implementation of Marshalable missing for %s", type_.String())
		}
	}

	return nil
}

// Get the first field of a struct type.
func findTerminationField(type_ reflect.Type) (reflect.Type, error) {
	if type_.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type:term only applies to struct slices")
	}

	if type_.NumField() == 0 {
		return nil, fmt.Errorf("type:term requires at least one field")
	}

	// i'll be back
	terminator := type_.Field(0).Type

	switch terminator.Kind() {
	case reflect.String:
		fallthrough
	case reflect.Int32, reflect.Int:
		return terminator, nil
	default:
		return nil, fmt.Errorf("type:term had invalid terminator type")
	}
}

func unmarshalStruct(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot unmarshal non-struct")
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		switch field.Type.Kind() {
		case reflect.Array:
			numElements := field.Type.Len()
			for i := 0; i < numElements; i++ {
				err := UnmarshalValue(p, field.Type.Elem(), fieldValue.Index(i).Addr())
				if err != nil {
					return err
				}
			}

		case reflect.Slice:
			element := field.Type.Elem()

			endType := "count"

			tag := field.Tag
			if len(tag) != 0 {
				endType, _ = field.Tag.Lookup("type")
			}

			slice := reflect.MakeSlice(field.Type, 0, 0)

			switch endType {
			case "term":
				// There is some condition that indicates the array is done
				terminator, err := findTerminationField(element)
				if err != nil {
					return err
				}

				for {
					peekable := Packet(*p)

					done := false
					switch terminator.Kind() {
					case reflect.Int, reflect.Int32:
						endValue, ok := peekable.GetInt()
						if !ok {
							return fmt.Errorf("failed to read int condition")
						}

						if endValue < 0 {
							p.GetInt()
							done = true
							break
						}
					case reflect.String:
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
					err := UnmarshalValue(p, element, entry)
					if err != nil {
						return err
					}

					slice = reflect.Append(slice, entry.Elem())
				}
			case "count":
				readElements, ok := p.GetInt()
				if !ok {
					return fmt.Errorf("failed to read number of elements")
				}
				numElements := int(readElements)

				for i := 0; i < numElements; i++ {
					entry := reflect.New(element)
					err := UnmarshalValue(p, element, entry)
					if err != nil {
						return err
					}

					slice = reflect.Append(slice, entry.Elem())
				}
				break
			default:
				return fmt.Errorf("unhandled end type: %s", endType)
			}

			fieldValue.Set(slice)

		default:
			err := UnmarshalValue(p, field.Type, fieldValue.Addr())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func UnmarshalValue(p *Packet, type_ reflect.Type, valuePtr reflect.Value) error {
	if valuePtr.Kind() != reflect.Pointer {
		return fmt.Errorf("cannot unmarshal into non-pointer value")
	}

	if u, ok := valuePtr.Interface().(Unmarshalable); ok {
		err := checkInterfaces(type_, valuePtr)
		if err != nil {
			return err
		}

		return u.Unmarshal(p)
	}

	value := valuePtr.Elem()

	switch type_.Kind() {
	case reflect.Int32, reflect.Int:
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
	case reflect.Float64:
		readValue, ok := p.GetInt()
		if !ok {
			return fmt.Errorf("error reading float")
		}
		value.SetFloat(float64(readValue) / constants.DMF)
	case reflect.Uint32:
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
		err := UnmarshalValue(
			p,
			reflect.TypeOf(piece).Elem(),
			reflect.ValueOf(piece),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func marshalStruct(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("cannot marshal non-struct")
	}

	for i := 0; i < type_.NumField(); i++ {
		field := type_.Field(i)
		fieldValue := value.Field(i)

		switch field.Type.Kind() {
		case reflect.Array:
			// No need to put the number of elements if it's constant
			for i := 0; i < field.Type.Len(); i++ {
				err := MarshalValue(p, field.Type.Elem(), fieldValue.Index(i))
				if err != nil {
					return err
				}
			}

		case reflect.Slice:
			element := field.Type.Elem()

			endType := "count"

			tag := field.Tag
			if len(tag) != 0 {
				endType, _ = field.Tag.Lookup("type")
			}

			switch endType {
			case "term":
				// There is some condition that indicates the array is done
				terminator, err := findTerminationField(element)
				if err != nil {
					return err
				}

				numElements := fieldValue.Len()

				for i := 0; i < numElements; i++ {
					err := MarshalValue(p, element, fieldValue.Index(i))
					if err != nil {
						return err
					}
				}

				switch terminator.Kind() {
				case reflect.Int32, reflect.Int:
					err := p.Put(-1)
					if err != nil {
						return err
					}
				case reflect.String:
					err := p.Put("")
					if err != nil {
						return err
					}
				}
			case "count":
				numElements := fieldValue.Len()

				err := p.Put(numElements)
				if err != nil {
					return err
				}

				for i := 0; i < numElements; i++ {
					err := MarshalValue(p, element, fieldValue.Index(i))
					if err != nil {
						return err
					}
				}
				break
			default:
				return fmt.Errorf("unhandled end type: %s", endType)
			}
		default:
			err := MarshalValue(p, field.Type, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func MarshalValue(p *Packet, type_ reflect.Type, value reflect.Value) error {
	if value.Kind() == reflect.Pointer {
		// we could support this, but it's a code smell because encoded
		// data by definition cannot "point" anywhere
		return fmt.Errorf("cannot marshal pointer to value")
	}

	if u, ok := value.Interface().(Marshalable); ok {
		err := checkInterfaces(type_, value)
		if err != nil {
			return err
		}

		return u.Marshal(p)
	}

	switch type_.Kind() {
	case reflect.Int32, reflect.Int:
		p.PutInt(int32(value.Int()))
	case reflect.Uint8:
		p.PutByte(byte(value.Uint()))
	case reflect.Float64:
		p.PutInt(int32(value.Float() * constants.DMF))
	case reflect.Bool:
		boolean := value.Bool()
		if boolean {
			p.PutInt(1)
		} else {
			p.PutInt(0)
		}
	case reflect.Uint32:
		p.PutUint(uint32(value.Uint()))
	case reflect.String:
		p.PutString(value.String())
	case reflect.Struct:
		err := marshalStruct(p, type_, value)
		if err != nil {
			return err
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

		err := MarshalValue(p, type_, value)
		if err != nil {
			return err
		}
	}

	return nil
}
