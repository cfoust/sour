package preview

import (
	"image"
	"image/color"
	"math"
	"sync"
)

type RenderConfig struct {
	Width  int
	Height int
}

// Sparse voxel lookup using a hash map. No resolution loss.
type voxelLookup struct {
	// Map from grid cell to voxel index. For variable-size voxels,
	// we store at the corner position with size info.
	entries map[[3]int32]int
	sizes   []int // cached sizes per voxel
}

func buildLookup(p *MapPreview, clipZ int, focusX, focusY, focusZ, radius float64) (*voxelLookup, *voxelLookup) {
	solid := &voxelLookup{
		entries: make(map[[3]int32]int, len(p.Voxels)),
		sizes:   make([]int, len(p.Voxels)),
	}
	liquid := &voxelLookup{
		entries: make(map[[3]int32]int),
		sizes:   make([]int, len(p.Voxels)),
	}
	for i := range p.Voxels {
		v := &p.Voxels[i]
		if int(v.Z) >= clipZ {
			continue
		}
		size := v.Size()
		if v.PaletteIndex == 0 {
			continue
		}
		mat := v.Flags & 0x1c
		if mat == FlagWater || mat == FlagLava {
			liquid.sizes[i] = size
			liquid.entries[[3]int32{int32(v.X), int32(v.Y), int32(v.Z)}] = i + 1
		} else {
			solid.sizes[i] = size
			solid.entries[[3]int32{int32(v.X), int32(v.Y), int32(v.Z)}] = i + 1
		}
	}
	return solid, liquid
}

// hit tests a point against the voxel lookup. Returns voxel index or -1.
func (lk *voxelLookup) hit(x, y, z int) int {
	// Check this exact cell
	if idx, ok := lk.entries[[3]int32{int32(x), int32(y), int32(z)}]; ok {
		return idx - 1
	}
	// Check if we're inside a larger voxel by checking aligned positions
	// Try power-of-2 aligned origins
	for shift := 1; shift <= 8; shift++ {
		mask := (1 << shift) - 1
		ox := x &^ mask
		oy := y &^ mask
		oz := z &^ mask
		if idx, ok := lk.entries[[3]int32{int32(ox), int32(oy), int32(oz)}]; ok {
			vi := idx - 1
			size := lk.sizes[vi]
			if size >= (1 << shift) {
				return vi
			}
		}
	}
	return -1
}

