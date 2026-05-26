package preview

import "github.com/cfoust/sour/pkg/maps"

// BuildFillGridFromOctree builds a fill-resolution occupancy grid directly
// from the octree, without extracting voxels first. Marks any non-empty cube
// as occupied, plus barrier materials (clip, death, lava).
func BuildFillGridFromOctree(root *maps.Cube, worldSize int) *OccupancyGrid {
	gridSize := 1 << fillGridDepth
	coordGrid := 1 << coordDepth
	shift := 0
	for s := coordGrid; s > gridSize; s >>= 1 {
		shift++
	}
	grid := &OccupancyGrid{
		Size:     gridSize,
		Shift:    shift,
		Occupied: make([]bool, gridSize*gridSize*gridSize),
	}

	if root == nil || len(root.Children) == 0 {
		return grid
	}

	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkOctreeOccupancy(child, ox, oy, oz, 1, grid)
	}

	return grid
}

func walkOctreeOccupancy(c *maps.Cube, ox, oy, oz, depth int, grid *OccupancyGrid) {
	if c == nil {
		return
	}

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
			walkOctreeOccupancy(child, nx, ny, nz, depth+1, grid)
		}
		return
	}

	// Mark occupied if non-empty geometry or barrier material
	isSolid := len(c.Children) == 0 && !c.IsEmpty()
	mat := c.Material
	isBarrier := mat&maps.MAT_DEATH != 0 ||
		mat&maps.MATF_VOLUME == maps.MAT_LAVA ||
		mat&maps.MATF_CLIP == maps.MAT_CLIP

	if !isSolid && !isBarrier {
		return
	}

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
