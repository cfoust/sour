package preview

import (
	"encoding/binary"
	"fmt"
)

var magic = [4]byte{'S', 'V', 'O', 'X'}

const (
	headerSize    = 32
	voxelBytes    = 9 // X(2) + Y(2) + Z(2) + PaletteIndex(1) + Flags(1) + AO(1)
	entityBytes   = 7 // X(2) + Y(2) + Z(2) + Type(1)
)

// Encode serializes a MapPreview to the .svox binary format (version 2).
func Encode(p *MapPreview) ([]byte, error) {
	numPalette := len(p.Palette)
	numVoxels := len(p.Voxels)
	numEntities := len(p.Entities)

	size := headerSize + numPalette*3 + numVoxels*voxelBytes + numEntities*entityBytes
	buf := make([]byte, size)

	// Header
	copy(buf[0:4], magic[:])
	buf[4] = 2 // version 2: uint16 coordinates
	buf[5] = p.MaxDepth
	binary.LittleEndian.PutUint16(buf[6:8], p.GridSize)
	binary.LittleEndian.PutUint32(buf[8:12], p.WorldSize)
	binary.LittleEndian.PutUint16(buf[12:14], uint16(numPalette))
	binary.LittleEndian.PutUint32(buf[14:18], uint32(numVoxels))
	binary.LittleEndian.PutUint16(buf[18:20], uint16(numEntities))

	// Sky colors
	copy(buf[20:23], p.SkyTop[:])
	copy(buf[23:26], p.SkyHorizon[:])
	copy(buf[26:29], p.Ambient[:])
	copy(buf[29:32], p.Sunlight[:])

	off := headerSize

	// Palette
	for _, c := range p.Palette {
		buf[off] = c[0]
		buf[off+1] = c[1]
		buf[off+2] = c[2]
		off += 3
	}

	// Voxels: X(u16) Y(u16) Z(u16) PaletteIndex(u8) Flags(u8) AO(u8)
	for _, v := range p.Voxels {
		binary.LittleEndian.PutUint16(buf[off:], v.X)
		binary.LittleEndian.PutUint16(buf[off+2:], v.Y)
		binary.LittleEndian.PutUint16(buf[off+4:], v.Z)
		buf[off+6] = v.PaletteIndex
		buf[off+7] = v.Flags
		buf[off+8] = v.AO
		off += voxelBytes
	}

	// Entities: X(u16) Y(u16) Z(u16) Type(u8)
	for _, e := range p.Entities {
		binary.LittleEndian.PutUint16(buf[off:], e.X)
		binary.LittleEndian.PutUint16(buf[off+2:], e.Y)
		binary.LittleEndian.PutUint16(buf[off+4:], e.Z)
		buf[off+6] = byte(e.Type)
		off += entityBytes
	}

	return buf, nil
}

// Decode parses .svox binary data into a MapPreview.
func Decode(data []byte) (*MapPreview, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("data too short for header")
	}
	if data[0] != 'S' || data[1] != 'V' || data[2] != 'O' || data[3] != 'X' {
		return nil, fmt.Errorf("invalid magic bytes")
	}

	version := data[4]
	if version != 2 {
		return nil, fmt.Errorf("unsupported version: %d", version)
	}

	p := &MapPreview{}
	p.MaxDepth = data[5]
	p.GridSize = binary.LittleEndian.Uint16(data[6:8])
	p.WorldSize = binary.LittleEndian.Uint32(data[8:12])

	numPalette := int(binary.LittleEndian.Uint16(data[12:14]))
	numVoxels := int(binary.LittleEndian.Uint32(data[14:18]))
	numEntities := int(binary.LittleEndian.Uint16(data[18:20]))

	copy(p.SkyTop[:], data[20:23])
	copy(p.SkyHorizon[:], data[23:26])
	copy(p.Ambient[:], data[26:29])
	copy(p.Sunlight[:], data[29:32])

	expected := headerSize + numPalette*3 + numVoxels*voxelBytes + numEntities*entityBytes
	if len(data) < expected {
		return nil, fmt.Errorf("data too short: need %d, got %d", expected, len(data))
	}

	off := headerSize

	p.Palette = make([][3]uint8, numPalette)
	for i := range p.Palette {
		p.Palette[i] = [3]uint8{data[off], data[off+1], data[off+2]}
		off += 3
	}

	p.Voxels = make([]Voxel, numVoxels)
	for i := range p.Voxels {
		p.Voxels[i] = Voxel{
			X:            binary.LittleEndian.Uint16(data[off:]),
			Y:            binary.LittleEndian.Uint16(data[off+2:]),
			Z:            binary.LittleEndian.Uint16(data[off+4:]),
			PaletteIndex: data[off+6],
			Flags:        data[off+7],
			AO:           data[off+8],
		}
		off += voxelBytes
	}

	p.Entities = make([]PreviewEntity, numEntities)
	for i := range p.Entities {
		p.Entities[i] = PreviewEntity{
			X:    binary.LittleEndian.Uint16(data[off:]),
			Y:    binary.LittleEndian.Uint16(data[off+2:]),
			Z:    binary.LittleEndian.Uint16(data[off+4:]),
			Type: PreviewEntityType(data[off+6]),
		}
		off += entityBytes
	}

	return p, nil
}