func Render(p *MapPreview, cfg RenderConfig) *image.RGBA {
	w, h := cfg.Width, cfg.Height
	gridSize := int(p.GridSize)
	clipZ := int(p.ClipY)
	if clipZ <= 0 || clipZ >= gridSize {
		clipZ = gridSize
	}

	focusX := float64(p.FocusX)
	focusY := float64(p.FocusY)
	focusZ := float64(p.FocusZ)
	radius := float64(p.FocusRadius)

	solidLk, liquidLk := buildLookup(p, clipZ, focusX, focusY, focusZ, radius)

	yawRad := float64(p.CameraYaw) / 10.0 * math.Pi / 180.0
	pitchRad := float64(p.CameraPitch) / 10.0 * math.Pi / 180.0

	camX := focusX + math.Cos(pitchRad)*math.Cos(yawRad)*radius
	camY := focusY + math.Cos(pitchRad)*math.Sin(yawRad)*radius
	camZ := focusZ + math.Sin(pitchRad)*radius

	// If camera is inside geometry, force a clip plane at the focus Z
	// and reposition camera above it. The interactive viewer handles this
	// via rotation, but a static render needs the clip to see inside.
	if solidLk.hit(int(camX), int(camY), int(camZ)) >= 0 {
		clipZ = int(focusZ) + int(radius*0.15)
		camZ = float64(clipZ) + radius*0.6
		camX = focusX + math.Cos(yawRad)*radius*0.4
		camY = focusY + math.Sin(yawRad)*radius*0.4
	}

	fwd := normalize([3]float64{focusX - camX, focusY - camY, focusZ - camZ})
	worldUp := [3]float64{0, 0, 1}
	right := normalize(cross(worldUp, fwd))
	up := cross(fwd, right)

	fov := 45.0 * math.Pi / 180.0
	halfTan := math.Tan(fov / 2)
	aspect := float64(w) / float64(h)

	// Lighting (match viewer)
	sunDir := normalize([3]float64{1.5, 0.5, 2.0})
	sunColor := [3]float64{1.0 * 1.2, 0.93 * 1.2, 0.87 * 1.2}
	fillDir := normalize([3]float64{-1, -1, 0.3})
	fillColor := [3]float64{0.4 * 0.4, 0.53 * 0.4, 0.67 * 0.4}
	hemiSky := [3]float64{0.53 * 0.5, 0.6 * 0.5, 0.8 * 0.5}
	hemiGround := [3]float64{0.33 * 0.5, 0.27 * 0.5, 0.2 * 0.5}

	skyR := math.Min(1, float64(p.SkyTop[0])/255*1.5+0.05)
	skyG := math.Min(1, float64(p.SkyTop[1])/255*1.5+0.05)
	skyB := math.Min(1, float64(p.SkyTop[2])/255*1.5+0.08)

	jitterHash := func(x, y, z int) float64 {
		h := (x*374761393 + y*668265263 + z*1274126177)
		h = ((h ^ (h >> 13)) * 1031)
		return float64((h^(h>>16))&0xffff) / 65535.0
	}

	// Render to float buffer for post-processing — parallelize by row
	buf := make([][3]float64, w*h)

	// Shade a solid voxel hit
	shadeVoxel := func(vox *Voxel, dir [3]float64) [3]float64 {
		palIdx := int(vox.PaletteIndex)
		if palIdx >= len(p.Palette) {
			palIdx = 0
		}
		cr := float64(p.Palette[palIdx][0]) / 255
		cg := float64(p.Palette[palIdx][1]) / 255
		cb := float64(p.Palette[palIdx][2]) / 255

		lum := 0.299*cr + 0.587*cg + 0.114*cb
		cr = math.Max(0, math.Min(1, (lum+(cr-lum)*2.2)*1.3))
		cg = math.Max(0, math.Min(1, (lum+(cg-lum)*2.2)*1.3))
		cb = math.Max(0, math.Min(1, (lum+(cb-lum)*2.2)*1.3))

		ao := 0.2 + 0.8*float64(vox.AO)/255
		jitter := 0.85 + 0.3*jitterHash(int(vox.X), int(vox.Y), int(vox.Z))

		nx, ny, nz := -dir[0], -dir[1], -dir[2]
		hemiT := nz*0.5 + 0.5
		lr := hemiGround[0]*(1-hemiT) + hemiSky[0]*hemiT
		lg := hemiGround[1]*(1-hemiT) + hemiSky[1]*hemiT
		lb := hemiGround[2]*(1-hemiT) + hemiSky[2]*hemiT
		sunDot := math.Max(0, nx*sunDir[0]+ny*sunDir[1]+nz*sunDir[2])
		fillDot := math.Max(0, nx*fillDir[0]+ny*fillDir[1]+nz*fillDir[2])
		lr += sunColor[0]*sunDot + fillColor[0]*fillDot
		lg += sunColor[1]*sunDot + fillColor[1]*fillDot
		lb += sunColor[2]*sunDot + fillColor[2]*fillDot

		return [3]float64{
			math.Min(1, cr*lr*ao*jitter),
			math.Min(1, cg*lg*ao*jitter),
			math.Min(1, cb*lb*ao*jitter),
		}
	}

	// Water/lava color constants
	waterColor := [3]float64{0.13, 0.4, 0.8}
	lavaColor := [3]float64{1.0, 0.27, 0.0}

	var wg sync.WaitGroup
	numWorkers := 8
	rowsPerWorker := (h + numWorkers - 1) / numWorkers

	for worker := 0; worker < numWorkers; worker++ {
		startRow := worker * rowsPerWorker
		endRow := startRow + rowsPerWorker
		if endRow > h {
			endRow = h
		}
		wg.Add(1)
		go func(startRow, endRow int) {
			defer wg.Done()
			for py := startRow; py < endRow; py++ {
				for px := 0; px < w; px++ {
					u := (2.0*float64(px)/float64(w) - 1.0) * halfTan * aspect
					v := (1.0 - 2.0*float64(py)/float64(h)) * halfTan

					dir := normalize([3]float64{
						fwd[0] + right[0]*u + up[0]*v,
						fwd[1] + right[1]*u + up[1]*v,
						fwd[2] + right[2]*u + up[2]*v,
					})

					idx := py*w + px

					liquidIdx, liquidDist := ddaMarch(camX, camY, camZ, dir[0], dir[1], dir[2], liquidLk, gridSize)
					solidIdx, solidDist := ddaMarch(camX, camY, camZ, dir[0], dir[1], dir[2], solidLk, gridSize)

					// Determine background behind liquid
					var bg [3]float64
					if solidIdx >= 0 {
						bg = shadeVoxel(&p.Voxels[solidIdx], dir)
					} else {
						bg = [3]float64{skyR, skyG, skyB}
					}

					// Blend liquid on top only if it's in front of the solid
					if liquidIdx >= 0 && (solidIdx < 0 || liquidDist < solidDist) {
						lv := &p.Voxels[liquidIdx]
						mat := lv.Flags & 0x1c
						var lc [3]float64
						var alpha float64
						if mat == FlagLava {
							lc = lavaColor
							alpha = 0.85
						} else {
							lc = waterColor
							alpha = 0.45
						}
						bg[0] = bg[0]*(1-alpha) + lc[0]*alpha
						bg[1] = bg[1]*(1-alpha) + lc[1]*alpha
						bg[2] = bg[2]*(1-alpha) + lc[2]*alpha
					}

					buf[idx] = bg
				}
			}
		}(startRow, endRow)
	}
	wg.Wait()

	// Post-processing: vignette for diorama look
	applyVignette(buf, w, h)
	out := buf

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			c := out[py*w+px]
			img.SetRGBA(px, py, color.RGBA{
				R: clampU8(c[0] * 255),
				G: clampU8(c[1] * 255),
				B: clampU8(c[2] * 255),
				A: 255,
			})
		}
	}

	return img
}

