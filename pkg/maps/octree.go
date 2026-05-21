package maps

import (
	"encoding/binary"
	"fmt"
	"io"
)

// reader wraps an io.Reader with helpers matching the C++ stream interface.
type reader struct {
	r   io.Reader
	err error
	ver int32 // map version
}

func (r *reader) failed() bool { return r.err != nil }

func (r *reader) getchar() byte {
	if r.err != nil {
		return 0
	}
	var b [1]byte
	_, r.err = io.ReadFull(r.r, b[:])
	return b[0]
}

func (r *reader) getuint16() uint16 {
	if r.err != nil {
		return 0
	}
	var v uint16
	r.err = binary.Read(r.r, binary.LittleEndian, &v)
	return v
}

func (r *reader) getint32() int32 {
	if r.err != nil {
		return 0
	}
	var v int32
	r.err = binary.Read(r.r, binary.LittleEndian, &v)
	return v
}

func (r *reader) getfloat32() float32 {
	if r.err != nil {
		return 0
	}
	var v float32
	r.err = binary.Read(r.r, binary.LittleEndian, &v)
	return v
}

func (r *reader) read(buf []byte) {
	if r.err != nil {
		return
	}
	_, r.err = io.ReadFull(r.r, buf)
}

func (r *reader) skip(n int) {
	if r.err != nil {
		return
	}
	_, r.err = io.CopyN(io.Discard, r.r, int64(n))
}

// LoadVSlots reads VSlot data from the binary stream.
func LoadVSlots(rd *reader, numVSlots int) []*VSlot {
	vslots := make([]*VSlot, 0, numVSlots)
	prev := make([]int, numVSlots)
	for i := range prev {
		prev[i] = -1
	}

	remaining := numVSlots
	for remaining > 0 {
		changed := rd.getint32()
		if rd.failed() {
			break
		}

		if changed < 0 {
			// Batch of unchanged vslots
			count := int(-changed)
			for i := 0; i < count; i++ {
				vs := NewVSlot(nil, int32(len(vslots)))
				vslots = append(vslots, vs)
			}
			remaining += int(changed) // changed is negative
		} else {
			prevIdx := rd.getint32()
			idx := len(vslots)
			if idx < len(prev) {
				prev[idx] = int(prevIdx)
			}
			vs := NewVSlot(nil, int32(idx))
			loadVSlot(rd, vs, changed)
			vslots = append(vslots, vs)
			remaining--
		}
	}

	// Link up next pointers
	for i, vs := range vslots {
		if i < len(prev) && prev[i] >= 0 && prev[i] < len(vslots) {
			vslots[prev[i]].Next = vs
		}
	}

	return vslots
}

func loadVSlot(rd *reader, vs *VSlot, changed int32) {
	vs.Changed = changed

	if changed&(1<<VSLOT_SHPARAM) != 0 {
		numParams := int(rd.getuint16())
		vs.Params = make([]SlotShaderParam, numParams)
		for i := 0; i < numParams; i++ {
			nlen := int(rd.getuint16())
			nameBytes := make([]byte, nlen)
			rd.read(nameBytes)
			vs.Params[i].Name = string(nameBytes)
			vs.Params[i].Loc = -1
			for k := 0; k < 4; k++ {
				vs.Params[i].Val[k] = rd.getfloat32()
			}
		}
	}
	if changed&(1<<VSLOT_SCALE) != 0 {
		vs.Scale = rd.getfloat32()
	}
	if changed&(1<<VSLOT_ROTATION) != 0 {
		rot := rd.getint32()
		if rot < 0 {
			rot = 0
		}
		if rot > 7 {
			rot = 7
		}
		vs.Rotation = rot
	}
	if changed&(1<<VSLOT_OFFSET) != 0 {
		vs.Offset.X = rd.getint32()
		vs.Offset.Y = rd.getint32()
	}
	if changed&(1<<VSLOT_SCROLL) != 0 {
		vs.Scroll.X = rd.getfloat32()
		vs.Scroll.Y = rd.getfloat32()
	}
	if changed&(1<<VSLOT_LAYER) != 0 {
		vs.Layer = rd.getint32()
	}
	if changed&(1<<VSLOT_ALPHA) != 0 {
		vs.AlphaFront = rd.getfloat32()
		vs.AlphaBack = rd.getfloat32()
	}
	if changed&(1<<VSLOT_COLOR) != 0 {
		vs.ColorScale.X = rd.getfloat32()
		vs.ColorScale.Y = rd.getfloat32()
		vs.ColorScale.Z = rd.getfloat32()
	}
}

