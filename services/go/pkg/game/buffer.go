package game

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

// Similar to game.Packet, but map IO does not do any compression.
type Buffer []byte

func (p *Buffer) unmarshalRawValue(type_ reflect.Type, value interface{}) error {
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

func (p *Buffer) Unmarshal(pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece).Elem()

		err := p.unmarshalRawValue(type_, piece)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Buffer) marshalRawValue(type_ reflect.Type, value interface{}) error {
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

func (p *Buffer) Marshal(pieces ...interface{}) error {
	for _, piece := range pieces {
		type_ := reflect.TypeOf(piece)

		err := p.marshalRawValue(type_, piece)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Buffer) GetByte() (byte, bool) {
	packet := Packet(*p)
	value, ok := packet.GetByte()
	(*p) = []byte(packet)
	return value, ok
}

func (p *Buffer) GetBytes(n int) ([]byte, bool) {
	if n > len(*p) {
		return nil, false
	}
	b := make([]byte, n)
	copy(b, (*p)[:n])
	*p = (*p)[n:]
	return b, true
}

func (p *Buffer) Read(n []byte) (int, error) {
	packet := Packet(*p)
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
	return p.Unmarshal(pieces...)
}

func (p *Buffer) Put(pieces ...interface{}) error {
	return p.Marshal(pieces...)
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