// ddaMarch returns (voxelIndex, distance) or (-1, 0).
func ddaMarch(ox, oy, oz, dx, dy, dz float64, lk *voxelLookup, gSize int) (int, float64) {
	tMin := 0.0
	tMax := float64(gSize) * 2

	for axis := 0; axis < 3; axis++ {
		var d, o float64
		switch axis {
		case 0:
			d, o = dx, ox
		case 1:
			d, o = dy, oy
		case 2:
			d, o = dz, oz
		}
		if math.Abs(d) < 1e-10 {
			if o < 0 || o >= float64(gSize) {
				return -1, 0
			}
			continue
		}
		t1 := -o / d
		t2 := (float64(gSize) - o) / d
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		if t1 > tMin {
			tMin = t1
		}
		if t2 < tMax {
			tMax = t2
		}
	}
	if tMin >= tMax {
		return -1, 0
	}
	if tMin < 0 {
		tMin = 0
	}

	px := ox + dx*(tMin+0.001)
	py := oy + dy*(tMin+0.001)
	pz := oz + dz*(tMin+0.001)

	ix := clampI(int(math.Floor(px)), 0, gSize-1)
	iy := clampI(int(math.Floor(py)), 0, gSize-1)
	iz := clampI(int(math.Floor(pz)), 0, gSize-1)

	stepX, stepY, stepZ := 1, 1, 1
	if dx < 0 {
		stepX = -1
	}
	if dy < 0 {
		stepY = -1
	}
	if dz < 0 {
		stepZ = -1
	}

	tDeltaX := math.Abs(1.0 / dx)
	tDeltaY := math.Abs(1.0 / dy)
	tDeltaZ := math.Abs(1.0 / dz)
	if math.Abs(dx) < 1e-10 {
		tDeltaX = 1e18
	}
	if math.Abs(dy) < 1e-10 {
		tDeltaY = 1e18
	}
	if math.Abs(dz) < 1e-10 {
		tDeltaZ = 1e18
	}

	var tmX, tmY, tmZ float64
	if dx > 0 {
		tmX = (float64(ix+1) - px) / dx
	} else if dx < 0 {
		tmX = (float64(ix) - px) / dx
	} else {
		tmX = 1e18
	}
	if dy > 0 {
		tmY = (float64(iy+1) - py) / dy
	} else if dy < 0 {
		tmY = (float64(iy) - py) / dy
	} else {
		tmY = 1e18
	}
	if dz > 0 {
		tmZ = (float64(iz+1) - pz) / dz
	} else if dz < 0 {
		tmZ = (float64(iz) - pz) / dz
	} else {
		tmZ = 1e18
	}

	t := tMin
	maxSteps := gSize * 3
	for step := 0; step < maxSteps; step++ {
		if ix < 0 || ix >= gSize || iy < 0 || iy >= gSize || iz < 0 || iz >= gSize {
			break
		}

		hit := lk.hit(ix, iy, iz)
		if hit >= 0 {
			return hit, t
		}

		if tmX < tmY {
			if tmX < tmZ {
				t = tmX
				ix += stepX
				tmX += tDeltaX
			} else {
				t = tmZ
				iz += stepZ
				tmZ += tDeltaZ
			}
		} else {
			if tmY < tmZ {
				t = tmY
				iy += stepY
				tmY += tDeltaY
			} else {
				t = tmZ
				iz += stepZ
				tmZ += tDeltaZ
			}
		}
	}

	return -1, 0
}

