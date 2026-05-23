package osrs

import (
	"fmt"
	"io"
)

// Buffer is a sequential byte reader matching OSRS's Buffer class.
// All multi-byte reads are big-endian.
type Buffer struct {
	data []byte
	pos  int
}

func NewBuffer(data []byte) *Buffer {
	return &Buffer{data: data}
}

func (b *Buffer) Remaining() int {
	return len(b.data) - b.pos
}

func (b *Buffer) Position() int {
	return b.pos
}

func (b *Buffer) SetPosition(pos int) {
	b.pos = pos
}

func (b *Buffer) ReadUnsignedByte() (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	v := int(b.data[b.pos]) & 0xff
	b.pos++
	return v, nil
}

func (b *Buffer) ReadSignedByte() (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	v := int(int8(b.data[b.pos]))
	b.pos++
	return v, nil
}

func (b *Buffer) ReadUShort() (int, error) {
	if b.pos+2 > len(b.data) {
		return 0, io.EOF
	}
	v := (int(b.data[b.pos]&0xff) << 8) + int(b.data[b.pos+1]&0xff)
	b.pos += 2
	return v, nil
}

func (b *Buffer) ReadTriByte() (int, error) {
	if b.pos+3 > len(b.data) {
		return 0, io.EOF
	}
	v := (int(b.data[b.pos]&0xff) << 16) + (int(b.data[b.pos+1]&0xff) << 8) + int(b.data[b.pos+2]&0xff)
	b.pos += 3
	return v, nil
}

func (b *Buffer) ReadInt() (int, error) {
	if b.pos+4 > len(b.data) {
		return 0, io.EOF
	}
	v := (int(b.data[b.pos]&0xff) << 24) + (int(b.data[b.pos+1]&0xff) << 16) +
		(int(b.data[b.pos+2]&0xff) << 8) + int(b.data[b.pos+3]&0xff)
	b.pos += 4
	return v, nil
}

// ReadUSmart reads a "smart" unsigned value: 1 byte if < 128, else 2-byte unsigned - 32768.
func (b *Buffer) ReadUSmart() (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	v := int(b.data[b.pos]) & 0xff
	if v < 128 {
		return b.ReadUnsignedByte()
	}
	us, err := b.ReadUShort()
	if err != nil {
		return 0, err
	}
	return us - 32768, nil
}

// ReadSmart reads a "smart" signed value: 1 byte - 64 if < 128, else 2-byte unsigned - 49152.
func (b *Buffer) ReadSmart() (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	v := int(b.data[b.pos]) & 0xff
	if v < 128 {
		ub, err := b.ReadUnsignedByte()
		if err != nil {
			return 0, err
		}
		return ub - 64, nil
	}
	us, err := b.ReadUShort()
	if err != nil {
		return 0, err
	}
	return us - 49152, nil
}

// ReadIncrementalSmart reads accumulated USmart values until one is != 32767.
func (b *Buffer) ReadIncrementalSmart() (int, error) {
	value := 0
	for {
		remainder, err := b.ReadUSmart()
		if err != nil {
			return 0, err
		}
		value += remainder
		if remainder != 32767 {
			break
		}
	}
	return value, nil
}

func (b *Buffer) ReadString() (string, error) {
	start := b.pos
	for b.pos < len(b.data) {
		if b.data[b.pos] == 10 { // newline
			s := string(b.data[start:b.pos])
			b.pos++
			return s, nil
		}
		b.pos++
	}
	return "", fmt.Errorf("unterminated string at offset %d", start)
}

func (b *Buffer) ReadBytes(n int) ([]byte, error) {
	if b.pos+n > len(b.data) {
		return nil, io.EOF
	}
	out := make([]byte, n)
	copy(out, b.data[b.pos:b.pos+n])
	b.pos += n
	return out, nil
}

func (b *Buffer) Skip(n int) {
	b.pos += n
}
