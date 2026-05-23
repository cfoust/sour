package maps

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	C "github.com/cfoust/sour/pkg/game/constants"
	V "github.com/cfoust/sour/pkg/game/variables"
)

// writer wraps a bytes.Buffer with helpers matching the C++ stream interface.
type writer struct {
	buf *bytes.Buffer
}

func newWriter() *writer {
	return &writer{buf: &bytes.Buffer{}}
}

func (w *writer) putchar(v byte) {
	w.buf.WriteByte(v)
}

func (w *writer) putuint16(v uint16) {
	binary.Write(w.buf, binary.LittleEndian, v)
}

func (w *writer) putint32(v int32) {
	binary.Write(w.buf, binary.LittleEndian, v)
}

func (w *writer) putfloat32(v float32) {
	binary.Write(w.buf, binary.LittleEndian, v)
}

func (w *writer) write(data []byte) {
	w.buf.Write(data)
}

func (w *writer) writeStruct(v interface{}) {
	binary.Write(w.buf, binary.LittleEndian, v)
}

// Encode serializes a GameMap to raw (uncompressed) OGZ binary data.
func Encode(m *GameMap) ([]byte, error) {
	w := newWriter()

	numVSlots := int32(len(m.VSlots))

	// Count non-empty entities
	numEnts := int32(len(m.Entities))

	// Count variables
	numVars := int32(len(m.Vars))

	// Write FileHeader
	hdr := FileHeader{
		Magic:      [4]byte{'O', 'C', 'T', 'A'},
		Version:    C.MAP_VERSION,
		HeaderSize: 40,
		WorldSize:  m.Header.WorldSize,
		NumEnts:    numEnts,
		NumPVs:     0,
		LightMaps:  m.Header.LightMaps,
	}
	w.writeStruct(hdr)

	// Write NewFooter
	footer := NewFooter{
		BlendMap:  m.Header.BlendMap,
		NumVars:   numVars,
		NumVSlots: numVSlots,
	}
	w.writeStruct(footer)

	// Write variables in sorted order for deterministic output
	varNames := make([]string, 0, len(m.Vars))
	for name := range m.Vars {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames)
	for _, name := range varNames {
		v := m.Vars[name]
		w.putchar(byte(v.Type()))
		w.putuint16(uint16(len(name)))
		w.write([]byte(name))
		switch v.Type() {
		case V.VariableTypeInt:
			w.putint32(int32(v.(V.IntVariable)))
		case V.VariableTypeFloat:
			w.putfloat32(float32(v.(V.FloatVariable)))
		case V.VariableTypeString:
			s := string(v.(V.StringVariable))
			w.putuint16(uint16(len(s)))
			w.write([]byte(s))
		}
	}

	// Write game type (C++ writes strlen as length byte, then strlen+1 bytes including null)
	gameType := m.Header.GameType
	if gameType == "" {
		gameType = "fps"
	}
	// Strip any existing null terminator
	for len(gameType) > 0 && gameType[len(gameType)-1] == 0 {
		gameType = gameType[:len(gameType)-1]
	}
	w.putchar(byte(len(gameType)))
	w.write([]byte(gameType))
	w.putchar(0) // null terminator

	// Extra entity info size and game data
	w.putuint16(0) // extraentinfosize
	w.putuint16(0) // game extras length

	// Texture MRU (empty)
	w.putuint16(0)

	// Write entities
	for _, ent := range m.Entities {
		w.writeStruct(ent)
	}

	// Write VSlots
	saveVSlots(w, m.VSlots)

	// Write octree
	if m.WorldRoot != nil && len(m.WorldRoot.Children) == CUBE_FACTOR {
		for _, child := range m.WorldRoot.Children {
			saveCube(w, child)
		}
	} else {
		// Write 8 empty cubes as fallback
		for i := 0; i < CUBE_FACTOR; i++ {
			w.putchar(OCTSAV_EMPTY)
			for j := 0; j < 6; j++ {
				w.putuint16(0)
			}
		}
	}

	// Write lightmaps (if any)
	if m.World != nil && m.Header.LightMaps > 0 {
		// Lightmap data would go here. For now we write none
		// (header.LightMaps should be 0 if we have none).
	}

	// No PVS data
	// No blendmap data (BlendMap should be 0)

	return w.buf.Bytes(), nil
}