// tiltShiftBlur applies a horizontal blur that increases toward top/bottom edges.
func tiltShiftBlur(buf [][3]float64, w, h int) [][3]float64 {
	out := make([][3]float64, w*h)
	copy(out, buf)

	// Focus band: center 40% of the image is sharp
	focusCenter := float64(h) / 2
	focusBand := float64(h) * 0.2

	for py := 0; py < h; py++ {
		dist := math.Abs(float64(py) - focusCenter)
		blurAmount := math.Max(0, (dist-focusBand)/float64(h)*2)
		radius := int(blurAmount * 8)
		if radius <= 0 {
			continue
		}

		for px := 0; px < w; px++ {
			var r, g, b float64
			count := 0
			for dx := -radius; dx <= radius; dx++ {
				for dy := -radius / 2; dy <= radius/2; dy++ {
					sx := px + dx
					sy := py + dy
					if sx < 0 || sx >= w || sy < 0 || sy >= h {
						continue
					}
					c := buf[sy*w+sx]
					r += c[0]
					g += c[1]
					b += c[2]
					count++
				}
			}
			if count > 0 {
				out[py*w+px] = [3]float64{r / float64(count), g / float64(count), b / float64(count)}
			}
		}
	}

	return out
}

// applyVignette darkens corners for a diorama look.
func applyVignette(buf [][3]float64, w, h int) {
	cx, cy := float64(w)/2, float64(h)/2
	maxDist := math.Sqrt(cx*cx + cy*cy)

	for py := 0; py < h; py++ {
		for px := 0; px < w; px++ {
			dx := float64(px) - cx
			dy := float64(py) - cy
			dist := math.Sqrt(dx*dx+dy*dy) / maxDist
			// Smooth vignette: starts at 50% from center
			v := 1.0 - math.Max(0, (dist-0.5))*1.2
			if v < 0.3 {
				v = 0.3
			}
			idx := py*w + px
			buf[idx][0] *= v
			buf[idx][1] *= v
			buf[idx][2] *= v
		}
	}
}

func normalize(v [3]float64) [3]float64 {
	l := math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
	if l < 1e-10 {
		return v
	}
	return [3]float64{v[0] / l, v[1] / l, v[2] / l}
}

func cross(a, b [3]float64) [3]float64 {
	return [3]float64{
		a[1]*b[2] - a[2]*b[1],
		a[2]*b[0] - a[0]*b[2],
		a[0]*b[1] - a[1]*b[0],
	}
}

func clampU8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func clampI(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
