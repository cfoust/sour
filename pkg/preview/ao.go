package preview

import "math"

// aoGridDepth is the resolution of the occupancy grid.
// Using 128³ (depth 7) keeps memory at ~2MB regardless of coordDepth.
const aoGridDepth = 7

// OccupancyGrid is a 3D boolean grid used for AO and camera placement.
type OccupancyGrid struct {
	Size     int
	Shift    int // right-shift to map coordDepth coords to this grid
	Occupied []bool
}

// BuildOccupancyGrid creates a fixed-resolution occupancy grid from voxels.
func BuildOccupancyGrid(voxels []Voxel, gridSize int) *OccupancyGrid {
	aoSize := 1 << aoGridDepth
	shift := 0
	for s := gridSize; s > aoSize; s >>= 1 {
		shift++
	}

	g := &OccupancyGrid{
		Size:     aoSize,
		Shift:    shift,
		Occupied: make([]bool, aoSize*aoSize*aoSize),
	}

	for _, v := range voxels {
		size := v.Size()
		ax := int(v.X) >> shift
		ay := int(v.Y) >> shift
		az := int(v.Z) >> shift
		asize := size >> shift
		if asize < 1 {
			asize = 1
		}
		for dz := 0; dz < asize; dz++ {
			for dy := 0; dy < asize; dy++ {
				for dx := 0; dx < asize; dx++ {
					x, y, z := ax+dx, ay+dy, az+dz
					if x < aoSize && y < aoSize && z < aoSize {
						g.Occupied[g.idx(x, y, z)] = true
					}
				}
			}
		}
	}

	return g
}

func (g *OccupancyGrid) idx(x, y, z int) int {
	return x + y*g.Size + z*g.Size*g.Size
}

func (g *OccupancyGrid) Get(x, y, z int) bool {
	if x < 0 || x >= g.Size || y < 0 || y >= g.Size || z < 0 || z >= g.Size {
		return false
	}
	return g.Occupied[g.idx(x, y, z)]
}

// ComputeAO computes ambient occlusion using a prebuilt occupancy grid.
func ComputeAO(voxels []Voxel, grid *OccupancyGrid) {
	for i := range voxels {
		size := voxels[i].Size()
		cx := (int(voxels[i].X) + size/2) >> grid.Shift
		cy := (int(voxels[i].Y) + size/2) >> grid.Shift
		cz := (int(voxels[i].Z) + size/2) >> grid.Shift

		count := 0
		for dz := -1; dz <= 1; dz++ {
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 && dz == 0 {
						continue
					}
					if grid.Get(cx+dx, cy+dy, cz+dz) {
						count++
					}
				}
			}
		}
		voxels[i].AO = uint8(255 - count*255/26)
	}
}

// FindBestCameraAngle samples candidate camera positions on a sphere around
// the focus point and picks the (yaw, pitch) with the clearest line of sight.
// Returns yaw in tenths of degrees (0-3599) and pitch in tenths of degrees (0-900).
func FindBestCameraAngle(grid *OccupancyGrid, focusX, focusY, focusZ, radius float64) (uint16, uint16) {
	bestYaw, bestPitch := 0.0, 30.0
	bestScore := math.MaxInt32

	for yawDeg := 0.0; yawDeg < 360; yawDeg += 5 {
		for _, pitchDeg := range []float64{20, 30, 40, 50} {
			yawRad := yawDeg * math.Pi / 180
			pitchRad := pitchDeg * math.Pi / 180

			// Camera position in grid coords (Sauer: X right, Y forward, Z up)
			camX := focusX + math.Cos(pitchRad)*math.Cos(yawRad)*radius
			camY := focusY + math.Cos(pitchRad)*math.Sin(yawRad)*radius
			camZ := focusZ + math.Sin(pitchRad)*radius

			score := rayMarchScore(grid, camX, camY, camZ, focusX, focusY, focusZ)

			if score < bestScore {
				bestScore = score
				bestYaw = yawDeg
				bestPitch = pitchDeg
			}
		}
	}

	return uint16(bestYaw * 10), uint16(bestPitch * 10)
}

// rayMarchScore counts occupied cells along a ray from camera to focus.
// Also heavily penalizes if the camera itself is inside geometry.
func rayMarchScore(grid *OccupancyGrid, cx, cy, cz, fx, fy, fz float64) int {
	dx := fx - cx
	dy := fy - cy
	dz := fz - cz
	dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
	if dist < 1 {
		return 0
	}

	steps := int(dist) + 1
	if steps > 200 {
		steps = 200
	}

	score := 0

	// Heavy penalty if camera position is in geometry
	gx := int(cx) >> grid.Shift
	gy := int(cy) >> grid.Shift
	gz := int(cz) >> grid.Shift
	if grid.Get(gx, gy, gz) {
		score += 1000
	}

	// March from camera toward focus, count hits
	stepX := dx / float64(steps)
	stepY := dy / float64(steps)
	stepZ := dz / float64(steps)

	for i := 0; i < steps; i++ {
		px := cx + stepX*float64(i)
		py := cy + stepY*float64(i)
		pz := cz + stepZ*float64(i)

		gx := int(px) >> grid.Shift
		gy := int(py) >> grid.Shift
		gz := int(pz) >> grid.Shift

		if grid.Get(gx, gy, gz) {
			// Hits near the camera are worse than hits near the focus
			weight := steps - i
			score += weight
		}
	}

	return score
}
