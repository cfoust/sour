package preview

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

type glInstance struct {
	offset [3]float32
	scale  [3]float32
	color  [3]float32
	ao     float32
}

// GLRenderer holds an OpenGL context for batch rendering.
type GLRenderer struct {
	window  *glfw.Window
	program uint32
	vao     uint32
	cubeVBO uint32
	cubeEBO uint32
	fbo     uint32
	rbo     uint32
	depthRBO uint32
	width   int
	height  int
}

// Vertex and fragment shaders matching Three.js MeshLambertMaterial lighting model.
// Three.js Lambert: outgoing = directDiffuse + indirectDiffuse
//   directDiffuse  = sum of saturate(dot(N, L)) * lightColor * (diffuse / PI)
//   indirectDiffuse = hemisphereIrradiance * (diffuse / PI)
//   hemisphereIrradiance = mix(groundColor, skyColor, 0.5 * dot(N, hemiDir) + 0.5)

var vertexShaderSrc = `
#version 410 core

layout(location = 0) in vec3 aPos;
layout(location = 1) in vec3 aNormal;
layout(location = 2) in vec3 iOffset;
layout(location = 3) in vec3 iScale;
layout(location = 4) in vec3 iColor;
layout(location = 5) in float iAO;

uniform mat4 uView;
uniform mat4 uProj;

out vec3 vColor;
out vec3 vNormalView;
out vec3 vViewPosition;
out float vAO;

void main() {
    vec3 worldPos = (aPos - 0.5) * iScale + iOffset;
    vec4 mvPosition = uView * vec4(worldPos, 1.0);
    gl_Position = uProj * mvPosition;

    // Transform normal to view space (Three.js does this via normalMatrix)
    // For uniform scale per axis, we can use the view matrix upper 3x3
    vec3 worldNormal = aNormal;
    vNormalView = normalize(mat3(uView) * worldNormal);
    vViewPosition = -mvPosition.xyz;
    vColor = iColor;
    vAO = iAO;
}
` + "\x00"

var fragmentShaderSrc = `
#version 410 core

#define RECIPROCAL_PI 0.3183098861837907
#define saturate(a) clamp(a, 0.0, 1.0)

in vec3 vColor;
in vec3 vNormalView;
in vec3 vViewPosition;
in float vAO;

// Directional lights (direction in view space)
uniform vec3 uDirLight0Dir;
uniform vec3 uDirLight0Color;
uniform vec3 uDirLight1Dir;
uniform vec3 uDirLight1Color;

// Hemisphere light (direction in view space)
uniform vec3 uHemiDir;
uniform vec3 uHemiSkyColor;
uniform vec3 uHemiGroundColor;

uniform float uAlpha;

out vec4 fragColor;

vec3 BRDF_Lambert(vec3 diffuseColor) {
    return RECIPROCAL_PI * diffuseColor;
}

void main() {
    vec3 normal = normalize(vNormalView);
    vec3 diffuseColor = vColor;

    // === Exact Three.js r184 Lambert model ===
    // Directional light 0 (sun)
    float dotNL0 = saturate(dot(normal, uDirLight0Dir));
    vec3 directDiffuse = dotNL0 * uDirLight0Color * BRDF_Lambert(diffuseColor);

    // Directional light 1 (fill)
    float dotNL1 = saturate(dot(normal, uDirLight1Dir));
    directDiffuse += dotNL1 * uDirLight1Color * BRDF_Lambert(diffuseColor);

    // Hemisphere light
    float hemiWeight = 0.5 * dot(normal, uHemiDir) + 0.5;
    vec3 hemiIrradiance = mix(uHemiGroundColor, uHemiSkyColor, hemiWeight);
    vec3 indirectDiffuse = hemiIrradiance * BRDF_Lambert(diffuseColor);

    vec3 outgoingLight = directDiffuse + indirectDiffuse;

    // AO
    float ao = 0.2 + 0.8 * vAO;
    outgoingLight *= ao;

    // Exposure + ACES filmic tonemap (Narkowicz 2015 fit)
    vec3 x = outgoingLight * 1.3;
    vec3 mapped = clamp(
        (x * (2.51 * x + 0.03)) / (x * (2.43 * x + 0.59) + 0.14),
        0.0, 1.0
    );
    // Linear to sRGB gamma
    fragColor = vec4(pow(mapped, vec3(1.0/2.2)), uAlpha);
}
` + "\x00"

