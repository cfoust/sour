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
// Matches JagexColor.HSLtoRGB from the RuneLite model exporter.
// HSL16 format: (hue & 63) << 10 | (sat & 7) << 7 | (lum & 127)
func HSL16ToRGB(hsl16 int16) color.RGBA {
	const (
		hueOffset = 0.5 / 64.0
		satOffset = 0.5 / 8.0
		brightness = 0.9 // BRIGHTNESS_MIN
	)

	hue := float64((hsl16>>10)&0x3F)/64.0 + hueOffset
	sat := float64((hsl16>>7)&0x07)/8.0 + satOffset
	lum := float64(hsl16&0x7F) / 128.0

	// Standard HSL→RGB using chroma method
	chroma := (1.0 - math.Abs(2.0*lum-1.0)) * sat
	x := chroma * (1.0 - math.Abs(math.Mod(hue*6.0, 2.0)-1.0))
	lightness := lum - chroma/2.0

	r, g, b := lightness, lightness, lightness
	switch int(hue * 6.0) {
	case 0:
		r += chroma
		g += x
	case 1:
		g += chroma
		r += x
	case 2:
		g += chroma
		b += x
	case 3:
		b += chroma
		g += x
	case 4:
		b += chroma
		r += x
	default:
		r += chroma
		b += x
	}

	// Apply brightness adjustment (gamma correction)
	r = math.Pow(r, brightness)
	g = math.Pow(g, brightness)
	b = math.Pow(b, brightness)

	ri := int(r * 256.0)
	gi := int(g * 256.0)
	bi := int(b * 256.0)
	if ri > 255 { ri = 255 }
	if gi > 255 { gi = 255 }
	if bi > 255 { bi = 255 }

	return color.RGBA{R: uint8(ri), G: uint8(gi), B: uint8(bi), A: 255}
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

	// Negate Y only (OSRS Y is inverted: negative = up)
	for i := 0; i < len(m.VertexX); i++ {
		x := float64(m.VertexX[i]) * scale
		y := float64(-m.VertexY[i]) * scale
		z := float64(m.VertexZ[i]) * scale
		obj += fmt.Sprintf("v %f %f %f\n", x, y, z)
	}
	obj += "\n"

	// Compute texture UVs using the exact OSRS algorithm
	uvs := m.computeTextureUVs()

	// Write UV coordinates
	hasComposite := m.DominantTexture() >= 0
	for i := 0; i < len(m.FaceA); i++ {
		idx := i * 6
		if m.TextureIDs[i] >= 0 && hasComposite {
			// Textured face: UVs map to left half of composite (0.0 - 0.5 in U)
			obj += fmt.Sprintf("vt %f %f\nvt %f %f\nvt %f %f\n",
				float64(uvs[idx])*0.5, float64(uvs[idx+1]),
				float64(uvs[idx+2])*0.5, float64(uvs[idx+3]),
				float64(uvs[idx+4])*0.5, float64(uvs[idx+5]))
		} else if m.TextureIDs[i] >= 0 {
			// Textured face, no composite: full texture
			obj += fmt.Sprintf("vt %f %f\nvt %f %f\nvt %f %f\n",
				float64(uvs[idx]), float64(uvs[idx+1]),
				float64(uvs[idx+2]), float64(uvs[idx+3]),
				float64(uvs[idx+4]), float64(uvs[idx+5]))
		} else if hasComposite {
			// Colored face in composite: right half
			obj += "vt 0.75 0.5\nvt 0.75 0.5\nvt 0.75 0.5\n"
		} else {
			// Color-only model: center
			obj += "vt 0.5 0.5\nvt 0.5 0.5\nvt 0.5 0.5\n"
		}
	}
	obj += "\n"

	// Faces: CBA order in OBJ file.
	// Sauer's OBJ loader stores face "f A B C" as triangle (C,B,A) — reversing winding.
	// So we output CBA to get ABC in the engine.
	for i := 0; i < len(m.FaceA); i++ {
		uvBase := i*3 + 1 // 1-based
		obj += fmt.Sprintf("f %d/%d %d/%d %d/%d\n",
			m.FaceC[i]+1, uvBase+2,
			m.FaceB[i]+1, uvBase+1,
			m.FaceA[i]+1, uvBase)
	}

	return obj, ""
}