// EncodeGZ serializes a GameMap to gzip-compressed OGZ data.
func EncodeGZ(m *GameMap) ([]byte, error) {
	raw, err := Encode(m)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// ToFile writes a GameMap to a gzip-compressed .ogz file.
func ToFile(m *GameMap, path string) error {
	data, err := EncodeGZ(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveVSlots writes VSlot data matching the C++ savevslots function.
func saveVSlots(w *writer, vslots []*VSlot) {
	if len(vslots) == 0 {
		return
	}

	numVSlots := len(vslots)

	// Build prev array (links changed vslots back to their base)
	prev := make([]int, numVSlots)
	for i := range prev {
		prev[i] = -1
	}
	for i := 0; i < numVSlots; i++ {
		vs := vslots[i]
		if vs.Changed != 0 {
			continue
		}
		cur := vs
		for {
			next := cur.Next
			if next == nil {
				break
			}
			// Skip vslots beyond our range
			for next != nil && next.Index >= int32(numVSlots) {
				next = next.Next
			}
			if next == nil {
				break
			}
			prev[next.Index] = int(cur.Index)
			cur = next
		}
	}

	lastRoot := 0
	for i := 0; i < numVSlots; i++ {
		vs := vslots[i]
		if vs.Changed == 0 {
			continue
		}
		if lastRoot < i {
			w.putint32(int32(-(i - lastRoot)))
		}
		saveVSlot(w, vs, prev[i])
		lastRoot = i + 1
	}
	if lastRoot < numVSlots {
		w.putint32(int32(-(numVSlots - lastRoot)))
	}
}

func saveVSlot(w *writer, vs *VSlot, prev int) {
	w.putint32(vs.Changed)
	w.putint32(int32(prev))

	if vs.Changed&(1<<VSLOT_SHPARAM) != 0 {
		w.putuint16(uint16(len(vs.Params)))
		for _, p := range vs.Params {
			w.putuint16(uint16(len(p.Name)))
			w.write([]byte(p.Name))
			for k := 0; k < 4; k++ {
				w.putfloat32(p.Val[k])
			}
		}
	}
	if vs.Changed&(1<<VSLOT_SCALE) != 0 {
		w.putfloat32(vs.Scale)
	}
	if vs.Changed&(1<<VSLOT_ROTATION) != 0 {
		w.putint32(vs.Rotation)
	}
	if vs.Changed&(1<<VSLOT_OFFSET) != 0 {
		w.putint32(vs.Offset.X)
		w.putint32(vs.Offset.Y)
	}
	if vs.Changed&(1<<VSLOT_SCROLL) != 0 {
		w.putfloat32(vs.Scroll.X)
		w.putfloat32(vs.Scroll.Y)
	}
	if vs.Changed&(1<<VSLOT_LAYER) != 0 {
		w.putint32(vs.Layer)
	}
	if vs.Changed&(1<<VSLOT_ALPHA) != 0 {
		w.putfloat32(vs.AlphaFront)
		w.putfloat32(vs.AlphaBack)
	}
	if vs.Changed&(1<<VSLOT_COLOR) != 0 {
		w.putfloat32(vs.ColorScale.X)
		w.putfloat32(vs.ColorScale.Y)
		w.putfloat32(vs.ColorScale.Z)
	}
}

// saveCube writes a single cube and its children, matching the C++ savec function.
func saveCube(w *writer, c *Cube) {
	if c == nil {
		w.putchar(OCTSAV_EMPTY)
		for i := 0; i < 6; i++ {
			w.putuint16(0)
		}
		return
	}

	if len(c.Children) == CUBE_FACTOR {
		w.putchar(OCTSAV_CHILDREN)
		for _, child := range c.Children {
			saveCube(w, child)
		}
		return
	}

	// Leaf cube
	oflags := byte(0)
	surfmask := byte(0)
	totalverts := byte(0)

	if c.Material != MAT_AIR {
		oflags |= 0x40
	}

	isEmpty := isEmptyCube(c)
	isSolid := isSolidCube(c)

	if !isEmpty {
		if c.Merged != 0 {
			oflags |= 0x80
		}
		// Check for surface data
		for j := 0; j < 6; j++ {
			surf := c.SurfaceInfo[j]
			if surf.Lmid[0] != 0 || surf.Lmid[1] != 0 || surf.Verts != 0 || surf.NumVerts != 0 {
				oflags |= 0x20
				surfmask |= 1 << uint(j)
				totalverts += surf.TotalVerts()
			}
		}
	}

	if isEmpty {
		w.putchar(oflags | OCTSAV_EMPTY)
	} else if isSolid {
		w.putchar(oflags | OCTSAV_SOLID)
	} else {
		w.putchar(oflags | OCTSAV_NORMAL)
		w.write(c.Edges[:])
	}

	// Write textures
	for i := 0; i < 6; i++ {
		w.putuint16(c.Texture[i])
	}

	// Write material
	if oflags&0x40 != 0 {
		w.putuint16(c.Material)
	}
	// Write merged
	if oflags&0x80 != 0 {
		w.putchar(c.Merged)
	}
	// Write surface info
	if oflags&0x20 != 0 {
		w.putchar(surfmask)
		w.putchar(totalverts)
		for j := 0; j < 6; j++ {
			if surfmask&(1<<uint(j)) == 0 {
				continue
			}
			surf := c.SurfaceInfo[j]
			w.writeStruct(surf)
			// Write per-vertex data if present
			if len(c.VertData[j]) > 0 {
				w.write(c.VertData[j])
			}
		}
	}
}

// isEmptyCube returns true if all edges are 0 (empty faces).
func isEmptyCube(c *Cube) bool {
	for _, e := range c.Edges {
		if e != 0 {
			return false
		}
	}
	return true
}

// isSolidCube returns true if all edges are 0x80 (entirely solid).
func isSolidCube(c *Cube) bool {
	for _, e := range c.Edges {
		if e != 0x80 {
			return false
		}
	}
	return true
}
