package osrs

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Model holds decoded OSRS 3D model data.
type Model struct {
	VertexX    []int
	VertexY    []int
	VertexZ    []int
	FaceA      []int   // triangle vertex indices
	FaceB      []int
	FaceC      []int
	Colors     []int16 // packed HSL16 per triangle
	TextureIDs []int16 // texture ID per face (-1 = no texture, use color)
}

// DecodeModel decodes an OSRS model from raw cache data (old format only for now).
func DecodeModel(data []byte) (*Model, error) {
	if len(data) < 18 {
		return nil, fmt.Errorf("model data too short: %d bytes", len(data))
	}

	// Detect format: new format has last 2 bytes == 0xFFFF
	if data[len(data)-1] == 0xFF && data[len(data)-2] == 0xFF {
		return decodeModelNew(data)
	}
	return decodeModelOld(data)
}

func decodeModelOld(data []byte) (*Model, error) {
	buf := NewBuffer(data)

	// Read header from end of data (last 18 bytes)
	buf.SetPosition(len(data) - 18)
	vertexCount, _ := buf.ReadUShort()
	triangleCount, _ := buf.ReadUShort()
	texTriCount, _ := buf.ReadUnsignedByte()
	renderTypeOpcode, _ := buf.ReadUnsignedByte()
	priorityOpcode, _ := buf.ReadUnsignedByte()
	alphaOpcode, _ := buf.ReadUnsignedByte()
	tSkinOpcode, _ := buf.ReadUnsignedByte()
	vSkinOpcode, _ := buf.ReadUnsignedByte()
	vertexXSize, _ := buf.ReadUShort()
	vertexYSize, _ := buf.ReadUShort()
	_, _ = buf.ReadUShort() // vertexZSize
	vertexPointsSize, _ := buf.ReadUShort()

	if vertexCount == 0 || triangleCount == 0 {
		return nil, fmt.Errorf("empty model: %d vertices, %d triangles", vertexCount, triangleCount)
	}

	// Calculate section offsets (matches Java decodeOld exactly)
	pos := 0
	vertexFlagOff := pos
	pos += vertexCount
	faceCompressOff := pos
	pos += triangleCount
	// facePriority
	if priorityOpcode == 255 {
		pos += triangleCount
	}
	// tSkin
	if tSkinOpcode == 1 {
		pos += triangleCount
	}
	renderTypeOff := pos
	if renderTypeOpcode == 1 {
		pos += triangleCount
	}
	// vSkin
	if vSkinOpcode == 1 {
		pos += vertexCount
	}
	// alpha
	if alphaOpcode == 1 {
		pos += triangleCount
	}
	pointsOff := pos
	pos += vertexPointsSize
	colorOff := pos
	pos += triangleCount * 2
	_ = texTriCount
	pos += texTriCount * 6
	vertexXOff := pos
	pos += vertexXSize
	vertexYOff := pos
	pos += vertexYSize
	vertexZOff := pos

	m := &Model{
		VertexX:    make([]int, vertexCount),
		VertexY:    make([]int, vertexCount),
		VertexZ:    make([]int, vertexCount),
		FaceA:      make([]int, triangleCount),
		FaceB:      make([]int, triangleCount),
		FaceC:      make([]int, triangleCount),
		Colors:     make([]int16, triangleCount),
		TextureIDs: make([]int16, triangleCount),
	}

	// Initialize texture IDs to -1 (no texture)
	for i := range m.TextureIDs {
		m.TextureIDs[i] = -1
	}

	// Read vertices (delta-encoded with smart compression)
	flagBuf := NewBuffer(data)
	flagBuf.SetPosition(vertexFlagOff)
	xBuf := NewBuffer(data)
	xBuf.SetPosition(vertexXOff)
	yBuf := NewBuffer(data)
	yBuf.SetPosition(vertexYOff)
	zBuf := NewBuffer(data)
	zBuf.SetPosition(vertexZOff)

	var sx, sy, sz int
	for i := 0; i < vertexCount; i++ {
		mask, _ := flagBuf.ReadUnsignedByte()
		var dx, dy, dz int
		if mask&0x1 != 0 {
			dx, _ = xBuf.ReadSmart()
		}
		if mask&0x2 != 0 {
			dy, _ = yBuf.ReadSmart()
		}
		if mask&0x4 != 0 {
			dz, _ = zBuf.ReadSmart()
		}
		sx += dx
		sy += dy
		sz += dz
		m.VertexX[i] = sx
		m.VertexY[i] = sy
		m.VertexZ[i] = sz
	}

	// Read colors and texture info
	colorBuf := NewBuffer(data)
	colorBuf.SetPosition(colorOff)
	var rtBuf *Buffer
	if renderTypeOpcode == 1 {
		rtBuf = NewBuffer(data)
		rtBuf.SetPosition(renderTypeOff)
	}
	for i := 0; i < triangleCount; i++ {
		c, _ := colorBuf.ReadUShort()
		m.Colors[i] = int16(c)
		if rtBuf != nil {
			flag, _ := rtBuf.ReadUnsignedByte()
			if flag&0x2 != 0 {
				m.TextureIDs[i] = m.Colors[i]
				m.Colors[i] = 127
			}
		}
	}

	// Read face indices (compressed triangle strip)
	idxBuf := NewBuffer(data)
	idxBuf.SetPosition(pointsOff)
	compBuf := NewBuffer(data)
	compBuf.SetPosition(faceCompressOff)

	var a, b, c, offset int
	for i := 0; i < triangleCount; i++ {
		opcode, _ := compBuf.ReadUnsignedByte()
		switch opcode {
		case 1:
			v, _ := idxBuf.ReadSmart()
			a = v + offset
			offset = a
			v, _ = idxBuf.ReadSmart()
			b = v + offset
			offset = b
			v, _ = idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 2:
			b = c
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 3:
			a = c
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 4:
			a, b = b, a
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		}
		m.FaceA[i] = a
		m.FaceB[i] = b
		m.FaceC[i] = c
	}

	return m, nil
}