// NewGLRenderer creates an OpenGL context and compiles shaders.
// Returns nil if GPU init fails.
func NewGLRenderer(width, height int) *GLRenderer {
	runtime.LockOSThread()

	if err := glfw.Init(); err != nil {
		return nil
	}

	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, "offscreen", nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil
	}
	window.MakeContextCurrent()

	if err := gl.Init(); err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil
	}

	program, err := compileProgram(vertexShaderSrc, fragmentShaderSrc)
	if err != nil {
		window.Destroy()
		glfw.Terminate()
		return nil
	}

	r := &GLRenderer{
		window:  window,
		program: program,
		width:   width,
		height:  height,
	}

	r.initCubeGeometry()
	r.initFramebuffer()

	return r
}

func (r *GLRenderer) Close() {
	gl.DeleteFramebuffers(1, &r.fbo)
	gl.DeleteRenderbuffers(1, &r.rbo)
	gl.DeleteRenderbuffers(1, &r.depthRBO)
	gl.DeleteBuffers(1, &r.cubeVBO)
	gl.DeleteBuffers(1, &r.cubeEBO)
	gl.DeleteVertexArrays(1, &r.vao)
	gl.DeleteProgram(r.program)
	r.window.Destroy()
	glfw.Terminate()
}

func (r *GLRenderer) initCubeGeometry() {
	// Unit cube vertices with normals
	vertices := []float32{
		// pos (3)          normal (3)
		// Front face (+Z in local)
		0, 0, 1, 0, 0, 1,
		1, 0, 1, 0, 0, 1,
		1, 1, 1, 0, 0, 1,
		0, 1, 1, 0, 0, 1,
		// Back face (-Z)
		0, 0, 0, 0, 0, -1,
		0, 1, 0, 0, 0, -1,
		1, 1, 0, 0, 0, -1,
		1, 0, 0, 0, 0, -1,
		// Top face (+Y)
		0, 1, 0, 0, 1, 0,
		0, 1, 1, 0, 1, 0,
		1, 1, 1, 0, 1, 0,
		1, 1, 0, 0, 1, 0,
		// Bottom face (-Y)
		0, 0, 0, 0, -1, 0,
		1, 0, 0, 0, -1, 0,
		1, 0, 1, 0, -1, 0,
		0, 0, 1, 0, -1, 0,
		// Right face (+X)
		1, 0, 0, 1, 0, 0,
		1, 1, 0, 1, 0, 0,
		1, 1, 1, 1, 0, 0,
		1, 0, 1, 1, 0, 0,
		// Left face (-X)
		0, 0, 0, -1, 0, 0,
		0, 0, 1, -1, 0, 0,
		0, 1, 1, -1, 0, 0,
		0, 1, 0, -1, 0, 0,
	}

	indices := []uint32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}

	gl.GenVertexArrays(1, &r.vao)
	gl.BindVertexArray(r.vao)

	gl.GenBuffers(1, &r.cubeVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.cubeVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.STATIC_DRAW)

	// Position
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 3, gl.FLOAT, false, 24, 0)
	// Normal
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 3, gl.FLOAT, false, 24, 12)

	gl.GenBuffers(1, &r.cubeEBO)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, r.cubeEBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(indices)*4, gl.Ptr(indices), gl.STATIC_DRAW)

	gl.BindVertexArray(0)
}

func (r *GLRenderer) initFramebuffer() {
	gl.GenFramebuffers(1, &r.fbo)
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.fbo)

	gl.GenRenderbuffers(1, &r.rbo)
	gl.BindRenderbuffer(gl.RENDERBUFFER, r.rbo)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.RGBA8, int32(r.width), int32(r.height))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, r.rbo)

	gl.GenRenderbuffers(1, &r.depthRBO)
	gl.BindRenderbuffer(gl.RENDERBUFFER, r.depthRBO)
	gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT24, int32(r.width), int32(r.height))
	gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, r.depthRBO)

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