func convertOldMaterial(mat int) uint16 {
	return uint16(((mat & 7) << MATF_VOLUME_SHIFT) | (((mat >> 3) & 3) << MATF_CLIP_SHIFT) | (((mat >> 5) & 7) << MATF_FLAG_SHIFT))
}

// LoadChildren reads 8 child cubes from the binary stream.
func LoadChildren(rd *reader, size int) ([]*Cube, error) {
	cubes := make([]*Cube, CUBE_FACTOR)
	for i := 0; i < CUBE_FACTOR; i++ {
		cubes[i] = &Cube{}
		if err := loadCube(rd, cubes[i], size); err != nil {
			return nil, err
		}
		if rd.failed() {
			return nil, fmt.Errorf("read error loading cube: %w", rd.err)
		}
	}
	return cubes, nil
}

func loadCube(rd *reader, c *Cube, size int) error {
	hasChildren := false
	octsav := rd.getchar()
	if rd.failed() {
		return rd.err
	}

	switch octsav & 0x7 {
	case OCTSAV_CHILDREN:
		children, err := LoadChildren(rd, size>>1)
		if err != nil {
			return err
		}
		c.Children = children
		return nil

	case OCTSAV_LODCUB:
		hasChildren = true

	case OCTSAV_EMPTY:
		c.EmptyFaces()

	case OCTSAV_SOLID:
		c.SolidFaces()

	case OCTSAV_NORMAL:
		rd.read(c.Edges[:])

	default:
		return fmt.Errorf("invalid octsav type: %d", octsav&0x7)
	}

	// Read textures
	for i := 0; i < 6; i++ {
		if rd.ver < 14 {
			c.Texture[i] = uint16(rd.getchar())
		} else {
			c.Texture[i] = rd.getuint16()
		}
	}

	if rd.ver < 7 {
		rd.skip(3)
	} else if rd.ver <= 31 {
		// Old format with surface/normal/merge compat structures
		mask := rd.getchar()

		if mask&0x80 != 0 {
			mat := int(rd.getchar())
			if rd.ver < 27 {
				matConv := []uint16{MAT_AIR, MAT_WATER, MAT_CLIP, MAT_GLASS | MAT_CLIP, MAT_NOCLIP, MAT_LAVA | MAT_DEATH, MAT_GAMECLIP, MAT_DEATH}
				if mat < len(matConv) {
					c.Material = matConv[mat]
				} else {
					c.Material = MAT_AIR
				}
			} else {
				c.Material = convertOldMaterial(mat)
			}
		}

		// Read and skip old surface/normal/merge data
		numSurfs := 6
		if mask&0x3F != 0 {
			for i := 0; i < numSurfs; i++ {
				if i >= 6 || mask&(1<<uint(i)) != 0 {
					// Read surfacecompat (14 bytes)
					var surf SurfaceCompat
					binary.Read(rd.r, binary.LittleEndian, &surf)
					if i < 6 {
						if mask&0x40 != 0 {
							// Read normalscompat (12 bytes)
							rd.skip(12)
						}
						if surf.Layer&2 != 0 {
							numSurfs++
						}
					}
				}
			}
		}

		if rd.ver >= 20 {
			if octsav&0x80 != 0 {
				merged := rd.getchar()
				c.Merged = merged & 0x3F
				if merged&0x80 != 0 {
					mmask := rd.getchar()
					if mmask != 0 {
						for i := 0; i < 6; i++ {
							if mmask&(1<<uint(i)) != 0 {
								// Read mergecompat (8 bytes)
								rd.skip(8)
							}
						}
					}
				}
			}
		}
	} else {
		// Current format (version >= 32)
		if octsav&0x40 != 0 {
			if rd.ver <= 32 {
				mat := int(rd.getchar())
				c.Material = convertOldMaterial(mat)
			} else {
				c.Material = rd.getuint16()
			}
		}
		if octsav&0x80 != 0 {
			c.Merged = rd.getchar()
		}
		if octsav&0x20 != 0 {
			surfmask := rd.getchar()
			totalverts := rd.getchar()
			if totalverts > 0 {
				// We need to store surface info
				for i := 0; i < 6; i++ {
					if surfmask&(1<<uint(i)) != 0 {
						// Read surfaceinfo (4 bytes)
						var surf SurfaceInfo
						binary.Read(rd.r, binary.LittleEndian, &surf)
						c.SurfaceInfo[i] = surf

						vertmask := surf.Verts
						numverts := surf.TotalVerts()
						if numverts == 0 {
							continue
						}

						layerverts := int(surf.NumVerts & MAXFACEVERTS)

						hasxyz := vertmask&0x04 != 0
						hasuv := vertmask&0x40 != 0
						hasnorm := vertmask&0x80 != 0

						if layerverts == 4 {
							if hasxyz && vertmask&0x01 != 0 {
								rd.skip(8) // 4 uint16s
								hasxyz = false
							}
							if hasuv && vertmask&0x02 != 0 {
								rd.skip(8) // 4 uint16s
								if surf.NumVerts&LAYER_DUP != 0 {
									rd.skip(8) // 4 more uint16s
								}
								hasuv = false
							}
						}
						if hasnorm && vertmask&0x08 != 0 {
							rd.skip(2) // 1 uint16
							hasnorm = false
						}
						if hasxyz || hasuv || hasnorm {
							bytesPerVert := 0
							if hasxyz {
								bytesPerVert += 4 // 2 uint16s
							}
							if hasuv {
								bytesPerVert += 4 // 2 uint16s
							}
							if hasnorm {
								bytesPerVert += 2 // 1 uint16
							}
							rd.skip(bytesPerVert * layerverts)
						}
						if surf.NumVerts&LAYER_DUP != 0 {
							dupBytes := 0
							if hasuv {
								dupBytes += 4 // 2 uint16s per vert
							}
							rd.skip(dupBytes * layerverts)
						}

						_ = numverts
					}
				}
			}
		}
	}

	if hasChildren {
		children, err := LoadChildren(rd, size>>1)
		if err != nil {
			return err
		}
		c.Children = children
	}

	return nil
}

