package osrs

import "math"

const RegionSize = 104

// Region holds parsed terrain data for a map region.
type Region struct {
	Heights             [4][RegionSize + 1][RegionSize + 1]int
	Flags               [4][RegionSize][RegionSize]byte
	Underlays           [4][RegionSize][RegionSize]byte
	Overlays            [4][RegionSize][RegionSize]byte
	OverlayTypes        [4][RegionSize][RegionSize]byte
	OverlayOrientations [4][RegionSize][RegionSize]byte
}

// PlacedObject is an object instance placed in a region.
type PlacedObject struct {
	ID       int
	Type     int // 0-22
	Rotation int // 0-3
	Plane    int
	LocalX   int
	LocalY   int
}

// LoadTerrain parses terrain data for a 64x64 chunk within the region.
// offsetX, offsetY are the chunk's position within the 104x104 region.
func (r *Region) LoadTerrain(data []byte, offsetX, offsetY int) error {
	buf := NewBuffer(data)
	for plane := 0; plane < 4; plane++ {
		for x := 0; x < 64; x++ {
			for y := 0; y < 64; y++ {
				tileX := x + offsetX
				tileY := y + offsetY
				if err := r.readTile(buf, tileX, tileY, plane); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *Region) readTile(buf *Buffer, x, y, plane int) error {
	inBounds := x >= 0 && x < RegionSize && y >= 0 && y < RegionSize

	if inBounds {
		r.Flags[plane][x][y] = 0
	}

	for {
		opcode, err := buf.ReadUnsignedByte()
		if err != nil {
			return err
		}

		if opcode == 0 {
			if inBounds {
				if plane == 0 {
					r.Heights[0][x][y] = -calculateVertexHeight(0xe3b7b+x, 0x87cce+y) * 8
				} else {
					r.Heights[plane][x][y] = r.Heights[plane-1][x][y] - 240
				}
			}
			return nil
		}

		if opcode == 1 {
			height, err := buf.ReadUnsignedByte()
			if err != nil {
				return err
			}
			if height == 1 {
				height = 0
			}
			if inBounds {
				if plane == 0 {
					r.Heights[0][x][y] = -height * 8
				} else {
					r.Heights[plane][x][y] = r.Heights[plane-1][x][y] - height*8
				}
			}
			return nil
		}

		if opcode <= 49 {
			if inBounds {
				overlay, err := buf.ReadSignedByte()
				if err != nil {
					return err
				}
				r.Overlays[plane][x][y] = byte(overlay)
				r.OverlayTypes[plane][x][y] = byte((opcode - 2) / 4)
				r.OverlayOrientations[plane][x][y] = byte((opcode - 2) & 3)
			} else {
				buf.Skip(1)
			}
		} else if opcode <= 81 {
			if inBounds {
				r.Flags[plane][x][y] = byte(opcode - 49)
			}
		} else {
			if inBounds {
				r.Underlays[plane][x][y] = byte(opcode - 81)
			}
		}
	}
}

// LoadObjects parses object placement data for a 64x64 chunk.
func LoadObjects(data []byte, offsetX, offsetY int) ([]PlacedObject, error) {
	buf := NewBuffer(data)
	var objects []PlacedObject

	objectID := -1
	for {
		delta, err := buf.ReadIncrementalSmart()
		if err != nil {
			return objects, nil // EOF is normal
		}
		if delta == 0 {
			break
		}
		objectID += delta

		tilePos := 0
		for {
			posDelta, err := buf.ReadUSmart()
			if err != nil {
				break
			}
			if posDelta == 0 {
				break
			}
			tilePos += posDelta - 1

			localY := tilePos & 0x3f
			localX := (tilePos >> 6) & 0x3f
			plane := tilePos >> 12

			config, err := buf.ReadUnsignedByte()
			if err != nil {
				break
			}
			objectType := config >> 2
			rotation := config & 3

			x := localX + offsetX
			y := localY + offsetY

			if x >= 0 && x < RegionSize && y >= 0 && y < RegionSize && plane >= 0 && plane < 4 {
				objects = append(objects, PlacedObject{
					ID:       objectID,
					Type:     objectType,
					Rotation: rotation,
					Plane:    plane,
					LocalX:   x,
					LocalY:   y,
				})
			}
		}
	}

	return objects, nil
}

// Perlin noise functions matching OSRS terrain generation.

func calculateVertexHeight(x, y int) int {
	noise := (interpolatedNoise(x+45365, y+0x16713, 4) - 128) +
		(interpolatedNoise(x+10294, y+37821, 2)-128)>>1 +
		(interpolatedNoise(x, y, 1)-128)>>2
	noise = int(float64(noise)*0.3) + 35
	if noise < 10 {
		noise = 10
	} else if noise > 60 {
		noise = 60
	}
	return noise
}

func interpolatedNoise(x, y, freq int) int {
	lx := x / freq
	fx := x & (freq - 1)
	ly := y / freq
	fy := y & (freq - 1)
	v00 := smoothNoise(lx, ly)
	v10 := smoothNoise(lx+1, ly)
	v01 := smoothNoise(lx, ly+1)
	v11 := smoothNoise(lx+1, ly+1)
	i0 := cosineInterpolate(v00, v10, fx, freq)
	i1 := cosineInterpolate(v01, v11, fx, freq)
	return cosineInterpolate(i0, i1, fy, freq)
}

// cosine lookup table matching Rasterizer3D.cosine
var cosineTable [2048]int

func init() {
	for i := 0; i < 2048; i++ {
		cosineTable[i] = int(65536.0 * math.Cos(float64(i)*0.0030679615))
	}
}

func cosineInterpolate(a, b, angle, freq int) int {
	cosine := 0x10000 - cosineTable[(angle*1024)/freq]
	cosine >>= 1
	return (a*(0x10000-cosine) >> 16) + (b*cosine >> 16)
}

func smoothNoise(x, y int) int {
	corners := calculateNoise(x-1, y-1) + calculateNoise(x+1, y-1) +
		calculateNoise(x-1, y+1) + calculateNoise(x+1, y+1)
	sides := calculateNoise(x-1, y) + calculateNoise(x+1, y) +
		calculateNoise(x, y-1) + calculateNoise(x, y+1)
	center := calculateNoise(x, y)
	return corners/16 + sides/8 + center/4
}

func calculateNoise(x, y int) int {
	k := x + y*57
	k = k<<13 ^ k
	l := k*(k*k*15731+0xc0ae5) + 0x5208dd0d
	l &= 0x7fffffff
	return l >> 19 & 0xff
}