// RenderGL renders a MapPreview using GPU-accelerated instanced cubes.
func (r *GLRenderer) RenderGL(p *MapPreview) *image.RGBA {
	w, h := r.width, r.height
	gridSize := int(p.GridSize)
	clipZ := int(p.ClipY)
	if clipZ <= 0 || clipZ >= gridSize {
		clipZ = gridSize
	}

	// Camera setup (same math as software renderer)
	focusX := float64(p.FocusX)
	focusY := float64(p.FocusY)
	focusZ := float64(p.FocusZ)
	radius := float64(p.FocusRadius)

	yawRad := float64(p.CameraYaw) / 10.0 * math.Pi / 180.0
	pitchRad := float64(p.CameraPitch) / 10.0 * math.Pi / 180.0

	camX := float32(focusX + math.Cos(pitchRad)*math.Cos(yawRad)*radius)
	camY := float32(focusY + math.Cos(pitchRad)*math.Sin(yawRad)*radius)
	camZ := float32(focusZ + math.Sin(pitchRad)*radius)

	// Three.js coordinate mapping: Sauer (X,Y,Z) → GL (X,Z,Y)
	// Camera in GL coords
	camGL := mgl32.Vec3{camX, camZ, camY}
	focusGL := mgl32.Vec3{float32(focusX), float32(focusZ), float32(focusY)}

	view := mgl32.LookAtV(camGL, focusGL, mgl32.Vec3{0, 1, 0})
	proj := mgl32.Perspective(mgl32.DegToRad(45), float32(w)/float32(h), 0.1, float32(gridSize)*10)

	// Build glInstance data

	jitterHash := func(x, y, z int) float32 {
		h := (x*374761393 + y*668265263 + z*1274126177)
		h = ((h ^ (h >> 13)) * 1031)
		return float32((h^(h>>16))&0xffff) / 65535.0
	}

	var solidInstances []glInstance
	var waterInstances []glInstance
	var lavaInstances []glInstance

	for _, v := range p.Voxels {
		if int(v.Z) >= clipZ {
			continue
		}
		size := float32(v.Size())
		halfSize := (size - 1) * 0.5

		off := [3]float32{
			float32(v.X) + halfSize,
			float32(v.Z) + halfSize,
			float32(v.Y) + halfSize,
		}

		mat := v.Flags & 0x1c
		if mat == FlagWater {
			waterInstances = append(waterInstances, glInstance{
				offset: off, scale: [3]float32{size, size, size},
				color: [3]float32{0.13, 0.4, 0.8}, ao: 1,
			})
			continue
		}
		if mat == FlagLava {
			lavaInstances = append(lavaInstances, glInstance{
				offset: off, scale: [3]float32{size, size, size},
				color: [3]float32{1.0, 0.27, 0.0}, ao: 1,
			})
			continue
		}

		palIdx := int(v.PaletteIndex)
		if palIdx >= len(p.Palette) {
			palIdx = 0
		}
		cr := float32(p.Palette[palIdx][0]) / 255
		cg := float32(p.Palette[palIdx][1]) / 255
		cb := float32(p.Palette[palIdx][2]) / 255

		// Saturation boost — ACES in shader handles overshoot
		lum := float32(0.299)*cr + float32(0.587)*cg + float32(0.114)*cb
		cr = clampF32(lum+(cr-lum)*1.8, 0, 1)
		cg = clampF32(lum+(cg-lum)*1.8, 0, 1)
		cb = clampF32(lum+(cb-lum)*1.8, 0, 1)

		jitter := 0.85 + 0.3*jitterHash(int(v.X), int(v.Y), int(v.Z))
		cr *= jitter
		cg *= jitter
		cb *= jitter

		ao := 0.2 + 0.8*float32(v.AO)/255

		solidInstances = append(solidInstances, glInstance{
			offset: off, scale: [3]float32{size, size, size},
			color: [3]float32{cr, cg, cb}, ao: ao,
		})
	}



	// Render
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.fbo)
	gl.Viewport(0, 0, int32(w), int32(h))

	// Sky color
	skyR := clampF32(float32(p.SkyTop[0])/255*1.5+0.05, 0, 1)
	skyG := clampF32(float32(p.SkyTop[1])/255*1.5+0.05, 0, 1)
	skyB := clampF32(float32(p.SkyTop[2])/255*1.5+0.08, 0, 1)
	gl.ClearColor(skyR, skyG, skyB, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)

	gl.UseProgram(r.program)

	// Set view and projection matrices
	viewLoc := gl.GetUniformLocation(r.program, gl.Str("uView\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])
	projLoc := gl.GetUniformLocation(r.program, gl.Str("uProj\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &proj[0])

	// Three.js transforms light directions into view space.
	// viewMatrix upper 3x3 rotates world → view.
	viewRot := view.Mat3()
	transformDir := func(dir mgl32.Vec3) mgl32.Vec3 {
		// mat3(viewMatrix) * worldDir, then normalize
		out := mgl32.Vec3{
			viewRot[0]*dir[0] + viewRot[3]*dir[1] + viewRot[6]*dir[2],
			viewRot[1]*dir[0] + viewRot[4]*dir[1] + viewRot[7]*dir[2],
			viewRot[2]*dir[0] + viewRot[5]*dir[1] + viewRot[8]*dir[2],
		}
		return out.Normalize()
	}

	// Camera-relative lighting: key light over viewer's right shoulder,
	// fill from the opposite side. Hemisphere stays world-aligned.
	setVec3(r.program, "uHemiDir", transformDir(mgl32.Vec3{0, 1, 0}))
	setVec3(r.program, "uHemiSkyColor", mgl32.Vec3{0.27, 0.30, 0.40})
	setVec3(r.program, "uHemiGroundColor", mgl32.Vec3{0.17, 0.14, 0.10})

	// Key light: 30° right of camera, 40° elevation (in Sauer coords → GL)
	keyYaw := yawRad - math.Pi/6
	keyElev := 40.0 * math.Pi / 180.0
	sunWorldDir := mgl32.Vec3{
		float32(math.Cos(keyElev) * math.Cos(keyYaw)),
		float32(math.Sin(keyElev)),
		float32(math.Cos(keyElev) * math.Sin(keyYaw)),
	}.Normalize()
	setVec3(r.program, "uDirLight0Dir", transformDir(sunWorldDir))
	setVec3(r.program, "uDirLight0Color", mgl32.Vec3{1.2, 1.12, 1.04})

	// Fill light: opposite side, low angle
	fillYaw := yawRad + math.Pi*2/3
	fillElev := 10.0 * math.Pi / 180.0
	fillWorldDir := mgl32.Vec3{
		float32(math.Cos(fillElev) * math.Cos(fillYaw)),
		float32(math.Sin(fillElev)),
		float32(math.Cos(fillElev) * math.Sin(fillYaw)),
	}.Normalize()
	setVec3(r.program, "uDirLight1Dir", transformDir(fillWorldDir))
	setVec3(r.program, "uDirLight1Color", mgl32.Vec3{0.16, 0.21, 0.27})

	// Backface culling like Three.js
	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)

	// Draw solid instances (alpha = 1.0)
	alphaLoc := gl.GetUniformLocation(r.program, gl.Str("uAlpha\x00"))
	gl.Uniform1f(alphaLoc, 1.0)
	r.drawInstances(solidInstances)

	// Draw water/lava with transparency
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.DepthMask(false)

	gl.Uniform1f(alphaLoc, 0.5)
	r.drawInstances(waterInstances)

	gl.Uniform1f(alphaLoc, 0.9)
	r.drawInstances(lavaInstances)

	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)

	// Read depth buffer for edge emphasis
	depthBuf := make([]float32, w*h)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(depthBuf))

	// Read back pixels
	pixels := make([]uint8, w*h*4)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))

	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	// Post-processing: edge emphasis + vignette (CPU-side on readback)
	applyGLPostProcess(pixels, depthBuf, w, h, float64(gridSize))

	// Convert to image.RGBA (OpenGL is bottom-up, flip Y)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		srcRow := (h - 1 - y) * w * 4
		for x := 0; x < w; x++ {
			i := srcRow + x*4
			img.SetRGBA(x, y, color.RGBA{pixels[i], pixels[i+1], pixels[i+2], 255})
		}
	}

	return img
}