func decodeModelNew(data []byte) (*Model, error) {
	// New format has 23-byte header at end and extra texture fields
	// For simplicity, use the same approach but with the extended header
	buf := NewBuffer(data)
	buf.SetPosition(len(data) - 23)
	vertexCount, _ := buf.ReadUShort()
	triangleCount, _ := buf.ReadUShort()
	texTriCount, _ := buf.ReadUnsignedByte()
	renderTypeOpcode, _ := buf.ReadUnsignedByte()
	priorityOpcode, _ := buf.ReadUnsignedByte()
	alphaOpcode, _ := buf.ReadUnsignedByte()
	tSkinOpcode, _ := buf.ReadUnsignedByte()
	_ /* textureOpcode */, _ = buf.ReadUnsignedByte()
	vSkinOpcode, _ := buf.ReadUnsignedByte()
	vertexXSize, _ := buf.ReadUShort()
	vertexYSize, _ := buf.ReadUShort()
	_, _ = buf.ReadUShort() // vertexZSize
	vertexPointsSize, _ := buf.ReadUShort()
	_ /* textureIndicesSize */, _ = buf.ReadUShort()

	if vertexCount == 0 || triangleCount == 0 {
		return nil, fmt.Errorf("empty model: %d vertices, %d triangles", vertexCount, triangleCount)
	}

	// Calculate offsets (same structure as old, just more fields)
	pos := 0
	vertexFlagOff := pos
	pos += vertexCount
	faceCompressOff := pos
	pos += triangleCount
	if priorityOpcode == 255 {
		pos += triangleCount
	}
	if tSkinOpcode == 1 {
		pos += triangleCount
	}
	if renderTypeOpcode == 1 {
		pos += triangleCount
	}
	if vSkinOpcode == 1 {
		pos += vertexCount
	}
	if alphaOpcode == 1 {
		pos += triangleCount
	}
	pointsOff := pos
	pos += vertexPointsSize
	colorOff := pos
	pos += triangleCount * 2
	_ = texTriCount
	pos += texTriCount * 6
	vertexXOff := pos
	pos += vertexXSize
	vertexYOff := pos
	pos += vertexYSize
	vertexZOff := pos

	m := &Model{
		VertexX: make([]int, vertexCount),
		VertexY: make([]int, vertexCount),
		VertexZ: make([]int, vertexCount),
		FaceA:   make([]int, triangleCount),
		FaceB:   make([]int, triangleCount),
		FaceC:   make([]int, triangleCount),
		Colors:  make([]int16, triangleCount),
	}

	// Read vertices
	flagBuf := NewBuffer(data)
	flagBuf.SetPosition(vertexFlagOff)
	xBuf := NewBuffer(data)
	xBuf.SetPosition(vertexXOff)
	yBuf := NewBuffer(data)
	yBuf.SetPosition(vertexYOff)
	zBuf := NewBuffer(data)
	zBuf.SetPosition(vertexZOff)

	var sx, sy, sz int
	for i := 0; i < vertexCount; i++ {
		mask, _ := flagBuf.ReadUnsignedByte()
		var dx, dy, dz int
		if mask&0x1 != 0 {
			dx, _ = xBuf.ReadSmart()
		}
		if mask&0x2 != 0 {
			dy, _ = yBuf.ReadSmart()
		}
		if mask&0x4 != 0 {
			dz, _ = zBuf.ReadSmart()
		}
		sx += dx
		sy += dy
		sz += dz
		m.VertexX[i] = sx
		m.VertexY[i] = sy
		m.VertexZ[i] = sz
	}

	// Read colors
	colorBuf := NewBuffer(data)
	colorBuf.SetPosition(colorOff)
	for i := 0; i < triangleCount; i++ {
		c, _ := colorBuf.ReadUShort()
		m.Colors[i] = int16(c)
	}

	// Read face indices
	idxBuf := NewBuffer(data)
	idxBuf.SetPosition(pointsOff)
	compBuf := NewBuffer(data)
	compBuf.SetPosition(faceCompressOff)

	var a, b, c, offset int
	for i := 0; i < triangleCount; i++ {
		opcode, _ := compBuf.ReadUnsignedByte()
		switch opcode {
		case 1:
			v, _ := idxBuf.ReadSmart()
			a = v + offset
			offset = a
			v, _ = idxBuf.ReadSmart()
			b = v + offset
			offset = b
			v, _ = idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 2:
			b = c
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 3:
			a = c
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		case 4:
			a, b = b, a
			v, _ := idxBuf.ReadSmart()
			c = v + offset
			offset = c
		}
		m.FaceA[i] = a
		m.FaceB[i] = b
		m.FaceC[i] = c
	}

	return m, nil
}

