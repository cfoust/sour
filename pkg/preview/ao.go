package preview

// ComputeAO computes simple ambient occlusion for each voxel by counting
// occupied neighbors in a 3x3x3 neighborhood (26 neighbors).
func ComputeAO(voxels []Voxel, gridSize int) {
	// Build 3D occupancy grid.
	occupied := make([]bool, gridSize*gridSize*gridSize)
	idx := func(x, y, z int) int {
		return x + y*gridSize + z*gridSize*gridSize
	}
	for _, v := range voxels {
		occupied[idx(int(v.X), int(v.Y), int(v.Z))] = true
	}

	for i := range voxels {
		x, y, z := int(voxels[i].X), int(voxels[i].Y), int(voxels[i].Z)
		count := 0
		for dz := -1; dz <= 1; dz++ {
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 && dz == 0 {
						continue
					}
					nx, ny, nz := x+dx, y+dy, z+dz
					if nx < 0 || nx >= gridSize || ny < 0 || ny >= gridSize || nz < 0 || nz >= gridSize {
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