// applyGLPostProcess applies edge emphasis and vignette to GL-rendered pixels.
// Linearizes the GL depth buffer so edge detection uses world-space distances
// with the same scale factor as the software renderer.
func applyGLPostProcess(pixels []uint8, depthBuf []float32, w, h int, gridSize float64) {
	near := 0.1
	far := gridSize * 10

	// Linearize GL depth buffer (hyperbolic → world units)
	linearDepth := make([]float64, w*h)
	for i, d := range depthBuf {
		zNDC := 2*float64(d) - 1
		linearDepth[i] = 2 * near * far / (far + near - zNDC*(far-near))
	}

	// Sobel edge detection on linearized depth
	for py := 1; py < h-1; py++ {
		for px := 1; px < w-1; px++ {
			gx := -linearDepth[(py-1)*w+(px-1)] + linearDepth[(py-1)*w+(px+1)] +
				-2*linearDepth[py*w+(px-1)] + 2*linearDepth[py*w+(px+1)] +
				-linearDepth[(py+1)*w+(px-1)] + linearDepth[(py+1)*w+(px+1)]
			gy := -linearDepth[(py-1)*w+(px-1)] - 2*linearDepth[(py-1)*w+px] - linearDepth[(py-1)*w+(px+1)] +
				linearDepth[(py+1)*w+(px-1)] + 2*linearDepth[(py+1)*w+px] + linearDepth[(py+1)*w+(px+1)]

			edge := math.Sqrt(gx*gx + gy*gy)
			darken := 1.0 - math.Min(0.5, edge*0.02)

			i := (py*w + px) * 4
			pixels[i] = uint8(float64(pixels[i]) * darken)
			pixels[i+1] = uint8(float64(pixels[i+1]) * darken)
			pixels[i+2] = uint8(float64(pixels[i+2]) * darken)
		}
	}

	// Vignette (same as software renderer)
	cx, cy := float64(w)/2, float64(h)/2
	maxDist := math.Sqrt(cx*cx + cy*cy)
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			dx := float64(px) - cx
			dy := float64(py) - cy
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist
			v := 1.0 - math.Max(0, (dist-0.5))*1.2
			if v < 0.3 {
				v = 0.3
			}
			i := (py*w + px) * 4
			pixels[i] = uint8(float64(pixels[i]) * v)
			pixels[i+1] = uint8(float64(pixels[i+1]) * v)
			pixels[i+2] = uint8(float64(pixels[i+2]) * v)
		}
	}
}