// HSL16ToRGB converts OSRS packed HSL16 color to RGB.
// HSL16 format: (hue/4 << 10) | (sat/32 << 7) | (lum/2)
func HSL16ToRGB(hsl16 int16) color.RGBA {
	h := float64((hsl16>>10)&0x3F) / 64.0
	s := float64((hsl16>>7)&0x07) / 8.0
	l := float64(hsl16&0x7F) / 128.0

	var r, g, b float64
	if s == 0 {
		r, g, b = l, l, l
	} else {
		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hueToRGB(p, q, h+1.0/3.0)
		g = hueToRGB(p, q, h)
		b = hueToRGB(p, q, h-1.0/3.0)
	}

	return color.RGBA{
		R: uint8(math.Min(255, r*256)),
		G: uint8(math.Min(255, g*256)),
		B: uint8(math.Min(255, b*256)),
		A: 255,
	}
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if 6*t < 1 {
		return p + (q-p)*6*t
	}
	if 2*t < 1 {
		return q
	}
	if 3*t < 2 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// Rotate90 rotates the model 90 degrees clockwise (viewed from above).
// Matches OSRS Model.rotate90Degrees(): (x,z) → (z,-x)
func (m *Model) Rotate90() {
	for i := range m.VertexX {
		x := m.VertexX[i]
		m.VertexX[i] = m.VertexZ[i]
		m.VertexZ[i] = -x
	}
}

// ApplyObjectDef applies the ObjectDefinition's scale and translate to vertices.
// Matches OSRS: scale(scaleX, scaleZ, scaleY) then translate(x, y, z).
// Note the argument order swap: Java calls scale(scaleX, scaleZ, scaleY).
func (m *Model) ApplyObjectDef(def ObjectDef) {
	if def.ScaleX != 128 || def.ScaleY != 128 || def.ScaleZ != 128 {
		for i := range m.VertexX {
			m.VertexX[i] = (m.VertexX[i] * def.ScaleX) / 128
			m.VertexY[i] = (m.VertexY[i] * def.ScaleY) / 128
			m.VertexZ[i] = (m.VertexZ[i] * def.ScaleZ) / 128
		}
	}
	if def.TranslateX != 0 || def.TranslateY != 0 || def.TranslateZ != 0 {
		for i := range m.VertexX {
			m.VertexX[i] += def.TranslateX
			m.VertexY[i] += def.TranslateY
			m.VertexZ[i] += def.TranslateZ
		}
	}
}

// AverageColor returns the average color across all faces.
func (m *Model) AverageColor() color.RGBA {
	if len(m.Colors) == 0 {
		return color.RGBA{R: 128, G: 128, B: 128, A: 255}
	}
	var r, g, b int
	for _, c := range m.Colors {
		rgb := HSL16ToRGB(c)
		r += int(rgb.R)
		g += int(rgb.G)
		b += int(rgb.B)
	}
	n := len(m.Colors)
	return color.RGBA{
		R: uint8(r / n),
		G: uint8(g / n),
		B: uint8(b / n),
		A: 255,
	}
}

// ToOBJ exports the model as a Wavefront OBJ with UV coordinates mapped to a
// color atlas texture. Returns the OBJ content string.
// The atlas is a 1-pixel-tall image where each pixel is a unique face color.
func (m *Model) ToOBJ(name string, scale float64) (string, string) {
	// Collect unique colors and assign atlas positions
	colorSet := make(map[int16]int)
	var colorList []int16
	for _, c := range m.Colors {
		if _, ok := colorSet[c]; !ok {
			colorSet[c] = len(colorList)
			colorList = append(colorList, c)
		}
	}

	n := len(colorList)
	if n == 0 {
		n = 1
	}
	atlasSide := 1
	for atlasSide*atlasSide < n {
		atlasSide *= 2
	}
	if atlasSide < 4 {
		atlasSide = 4
	}

	// Build OBJ with UVs
	obj := fmt.Sprintf("# OSRS model %s\n\n", name)

	// OSRS coords: X=east, Y=height(neg=up), Z=south
	// OBJ convention: Y=up
	// Sauer OBJ loader remaps: OBJ(x,y,z) → Sauer(z,-x,y)
	// So: OBJ-X → Sauer-(-Y), OBJ-Y → Sauer-Z(up), OBJ-Z → Sauer-X
	// We output standard OBJ Y-up. Don't swap axes — let entity yaw handle rotation.
	for i := 0; i < len(m.VertexX); i++ {
		x := float64(m.VertexX[i]) * scale
		y := float64(-m.VertexY[i]) * scale // negate: OSRS Y is inverted
		z := float64(-m.VertexZ[i]) * scale // negate Z to fix handedness
		obj += fmt.Sprintf("v %f %f %f\n", x, y, z)
	}

	obj += "\n"

	// UV coordinates: one per face, pointing to the color's pixel in the grid atlas
	for i := 0; i < len(m.FaceA); i++ {
		colIdx := colorSet[m.Colors[i]]
		cx := colIdx % atlasSide
		cy := colIdx / atlasSide
		u := (float64(cx) + 0.5) / float64(atlasSide)
		v := 1.0 - (float64(cy)+0.5)/float64(atlasSide) // flip V for OBJ convention
		obj += fmt.Sprintf("vt %f %f\n", u, v)
	}

	obj += "\n"

	// Faces with UV indices — ABC order.
	// The Z negation in vertex output flips handedness, which combined with
	// Sauer's OBJ remap (which also flips) gives correct front-face winding.
	for i := 0; i < len(m.FaceA); i++ {
		uvIdx := i + 1 // 1-based
		obj += fmt.Sprintf("f %d/%d %d/%d %d/%d\n",
			m.FaceA[i]+1, uvIdx,
			m.FaceB[i]+1, uvIdx,
			m.FaceC[i]+1, uvIdx)
	}

	// We don't use MTL anymore — colors come from the atlas texture
	return obj, ""
}

// ColorAtlas generates a color atlas image for the model.
// Returns a 1-pixel-tall image where each pixel is a unique face color.
func (m *Model) ColorAtlas() *image.RGBA {
	colorSet := make(map[int16]int)
	var colorList []int16
	for _, c := range m.Colors {
		if _, ok := colorSet[c]; !ok {
			colorSet[c] = len(colorList)
			colorList = append(colorList, c)
		}
	}

	n := len(colorList)
	if n == 0 {
		n = 1
		colorList = []int16{0}
	}

	// Use a power-of-2 square texture (Sauer requires power-of-2)
	// Lay colors out in a grid. Find smallest power-of-2 side that fits.
	side := 1
	for side*side < n {
		side *= 2
	}
	if side < 4 {
		side = 4 // minimum 4x4
	}

	img := image.NewRGBA(image.Rect(0, 0, side, side))
	for i, c := range colorList {
		x := i % side
		y := i / side
		img.Set(x, y, HSL16ToRGB(c))
	}

	return img
}
