package osrs

import "fmt"

// ObjectDef describes an OSRS object's properties.
type ObjectDef struct {
	ID         int
	Name       string
	SizeX      int
	SizeY      int
	Solid      bool
	Walkable   bool
	Occludes   bool
	ModelIDs   []int
	ScaleX     int // default 128 = 1x
	ScaleY     int
	ScaleZ     int
	TranslateX int
	TranslateY int
	TranslateZ int
}

// ObjectDefs holds all object definitions loaded from the cache.
type ObjectDefs struct {
	defs    []ObjectDef
	indices []int
	data    []byte
}

// LoadObjectDefs parses loc.dat and loc.idx from a config archive.
func LoadObjectDefs(locDat, locIdx []byte) (*ObjectDefs, error) {
	idxBuf := NewBuffer(locIdx)
	count, err := idxBuf.ReadUShort()
	if err != nil {
		return nil, fmt.Errorf("reading object count: %w", err)
	}

	indices := make([]int, count)
	offset := 2
	for i := 0; i < count; i++ {
		indices[i] = offset
		size, err := idxBuf.ReadUShort()
		if err != nil {
			return nil, fmt.Errorf("reading index %d: %w", i, err)
		}
		offset += size
	}

	return &ObjectDefs{
		defs:    make([]ObjectDef, count),
		indices: indices,
		data:    locDat,
	}, nil
}

// Get returns the object definition for the given ID.
func (o *ObjectDefs) Get(id int) ObjectDef {
	if id < 0 || id >= len(o.indices) {
		return ObjectDef{SizeX: 1, SizeY: 1, Solid: true, Walkable: true}
	}

	def := &o.defs[id]
	if def.ID == id && def.Name != "" {
		return *def
	}

	// Decode on demand
	def.ID = id
	def.SizeX = 1
	def.SizeY = 1
	def.Solid = true
	def.Walkable = true
	def.ScaleX = 128
	def.ScaleY = 128
	def.ScaleZ = 128

	buf := NewBuffer(o.data)
	buf.SetPosition(o.indices[id])
	decodeObjectDef(buf, def)
	return *def
}

// Count returns the number of object definitions.
func (o *ObjectDefs) Count() int {
	return len(o.indices)
}

func decodeObjectDef(buf *Buffer, def *ObjectDef) {
	for {
		opcode, err := buf.ReadUnsignedByte()
		if err != nil || opcode == 0 {
			break
		}

		switch {
		case opcode == 1:
			length, _ := buf.ReadUnsignedByte()
			if length > 0 {
				def.ModelIDs = make([]int, length)
				for i := 0; i < length; i++ {
					def.ModelIDs[i], _ = buf.ReadUShort()
					buf.ReadUnsignedByte() // type
				}
			}
		case opcode == 2:
			def.Name, _ = buf.ReadString()
		case opcode == 5:
			length, _ := buf.ReadUnsignedByte()
			if length > 0 {
				def.ModelIDs = make([]int, length)
				for i := 0; i < length; i++ {
					def.ModelIDs[i], _ = buf.ReadUShort()
				}
			}
		case opcode == 14:
			def.SizeX, _ = buf.ReadUnsignedByte()
		case opcode == 15:
			def.SizeY, _ = buf.ReadUnsignedByte()
		case opcode == 17:
			def.Solid = false
			def.Walkable = false
		case opcode == 18:
			def.Walkable = false
		case opcode == 19:
			buf.ReadUnsignedByte()
		case opcode == 21, opcode == 22, opcode == 23:
			if opcode == 23 {
				def.Occludes = true
			}
		case opcode == 24:
			buf.ReadUShort()
		case opcode == 27:
			// nothing
		case opcode == 28:
			buf.ReadUnsignedByte()
		case opcode == 29:
			buf.ReadSignedByte()
		case opcode >= 30 && opcode < 35:
			buf.ReadString()
		case opcode == 39:
			buf.ReadSignedByte()
		case opcode == 40:
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i < length; i++ {
				buf.ReadUShort()
				buf.ReadUShort()
			}
		case opcode == 41:
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i < length; i++ {
				buf.ReadUShort()
				buf.ReadUShort()
			}
		case opcode == 62, opcode == 64:
			// flags
		case opcode == 65:
			def.ScaleX, _ = buf.ReadUShort()
		case opcode == 66:
			def.ScaleY, _ = buf.ReadUShort()
		case opcode == 67:
			def.ScaleZ, _ = buf.ReadUShort()
		case opcode == 68:
			buf.ReadUShort()
		case opcode == 69:
			buf.ReadUnsignedByte()
		case opcode == 70:
			def.TranslateX, _ = buf.ReadUShort()
		case opcode == 71:
			def.TranslateY, _ = buf.ReadUShort()
		case opcode == 72:
			def.TranslateZ, _ = buf.ReadUShort()
		case opcode == 73, opcode == 74:
			// flags
		case opcode == 75:
			buf.ReadUnsignedByte()
		case opcode == 77:
			buf.ReadUShort()
			buf.ReadUShort()
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i <= length; i++ {
				buf.ReadUShort()
			}
		case opcode == 78:
			buf.ReadUShort()
			buf.ReadUnsignedByte()
		case opcode == 79:
			buf.ReadUShort()
			buf.ReadUShort()
			buf.ReadUnsignedByte()
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i < length; i++ {
				buf.ReadUShort()
			}
		case opcode == 81:
			buf.ReadUnsignedByte()
		case opcode == 82:
			buf.ReadUShort()
		case opcode == 92:
			buf.ReadUShort()
			buf.ReadUShort()
			buf.ReadUShort()
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i <= length; i++ {
				buf.ReadUShort()
			}
		case opcode == 249:
			length, _ := buf.ReadUnsignedByte()
			for i := 0; i < length; i++ {
				isString, _ := buf.ReadUnsignedByte()
				buf.ReadTriByte()
				if isString == 1 {
					buf.ReadString()
				} else {
					buf.ReadInt()
				}
			}
		}
	}
}