func (r *GLRenderer) drawInstances(instances []glInstance) {
	if len(instances) == 0 {
		return
	}

	// Pack into flat arrays
	offsets := make([]float32, len(instances)*3)
	scales := make([]float32, len(instances)*3)
	colors := make([]float32, len(instances)*3)
	aos := make([]float32, len(instances))

	for i, inst := range instances {
		offsets[i*3] = inst.offset[0]
		offsets[i*3+1] = inst.offset[1]
		offsets[i*3+2] = inst.offset[2]
		scales[i*3] = inst.scale[0]
		scales[i*3+1] = inst.scale[1]
		scales[i*3+2] = inst.scale[2]
		colors[i*3] = inst.color[0]
		colors[i*3+1] = inst.color[1]
		colors[i*3+2] = inst.color[2]
		aos[i] = inst.ao
	}

	gl.BindVertexArray(r.vao)

	var offsetVBO, scaleVBO, colorVBO, aoVBO uint32

	// Offset
	gl.GenBuffers(1, &offsetVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, offsetVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(offsets)*4, gl.Ptr(offsets), gl.STREAM_DRAW)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 3, gl.FLOAT, false, 0, 0)
	gl.VertexAttribDivisor(2, 1)

	// Scale (vec3)
	gl.GenBuffers(1, &scaleVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, scaleVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(scales)*4, gl.Ptr(scales), gl.STREAM_DRAW)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 3, gl.FLOAT, false, 0, 0)
	gl.VertexAttribDivisor(3, 1)

	// Color
	gl.GenBuffers(1, &colorVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, colorVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(colors)*4, gl.Ptr(colors), gl.STREAM_DRAW)
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointerWithOffset(4, 3, gl.FLOAT, false, 0, 0)
	gl.VertexAttribDivisor(4, 1)

	// AO
	gl.GenBuffers(1, &aoVBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, aoVBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(aos)*4, gl.Ptr(aos), gl.STREAM_DRAW)
	gl.EnableVertexAttribArray(5)
	gl.VertexAttribPointerWithOffset(5, 1, gl.FLOAT, false, 0, 0)
	gl.VertexAttribDivisor(5, 1)

	gl.DrawElementsInstanced(gl.TRIANGLES, 36, gl.UNSIGNED_INT, nil, int32(len(instances)))

	gl.DeleteBuffers(1, &offsetVBO)
	gl.DeleteBuffers(1, &scaleVBO)
	gl.DeleteBuffers(1, &colorVBO)
	gl.DeleteBuffers(1, &aoVBO)

	gl.BindVertexArray(0)
}

func compileProgram(vertSrc, fragSrc string) (uint32, error) {
	vert, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	frag, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		return 0, fmt.Errorf("link error: %s", log)
	}

	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	return prog, nil
}

func compileShader(src string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))
		return 0, fmt.Errorf("compile error: %s", log)
	}

	return shader, nil
}

func setVec3(prog uint32, name string, v mgl32.Vec3) {
	loc := gl.GetUniformLocation(prog, gl.Str(name+"\x00"))
	gl.Uniform3f(loc, v[0], v[1], v[2])
}

func clampF32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