// LoadWorld reads the full world data (vslots + octree + lightmaps etc) from a buffer.
// This replaces the C++ partial_load_world function.
func LoadWorld(data []byte, numVSlots int, worldSize int, mapVersion int32, numLightMaps int, numPVs int, blendMap int) (*WorldState, error) {
	rd := &reader{
		r:   newByteReader(data),
		ver: mapVersion,
	}

	// Load VSlots
	vslots := LoadVSlots(rd, numVSlots)
	if rd.failed() {
		return nil, fmt.Errorf("failed to load vslots: %w", rd.err)
	}

	// Load octree
	children, err := LoadChildren(rd, worldSize>>1)
	if err != nil {
		return nil, fmt.Errorf("failed to load cubes: %w", err)
	}

	root := &Cube{Children: children}

	// Skip lightmaps
	if mapVersion >= 7 {
		for i := 0; i < numLightMaps; i++ {
			bpp := 3
			if mapVersion >= 17 {
				lmtype := rd.getchar()
				if mapVersion >= 20 && lmtype&0x80 != 0 {
					rd.skip(4) // unlitx + unlity
				}
				if (lmtype&0x7F)&2 != 0 && (lmtype&0x7F)&0x0E != 4 {
					bpp = 4
				}
			}
			rd.skip(bpp * 512 * 512) // LM_PACKW * LM_PACKH
		}
	}

	// We skip PVS and blendmap data — not needed for asset extraction

	state := &WorldState{
		VSlots:   vslots,
		Root:     root,
		WorldSize: worldSize,
	}

	return state, nil
}

// WorldState holds the loaded world data, replacing the C++ MapState.
type WorldState struct {
	VSlots    []*VSlot
	Root      *Cube
	WorldSize int
}

// CountVSlotRefs walks the octree and counts how many times each VSlot index
// appears on cube faces. This replaces the C++ getrefs function.
func (ws *WorldState) CountVSlotRefs(numSlots int) []int32 {
	refs := make([]int32, numSlots)
	if ws.Root != nil {
		for _, child := range ws.Root.Children {
			countCubeRefs(child, refs, numSlots)
		}
	}
	return refs
}

func countCubeRefs(c *Cube, refs []int32, numSlots int) {
	if c == nil {
		return
	}
	if len(c.Children) > 0 {
		for _, child := range c.Children {
			countCubeRefs(child, refs, numSlots)
		}
		return
	}
	for i := 0; i < 6; i++ {
		t := int(c.Texture[i])
		if t >= 0 && t < numSlots {
			refs[t]++
		}
	}
}

// byteReader wraps a byte slice as an io.Reader.
type byteReader struct {
	data []byte
	pos  int
}

func newByteReader(data []byte) *byteReader {
	return &byteReader{data: data}
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}
