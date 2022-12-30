package maps

import (
	"bytes"
	"encoding/binary"
	"reflect"

	"github.com/cfoust/sour/pkg/game"
)

// Similar to game.Packet, but map IO does not do any compression.
type Buffer []byte

func unmarshalRawValue(p *Buffer, type_ reflect.Type, value interface{}) error {
	var err error
	switch v := value.(type) {
	default:
		err = binary.Read(p, binary.LittleEndian, v)
	}

	if err != nil {
		return err
	}

	return nil
}

func Unmarshal(p *Buffer, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece).Elem()

		err := unmarshalRawValue(p, type_, piece)
		if err != nil {
			return err
		}
	}

	return nil
}

func marshalRawValue(p *Buffer, type_ reflect.Type, value interface{}) error {
	var buffer bytes.Buffer

	var err error
	switch v := value.(type) {
	default:
		err = binary.Write(&buffer, binary.LittleEndian, v)
	}

	if err != nil {
		return err
	}

	*p = append(*p, buffer.Bytes()...)

	return nil
}

func Marshal(p *Buffer, pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece)

		err := marshalRawValue(p, type_, piece)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Buffer) GetByte() (byte, bool) {
	packet := game.Packet(*p)
	value, ok := packet.GetByte()
	(*p) = []byte(packet)
	return value, ok
}

func (p *Buffer) Read(n []byte) (int, error) {
	packet := game.Packet(*p)
	numRead, err := packet.Read(n)
	(*p) = []byte(packet)
	return numRead, err
}

func (p *Buffer) Skip(n int) bool {
	if n > len(*p) {
		return false
	}
	(*p) = (*p)[n:]
	return true
}

func (p *Buffer) Get(pieces ...interface{}) error {
	return Unmarshal(p, pieces...)
}

func (p *Buffer) Put(pieces ...interface{}) error {
	return Marshal(p, pieces...)
}

func (p *Buffer) GetString() (string, bool) {
	var length uint16
	err := p.Get(&length)
	if err != nil {
		return "", false
	}
	value := make([]byte, length)
	err = p.Get(&value)
	if err != nil {
		return "", false
	}
	return string(value), true
}

func (p *Buffer) GetFloat() (float32, bool) {
	var value float32
	err := p.Get(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *Buffer) GetShort() (uint16, bool) {
	var value uint16
	err := p.Get(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *Buffer) GetInt() (int32, bool) {
	var value int32
	err := p.Get(&value)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (p *Buffer) GetStringByte() (string, bool) {
	var length byte
	err := p.Get(&length)
	if err != nil {
		return "", false
	}
	value := make([]byte, length+1)
	err = p.Get(&value)
	if err != nil {
		return "", false
	}
	return string(value), true
}