// computeTextureUVs computes UV coordinates for all faces at once.
// Ports the exact algorithm from the Kotlin exporter's ModelDefinition.computeTextureUVCoordinates().
func (m *Model) computeTextureUVs() []float32 {
	uv := make([]float32, 6*len(m.FaceA))
	for face := 0; face < len(m.FaceA); face++ {
		if m.TextureIDs[face] < 0 {
			continue
		}

		idx := face * 6
		texCoord := int8(-1)
		if m.TexCoords != nil {
			texCoord = m.TexCoords[face]
		}

		if texCoord == -1 {
			// Default UVs when no texture coordinate mapping
			uv[idx] = 0.0
			uv[idx+1] = 1.0
			uv[idx+2] = 1.0
			uv[idx+3] = 1.0
			uv[idx+4] = 0.0
			uv[idx+5] = 0.0
			continue
		}

		tc := int(texCoord) & 0xFF
		if m.TexTriA == nil || tc >= len(m.TexTriA) {
			continue
		}

		// Texture triangle vertices
		t1 := int(m.TexTriA[tc])
		t2 := int(m.TexTriB[tc])
		t3 := int(m.TexTriC[tc])
		if t1 < 0 || t1 >= len(m.VertexX) || t2 < 0 || t2 >= len(m.VertexX) || t3 < 0 || t3 >= len(m.VertexX) {
			continue
		}

		// Triangle origin (vertex 1 of texture triangle)
		triX := float32(m.VertexX[t1])
		triY := float32(m.VertexY[t1])
		triZ := float32(m.VertexZ[t1])

		// Edge vectors from vertex 1 to 2 and 1 to 3
		e1x := float32(m.VertexX[t2]) - triX
		e1y := float32(m.VertexY[t2]) - triY
		e1z := float32(m.VertexZ[t2]) - triZ
		e2x := float32(m.VertexX[t3]) - triX
		e2y := float32(m.VertexY[t3]) - triY
		e2z := float32(m.VertexZ[t3]) - triZ

		// Face vertex positions relative to triangle origin
		f1x := float32(m.VertexX[m.FaceA[face]]) - triX
		f1y := float32(m.VertexY[m.FaceA[face]]) - triY
		f1z := float32(m.VertexZ[m.FaceA[face]]) - triZ
		f2x := float32(m.VertexX[m.FaceB[face]]) - triX
		f2y := float32(m.VertexY[m.FaceB[face]]) - triY
		f2z := float32(m.VertexZ[m.FaceB[face]]) - triZ
		f3x := float32(m.VertexX[m.FaceC[face]]) - triX
		f3y := float32(m.VertexY[m.FaceC[face]]) - triY
		f3z := float32(m.VertexZ[m.FaceC[face]]) - triZ

		// Cross product: normal = e1 × e2
		nx := e1y*e2z - e1z*e2y
		ny := e1z*e2x - e1x*e2z
		nz := e1x*e2y - e1y*e2x

		// Compute U basis vector: e2 × normal
		ux := e2y*nz - e2z*ny
		uy := e2z*nx - e2x*nz
		uz := e2x*ny - e2y*nx
		denom := ux*e1x + uy*e1y + uz*e1z
		if denom == 0 {
			continue
		}
		invDenom := 1.0 / float32(denom)

		uv[idx] = (ux*f1x + uy*f1y + uz*f1z) * invDenom
		uv[idx+2] = (ux*f2x + uy*f2y + uz*f2z) * invDenom
		uv[idx+4] = (ux*f3x + uy*f3y + uz*f3z) * invDenom

		// Compute V basis vector: e1 × normal (note different order)
		vx := e1y*nz - e1z*ny
		vy := e1z*nx - e1x*nz
		vz := e1x*ny - e1y*nx
		denom2 := vx*e2x + vy*e2y + vz*e2z
		if denom2 == 0 {
			continue
		}
		invDenom2 := 1.0 / float32(denom2)

		uv[idx+1] = (vx*f1x + vy*f1y + vz*f1z) * invDenom2
		uv[idx+3] = (vx*f2x + vy*f2y + vz*f2z) * invDenom2
		uv[idx+5] = (vx*f3x + vy*f3y + vz*f3z) * invDenom2
	}
	return uv
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
