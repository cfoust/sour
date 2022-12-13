package game

import (
	"github.com/cfoust/sour/pkg/game/cubecode"
)

// Packet represents a Sauerbraten UDP packet.
type Packet []byte

// PutInt writes an int32 to the packet buffer.
func (p *Packet) PutInt(v int32) {
	if -0x7F < v && v < 0x80 {
		*p = append(*p, byte(v))
	} else if -0x7FFF <= v && v < 0x8000 {
		*p = append(*p, 0x80, byte(v), byte(v>>8))
	} else {
		*p = append(*p, 0x81, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	}
}

// PutUint writes a uint32 to the packet buffer. It only keeps the 28 lowest bits!
func (p *Packet) PutUint(v uint32) {
	if v < (1 << 7) {
		*p = append(*p, byte(v))
	} else if v < (1 << 14) {
		*p = append(*p, (byte(v)&0x7F)|0x80, byte(v>>7))
	} else if v < (1 << 21) {
		*p = append(*p, (byte(v)&0x7F)|0x80, (byte(v>>7)&0x7F)|0x80, byte(v>>14))
	} else {
		*p = append(*p, (byte(v)&0x7F)|0x80, (byte(v>>7)&0x7F)|0x80, (byte(v>>14)&0x7F)|0x80, byte(v>>21))
	}
}

// PutInt writes a string to the packet buffer, encoding it with Sauer's conversion table at the same time.
func (p *Packet) PutString(s string) {
	for _, r := range s {
		cpoint := cubecode.FromUnicode(r)
		if cpoint == 0 {
			continue
		}
		p.PutInt(cpoint)
	}
	(*p) = append(*p, 0x00)
}

func (p *Packet) GetByte() (byte, bool) {
	if len(*p) < 1 {
		return 0, false
	}
	b := (*p)[0]
	(*p) = (*p)[1:]
	return b, true
}

// GetInt returns the integer value encoded in the next byte(s) of the packet.
func (p *Packet) GetInt() (int32, bool) {
	b, ok := p.GetByte()
	if !ok {
		return -1, false
	}

	switch b {
	default:
		// most often, the value is only one byte
		return int32(int8(b)), true
	case 0x80:
		// value is contained in the next two bytes
		if len(*p) < 2 {
			return -1, false
		}
		v := int32((*p)[0]) + int32(int8((*p)[1]))<<8
		(*p) = (*p)[2:]
		return v, true

	case 0x81:
		// value is contained in the next four bytes
		if len(*p) < 4 {
			return -1, false
		}
		v := int32((*p)[0]) + int32((*p)[1])<<8 + int32((*p)[2])<<16 + int32(int8((*p)[3]))<<24
		(*p) = (*p)[4:]
		return v, true
	}
}

func (p *Packet) GetUint() (v uint32, ok bool) {
	b, ok := p.GetByte()
	if !ok {
		return 0, false
	}
	v += uint32(b)

	if v&(1<<7) != 0 {
		b, ok := p.GetByte()
		if !ok {
			return 0, false
		}
		v += (uint32(b) << 7) - (1 << 7)
	}

	if v&(1<<14) != 0 {
		b, ok := p.GetByte()
		if !ok {
			return 0, false
		}
		v += (uint32(b) << 14) - (1 << 14)
	}

	if v&(1<<21) != 0 {
		b, ok := p.GetByte()
		if !ok {
			return 0, false
		}
		v += (uint32(b) << 14) - (1 << 14)
	}

	if v&(1<<28) != 0 {
		v += uint32(0xF) << 28 // fills up the top bits with ones to keep sign (to handle int32 inputs)
	}

	return v, true
}

// GetString returns a string of the next bytes up to 0x00.
func (p *Packet) GetString() (s string, ok bool) {
	var cpoint int32
	for {
		cpoint, ok = p.GetInt()
		if !ok {
			return s, false
		}
		if cpoint == 0 {
			return s, true
		}
		s += string(cubecode.ToUnicode(cpoint))
	}
}

