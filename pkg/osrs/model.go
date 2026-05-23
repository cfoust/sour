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

	// Texture coordinate triangles: 3 vertex indices defining UV space
	TexTriA []int16
	TexTriB []int16
	TexTriC []int16
	// Per-face index into TexTri arrays (-1 = use face's own vertices)
	TexCoords []int8
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
		TexCoords:  make([]int8, triangleCount),
	}
	if texTriCount > 0 {
		m.TexTriA = make([]int16, texTriCount)
		m.TexTriB = make([]int16, texTriCount)
		m.TexTriC = make([]int16, texTriCount)
	}

	// Initialize
	for i := range m.TextureIDs {
		m.TextureIDs[i] = -1
		m.TexCoords[i] = -1
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
			if flag&0x1 == 1 {
				// flat shading flag
			}
			if flag&0x2 != 0 {
				m.TexCoords[i] = int8(flag >> 2)
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

	// Read texture triangles
	if texTriCount > 0 {
		texBuf := NewBuffer(data)
		texBuf.SetPosition(colorOff + triangleCount*2) // textureOffset
		for i := 0; i < texTriCount; i++ {
			ta, _ := texBuf.ReadUShort()
			tb, _ := texBuf.ReadUShort()
			tc, _ := texBuf.ReadUShort()
			m.TexTriA[i] = int16(ta)
			m.TexTriB[i] = int16(tb)
			m.TexTriC[i] = int16(tc)
		}
	}

	// Post-process: if textureCoordinates match face vertices, clear them
	// (matching Java logic at line 656-671)
	for i := 0; i < triangleCount; i++ {
		if m.TexCoords[i] != -1 && m.TexTriA != nil {
			coord := int(m.TexCoords[i]) & 0xff
			if coord < len(m.TexTriA) {
				if int(m.TexTriA[coord]) == m.FaceA[i] &&
					int(m.TexTriB[coord]) == m.FaceB[i] &&
					int(m.TexTriC[coord]) == m.FaceC[i] {
					m.TexCoords[i] = -1
				}
			}
		}
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
		VertexX:    make([]int, vertexCount),
		VertexY:    make([]int, vertexCount),
		VertexZ:    make([]int, vertexCount),
		FaceA:      make([]int, triangleCount),
		FaceB:      make([]int, triangleCount),
		FaceC:      make([]int, triangleCount),
		Colors:     make([]int16, triangleCount),
		TextureIDs: make([]int16, triangleCount),
		TexCoords:  make([]int8, triangleCount),
	}
	for i := range m.TextureIDs {
		m.TextureIDs[i] = -1
		m.TexCoords[i] = -1
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

// DominantTexture returns the most common texture ID across faces, or -1 if no faces are textured.
func (m *Model) DominantTexture() int {
	counts := make(map[int16]int)
	for _, t := range m.TextureIDs {
		if t >= 0 {
			counts[t]++
		}
	}
	if len(counts) == 0 {
		return -1
	}
	bestID := int16(-1)
	bestCount := 0
	for id, c := range counts {
		if c > bestCount {
			bestID = id
			bestCount = c
		}
	}
	return int(bestID)
}

// Invert mirrors the model by negating Z and swapping face winding.
// Matches OSRS Model.invert().
func (m *Model) Invert() {
	for i := range m.VertexZ {
		m.VertexZ[i] = -m.VertexZ[i]
	}
	for i := range m.FaceA {
		m.FaceA[i], m.FaceC[i] = m.FaceC[i], m.FaceA[i]
	}
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

// ToOBJ exports the model as a Wavefront OBJ with per-vertex UV coordinates.
// Returns the OBJ content string (second return is unused).
func (m *Model) ToOBJ(name string, scale float64) (string, string) {
	obj := fmt.Sprintf("# OSRS model %s\n\n", name)

	// Vertices: OSRS Y-up (neg=up), standard OBJ Y-up
	for i := 0; i < len(m.VertexX); i++ {
		x := float64(m.VertexX[i]) * scale
		y := float64(-m.VertexY[i]) * scale
		z := float64(m.VertexZ[i]) * scale
		obj += fmt.Sprintf("v %f %f %f\n", x, y, z)
	}
	obj += "\n"

	// UV coordinates: 3 per face (one per vertex of each triangle)
	// For textured faces: compute UVs from texture triangle projection
	// For colored faces: map to color pixel in composite texture
	for i := 0; i < len(m.FaceA); i++ {
		va, vb, vc := m.FaceA[i], m.FaceB[i], m.FaceC[i]

		hasTexture := m.TextureIDs[i] >= 0 && m.DominantTexture() >= 0

		if hasTexture {
			// Textured face: compute UVs from texture triangle
			// UVs map to left half of composite (0.0 - 0.5 in U)
			ta, tb, tc := va, vb, vc
			if m.TexCoords != nil && m.TexCoords[i] != -1 && m.TexTriA != nil {
				coord := int(m.TexCoords[i]) & 0xff
				if coord < len(m.TexTriA) {
					ta = int(m.TexTriA[coord])
					tb = int(m.TexTriB[coord])
					tc = int(m.TexTriC[coord])
				}
			}
			uA, vA := m.computeTexUV(va, ta, tb, tc)
			uB, vB := m.computeTexUV(vb, ta, tb, tc)
			uC, vC := m.computeTexUV(vc, ta, tb, tc)
			// Scale to left half (0-0.5 in U)
			obj += fmt.Sprintf("vt %f %f\nvt %f %f\nvt %f %f\n",
				uA*0.5, vA, uB*0.5, vB, uC*0.5, vC)
		} else {
			// Colored face: UV points to right half of composite (0.5-1.0 in U)
			// or full texture for color-only models
			if m.DominantTexture() >= 0 {
				// Composite: right half
				obj += "vt 0.75 0.5\nvt 0.75 0.5\nvt 0.75 0.5\n"
			} else {
				// Color-only: center of texture
				obj += "vt 0.5 0.5\nvt 0.5 0.5\nvt 0.5 0.5\n"
			}
		}
	}
	obj += "\n"

	// Faces: CBA winding (Sauer OBJ remap flips handedness)
	// Each face has 3 dedicated UV indices (3*i+1, 3*i+2, 3*i+3)
	for i := 0; i < len(m.FaceA); i++ {
		uvBase := i*3 + 1 // 1-based
		obj += fmt.Sprintf("f %d/%d %d/%d %d/%d\n",
			m.FaceC[i]+1, uvBase+2,
			m.FaceB[i]+1, uvBase+1,
			m.FaceA[i]+1, uvBase)
	}

	return obj, ""
}

// computeTexUV projects vertex v onto the texture triangle (ta, tb, tc)
// and returns (u, v) texture coordinates.
// Uses full 3D barycentric coordinates to handle arbitrary triangle orientations.
func (m *Model) computeTexUV(v, ta, tb, tc int) (float64, float64) {
	if ta < 0 || ta >= len(m.VertexX) || tb < 0 || tb >= len(m.VertexX) ||
		tc < 0 || tc >= len(m.VertexX) || v < 0 || v >= len(m.VertexX) {
		return 0, 0
	}

	// Texture triangle in 3D space
	ax, ay, az := float64(m.VertexX[ta]), float64(m.VertexY[ta]), float64(m.VertexZ[ta])
	bx, by, bz := float64(m.VertexX[tb]), float64(m.VertexY[tb]), float64(m.VertexZ[tb])
	cx, cy, cz := float64(m.VertexX[tc]), float64(m.VertexY[tc]), float64(m.VertexZ[tc])

	// Point to project
	px, py, pz := float64(m.VertexX[v]), float64(m.VertexY[v]), float64(m.VertexZ[v])

	// Edge vectors
	e1x, e1y, e1z := bx-ax, by-ay, bz-az
	e2x, e2y, e2z := cx-ax, cy-ay, cz-az
	epx, epy, epz := px-ax, py-ay, pz-az

	// Dot products for barycentric
	d11 := e1x*e1x + e1y*e1y + e1z*e1z
	d12 := e1x*e2x + e1y*e2y + e1z*e2z
	d22 := e2x*e2x + e2y*e2y + e2z*e2z
	dp1 := epx*e1x + epy*e1y + epz*e1z
	dp2 := epx*e2x + epy*e2y + epz*e2z

	denom := d11*d22 - d12*d12
	if denom == 0 {
		return 0, 0
	}

	// Barycentric coordinates (u for edge AB, w for edge AC)
	u := (d22*dp1 - d12*dp2) / denom
	w := (d11*dp2 - d12*dp1) / denom

	return u, w
}

// CompositeSkin generates a composite skin texture for the model.
// Layout: left half = OSRS texture (if any), right half = color palette.
// If textures is nil or the model has no textured faces, uses color-only.
func (m *Model) CompositeSkin(textures []*image.RGBA) *image.RGBA {
	// Collect unique colors for non-textured faces
	colorSet := make(map[int16]int)
	var colorList []int16
	for i, c := range m.Colors {
		if m.TextureIDs[i] >= 0 {
			continue // textured face, skip
		}
		if _, ok := colorSet[c]; !ok {
			colorSet[c] = len(colorList)
			colorList = append(colorList, c)
		}
	}
	if len(colorList) == 0 {
		colorList = []int16{0}
		colorSet[0] = 0
	}

	// Determine if we have an OSRS texture
	texID := m.DominantTexture()
	hasOSRSTex := texID >= 0 && textures != nil && texID < len(textures) && textures[texID] != nil

	if !hasOSRSTex {
		// Color-only: 128x128 texture filled with color patches
		img := image.NewRGBA(image.Rect(0, 0, 128, 128))
		patchSize := 128 / len(colorList)
		if patchSize < 1 {
			patchSize = 1
		}
		for i, c := range colorList {
			col := HSL16ToRGB(c)
			startX := (i * patchSize) % 128
			for y := 0; y < 128; y++ {
				for x := startX; x < startX+patchSize && x < 128; x++ {
					img.Set(x, y, col)
				}
			}
		}
		return img
	}

	// Composite: 256x128. Left 128x128 = OSRS texture, right 128x128 = color palette
	osrsTex := textures[texID]
	img := image.NewRGBA(image.Rect(0, 0, 256, 128))

	// Copy OSRS texture to left half
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, osrsTex.At(x, y))
		}
	}

	// Fill right half with color patches
	patchSize := 128 / len(colorList)
	if patchSize < 1 {
		patchSize = 1
	}
	for i, c := range colorList {
		col := HSL16ToRGB(c)
		startX := 128 + (i*patchSize)%128
		for y := 0; y < 128; y++ {
			for x := startX; x < startX+patchSize && x < 256; x++ {
				img.Set(x, y, col)
			}
		}
	}

	return img
}
