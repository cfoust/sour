package preview

import (
	"encoding/binary"
	"fmt"
)

var magic = [4]byte{'S', 'V', 'O', 'X'}

const headerSize = 32

// Encode serializes a MapPreview to the .svox binary format.
func Encode(p *MapPreview) ([]byte, error) {
	numPalette := len(p.Palette)
	numVoxels := len(p.Voxels)
	numEntities := len(p.Entities)

	if numPalette > 65535 || numVoxels > 4294967295 || numEntities > 65535 {
		return nil, fmt.Errorf("preview data exceeds format limits")
	}

	size := headerSize + numPalette*3 + numVoxels*6 + numEntities*4
	buf := make([]byte, size)

	// Header
	copy(buf[0:4], magic[:])
	buf[4] = 1 // version
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

	// Voxels
	for _, v := range p.Voxels {
		buf[off] = v.X
		buf[off+1] = v.Y
		buf[off+2] = v.Z
		buf[off+3] = v.PaletteIndex
		buf[off+4] = v.Flags
		buf[off+5] = v.AO
		off += 6
	}

	// Entities
	for _, e := range p.Entities {
		buf[off] = e.X
		buf[off+1] = e.Y
		buf[off+2] = e.Z
		buf[off+3] = byte(e.Type)
		off += 4
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
	if data[4] != 1 {
		return nil, fmt.Errorf("unsupported version: %d", data[4])
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

	expected := headerSize + numPalette*3 + numVoxels*6 + numEntities*4
	if len(data) < expected {
		return nil, fmt.Errorf("data too short: need %d, got %d", expected, len(data))
	}

	off := headerSize

	// Palette
	p.Palette = make([][3]uint8, numPalette)
	for i := range p.Palette {
		p.Palette[i] = [3]uint8{data[off], data[off+1], data[off+2]}
		off += 3
	}

	// Voxels
	p.Voxels = make([]Voxel, numVoxels)
	for i := range p.Voxels {
		p.Voxels[i] = Voxel{
			X:            data[off],
			Y:            data[off+1],
			Z:            data[off+2],
			PaletteIndex: data[off+3],
			Flags:        data[off+4],
			AO:           data[off+5],
		}
		off += 6
	}

	// Entities
	p.Entities = make([]PreviewEntity, numEntities)
	for i := range p.Entities {
		p.Entities[i] = PreviewEntity{
			X:    data[off],
			Y:    data[off+1],
			Z:    data[off+2],
			Type: PreviewEntityType(data[off+3]),
		}
		off += 4
	}

	return p, nil
}
