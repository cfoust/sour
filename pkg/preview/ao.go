package preview

// aoGridDepth is the resolution of the AO occupancy grid.
// Using 128³ (depth 7) keeps memory at ~2MB regardless of coordDepth.
const aoGridDepth = 7

// ComputeAO computes simple ambient occlusion for each voxel by counting
// occupied neighbors. Uses a fixed-resolution occupancy grid (128³) and
// maps variable-size voxels into it.
func ComputeAO(voxels []Voxel, gridSize int) {
	aoSize := 1 << aoGridDepth
	// Shift to map from coordDepth grid to aoGrid
	shift := 0
	for s := gridSize; s > aoSize; s >>= 1 {
		shift++
	}

	occupied := make([]bool, aoSize*aoSize*aoSize)
	idx := func(x, y, z int) int {
		return x + y*aoSize + z*aoSize*aoSize
	}

	// Fill occupancy grid, mapping voxels to AO grid resolution.
	for _, v := range voxels {
		size := v.Size()
		// Map to AO grid coordinates
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
						occupied[idx(x, y, z)] = true
					}
				}
			}
		}
	}

	// Compute AO by sampling 3x3x3 neighborhood at voxel center in AO grid.
	for i := range voxels {
		size := voxels[i].Size()
		// Center in AO grid
		cx := (int(voxels[i].X) + size/2) >> shift
		cy := (int(voxels[i].Y) + size/2) >> shift
		cz := (int(voxels[i].Z) + size/2) >> shift

		count := 0
		for dz := -1; dz <= 1; dz++ {
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 && dz == 0 {
						continue
					}
					nx, ny, nz := cx+dx, cy+dy, cz+dz
					if nx < 0 || nx >= aoSize || ny < 0 || ny >= aoSize || nz < 0 || nz >= aoSize {
						continue
					}
					if occupied[idx(nx, ny, nz)] {
						count++
					}
				}
			}
		}
		voxels[i].AO = uint8(255 - count*255/26)
	}
}
