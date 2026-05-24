package preview

import "github.com/cfoust/sour/pkg/maps"

// AddBarriersFromOctree walks the map octree and marks death, lava, and clip
// material cells as occupied in the fill grid. These materials are typically
// on empty cubes (invisible barriers), so they're missed by the voxel-based
// occupancy grid.
func AddBarriersFromOctree(grid *OccupancyGrid, root *maps.Cube, worldSize int) {
	if root == nil || len(root.Children) == 0 {
		return
	}
	gridSize := grid.Size
	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkBarriers(child, ox, oy, oz, 1, grid)
	}
}

func walkBarriers(c *maps.Cube, ox, oy, oz, depth int, grid *OccupancyGrid) {
	if c == nil {
		return
	}

	// Log2 of grid size gives us the max depth for this grid
	maxDepth := 0
	for s := grid.Size; s > 1; s >>= 1 {
		maxDepth++
	}

	span := 1 << (maxDepth - depth)

	if len(c.Children) > 0 && depth < maxDepth {
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			nx := ox + (i&1)*half
			ny := oy + ((i>>1)&1)*half
			nz := oz + ((i>>2)&1)*half
			walkBarriers(child, nx, ny, nz, depth+1, grid)
		}
		return
	}

	// Check if this cube has a barrier material
	mat := c.Material
	isBarrier := false
	if mat&maps.MAT_DEATH != 0 {
		isBarrier = true
	}
	if mat&maps.MATF_VOLUME == maps.MAT_LAVA {
		isBarrier = true
	}
	if mat&maps.MATF_CLIP == maps.MAT_CLIP {
		isBarrier = true
	}

	if !isBarrier {
		return
	}

	// Mark all cells this cube covers as occupied
	for dz := 0; dz < span; dz++ {
		for dy := 0; dy < span; dy++ {
			for dx := 0; dx < span; dx++ {
				x, y, z := ox+dx, oy+dy, oz+dz
				if x < grid.Size && y < grid.Size && z < grid.Size {
					grid.Occupied[grid.idx(x, y, z)] = true
				}
			}
		}
	}
}
