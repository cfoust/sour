package preview

import "math"

// aoGridDepth is the resolution of the AO occupancy grid (128³ = ~2MB).
const aoGridDepth = 7

// fillGridDepth is the resolution for the flood fill grid (512³ = ~134MB).
// Needs to be fine enough to detect thin clip barriers.
const fillGridDepth = 9

// OccupancyGrid is a 3D boolean grid used for AO and camera placement.
type OccupancyGrid struct {
	Size     int
	Shift    int // right-shift to map coordDepth coords to this grid
	Occupied []bool
}

// BuildOccupancyGrid creates an occupancy grid at AO resolution from voxels.
func BuildOccupancyGrid(voxels []Voxel, gridSize int) *OccupancyGrid {
	return BuildOccupancyGridAtDepth(voxels, gridSize, aoGridDepth)
}

// BuildFillGrid creates an occupancy grid at flood fill resolution from voxels.
// Unlike the AO grid, this also marks lava and death-material cells as
// occupied so the flood fill treats them as impassable.
func BuildFillGrid(voxels []Voxel, gridSize int) *OccupancyGrid {
	grid := BuildOccupancyGridAtDepth(voxels, gridSize, fillGridDepth)

	// Mark lava/death voxels as occupied (they're barriers to the player)
	shift := grid.Shift
	for _, v := range voxels {
		matBits := v.Flags & 0x1C // bits 2-4
		if matBits != FlagLava && matBits != FlagDeath {
			continue
		}
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
					if x < grid.Size && y < grid.Size && z < grid.Size {
						grid.Occupied[grid.idx(x, y, z)] = true
					}
				}
			}
		}
	}

	return grid
}

// BuildOccupancyGridAtDepth creates an occupancy grid at a given depth.
func BuildOccupancyGridAtDepth(voxels []Voxel, gridSize int, depth int) *OccupancyGrid {
	aoSize := 1 << depth
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

// BuildVisualGrid creates an occupancy grid at flood fill resolution that
// excludes invisible materials (clip, death) and non-occluding materials
// (water, lava). Used for occlusion checks where only visible geometry matters.
func BuildVisualGrid(voxels []Voxel, gridSize int) *OccupancyGrid {
	aoSize := 1 << fillGridDepth
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
		if v.PaletteIndex == 0 {
			continue
		}
		mat := v.Flags & 0x1c
		if mat == FlagClip || mat == FlagDeath || mat == FlagWater || mat == FlagLava {
			continue
		}
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

// ComputeAO computes ambient occlusion using multi-radius hemisphere sampling.
// Samples at 3 radii (1, 3, 7) with distance-weighted falloff for smoother,
// more spatially-aware occlusion than single-radius neighbor counting.
func ComputeAO(voxels []Voxel, grid *OccupancyGrid) {
	// 14 sample directions: 6 axis-aligned + 8 diagonal
	type dir3 = [3]int
	dirs := [14]dir3{
		{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1},
		{1, 1, 1}, {1, 1, -1}, {1, -1, 1}, {1, -1, -1},
		{-1, 1, 1}, {-1, 1, -1}, {-1, -1, 1}, {-1, -1, -1},
	}
	radii := [3]int{1, 3, 7}
	weights := [3]float64{0.50, 0.30, 0.20}

	for i := range voxels {
		size := voxels[i].Size()
		cx := (int(voxels[i].X) + size/2) >> grid.Shift
		cy := (int(voxels[i].Y) + size/2) >> grid.Shift
		cz := (int(voxels[i].Z) + size/2) >> grid.Shift

		occlusion := 0.0
		for ri, r := range radii {
			hits := 0
			for _, d := range dirs {
				if grid.Get(cx+d[0]*r, cy+d[1]*r, cz+d[2]*r) {
					hits++
				}
			}
			occlusion += weights[ri] * float64(hits) / 14.0
		}

		voxels[i].AO = uint8((1.0 - occlusion) * 255)
	}
}

// FindBestCameraAngleReachable samples candidate camera positions and picks
// the angle that maximizes visibility of the reachable play area after cutaway.
func FindBestCameraAngleReachable(grid *OccupancyGrid, reachable map[gridPos]bool, clipZ int, focusX, focusY, focusZ, radius float64) (uint16, uint16) {
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

			score := ScoreCameraAngleWithReachable(grid, reachable, clipZ, camX, camY, camZ, focusX, focusY, focusZ)

			if score < bestScore {
				bestScore = score
				bestYaw = yawDeg
				bestPitch = pitchDeg
			}
		}
	}

	return uint16(bestYaw * 10), uint16(bestPitch * 10)
}

// Old rayMarchScore removed — replaced by ScoreCameraAngleWithReachable in reachable.go
