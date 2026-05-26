package preview

import "github.com/cfoust/sour/pkg/maps"

// Coordinate grid depth for adaptive extraction (1024³ grid).
// At worldSize=1024 this gives 1-unit resolution at the finest level.
const coordDepth = 10

// Depth levels for adaptive detail.
const (
	detailMax = 10 // finest: 1-unit voxels near play area
	detailMid = 8  // 4-unit voxels at medium distance
	detailLow = 7  // 8-unit voxels far from play area
	detailMin = 7  // 8-unit blocks minimum — keeps thin geometry readable
)

type worldPos struct {
	X, Y, Z float32
}

// ExtractVoxels walks the octree at a uniform depth and emits one Voxel per
// occupied region. Each voxel stores its size in the Flags field.
// Used by the bench tool for comparison.
func ExtractVoxels(root *maps.Cube, worldSize int, maxDepth int, paletteMap map[uint16]uint8) []Voxel {
	var voxels []Voxel
	if root == nil || len(root.Children) == 0 {
		return voxels
	}
	gridSize := 1 << maxDepth
	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkCubeUniform(child, ox, oy, oz, 1, maxDepth, paletteMap, &voxels)
	}
	return voxels
}

func walkCubeUniform(c *maps.Cube, ox, oy, oz, depth, maxDepth int, paletteMap map[uint16]uint8, voxels *[]Voxel) {
	if c == nil {
		return
	}

	span := 1 << (maxDepth - depth)

	if len(c.Children) > 0 && depth < maxDepth {
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			cx := ox + (i&1)*half
			cy := oy + ((i>>1)&1)*half
			cz := oz + ((i>>2)&1)*half
			walkCubeUniform(child, cx, cy, cz, depth+1, maxDepth, paletteMap, voxels)
		}
		return
	}

	// At maxDepth with children, or a leaf — walk to actual leaves.
	emitLeaves(c, ox, oy, oz, depth, maxDepth, maxDepth, paletteMap, nil, voxels)
}

// PlayAreaGrid is a coarse boolean grid marking cells near the reachable
// play area. Used to drive adaptive voxel extraction density.
type PlayAreaGrid struct {
	Size  int
	Shift int // right-shift coord grid → this grid
	Cells []bool
}

// BuildPlayAreaGrid dilates the reachable set into a coarse grid that marks
// cells within `margin` cells of any reachable position. Cubes overlapping
// marked cells get maximum extraction depth.
func BuildPlayAreaGrid(reachable map[gridPos]bool, fillGridSize, worldSize int) *PlayAreaGrid {
	// Use AO grid resolution (128³) for the proximity check
	size := 1 << aoGridDepth
	coordGrid := 1 << coordDepth
	shift := 0
	for s := coordGrid; s > size; s >>= 1 {
		shift++
	}

	fillToProx := fillGridSize / size

	grid := &PlayAreaGrid{
		Size:  size,
		Shift: shift,
		Cells: make([]bool, size*size*size),
	}

	margin := 8 // cells in proximity grid ≈ 8/128 = 6% of map

	for p := range reachable {
		// Convert fill grid coords to proximity grid coords
		px := p.X / fillToProx
		py := p.Y / fillToProx
		pz := p.Z / fillToProx

		for dz := -margin; dz <= margin; dz++ {
			for dy := -margin; dy <= margin; dy++ {
				for dx := -margin; dx <= margin; dx++ {
					x, y, z := px+dx, py+dy, pz+dz
					if x >= 0 && x < size && y >= 0 && y < size && z >= 0 && z < size {
						grid.Cells[x+y*size+z*size*size] = true
					}
				}
			}
		}
	}

	return grid
}

func (g *PlayAreaGrid) Get(coordX, coordY, coordZ int) bool {
	x := coordX >> g.Shift
	y := coordY >> g.Shift
	z := coordZ >> g.Shift
	if x < 0 || x >= g.Size || y < 0 || y >= g.Size || z < 0 || z >= g.Size {
		return false
	}
	return g.Cells[x+y*g.Size+z*g.Size*g.Size]
}

// ExtractVoxelsAdaptive walks the octree with variable depth based on
// proximity to the play area. Near the play area: fine detail (detailMax).
// Far from play area: coarse blocks (detailMin).
// If playArea is nil, falls back to uniform detailMid everywhere.
func ExtractVoxelsAdaptive(root *maps.Cube, worldSize int, playArea *PlayAreaGrid, fillGrid *OccupancyGrid, paletteMap map[uint16]uint8) []Voxel {
	var voxels []Voxel
	if root == nil || len(root.Children) == 0 {
		return voxels
	}

	gridSize := 1 << coordDepth

	params := &adaptiveParams{
		playArea:   playArea,
		fillGrid:   fillGrid,
		paletteMap: paletteMap,
	}

	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkCubeAdaptive(child, ox, oy, oz, 1, params, &voxels)
	}
	return voxels
}

type adaptiveParams struct {
	playArea   *PlayAreaGrid
	fillGrid   *OccupancyGrid
	paletteMap map[uint16]uint8
}

func walkCubeAdaptive(c *maps.Cube, ox, oy, oz, depth int, p *adaptiveParams, voxels *[]Voxel) {
	if c == nil {
		return
	}

	span := 1 << (coordDepth - depth)

	// Extract at maximum detail everywhere. The post-extraction simplifier
	// (SimplifyVoxels) handles LOD reduction using error-driven octree
	// collapse, which automatically preserves thin features like bridges.
	targetDepth := detailMax

	// If we have children and haven't reached target depth, recurse
	if len(c.Children) > 0 && depth < targetDepth {
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			nx := ox + (i&1)*half
			ny := oy + ((i>>1)&1)*half
			nz := oz + ((i>>2)&1)*half
			walkCubeAdaptive(child, nx, ny, nz, depth+1, p, voxels)
		}
		return
	}

	// Reached target depth or leaf — walk to actual leaves to preserve
	// the octree's own spatial structure (rooms, passages, etc).
	emitLeaves(c, ox, oy, oz, depth, targetDepth, coordDepth, p.paletteMap, p.fillGrid, voxels)
}

// emitLeaves recursively walks a cube's descendants and emits a voxel for
// each non-empty leaf. Intermediate nodes are never merged into single
// solid blocks — this preserves empty spaces (rooms, corridors).
// targetDepth is the detail level the adaptive system chose for this region.
// Stops at maxCoordDepth to avoid sub-grid detail.
func emitLeaves(c *maps.Cube, ox, oy, oz, depth, targetDepth, maxCoordDepth int, paletteMap map[uint16]uint8, grid *OccupancyGrid, voxels *[]Voxel) {
	if c == nil {
		return
	}

	// If this cube has children and we haven't hit the coordinate limit,
	// recurse to find the actual leaf geometry.
	if len(c.Children) > 0 && depth < maxCoordDepth {
		span := 1 << (maxCoordDepth - depth)
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			nx := ox + (i&1)*half
			ny := oy + ((i>>1)&1)*half
			nz := oz + ((i>>2)&1)*half
			emitLeaves(child, nx, ny, nz, depth+1, targetDepth, maxCoordDepth, paletteMap, grid, voxels)
		}
		return
	}

	// Leaf node
	if len(c.Children) == 0 {
		if c.IsEmpty() {
			return
		}
		// Deformed cube: subdivide with volume intersection test
		if isCubeActuallyDeformed(c) {
			emitDeformedCube(c, ox, oy, oz, depth, targetDepth, maxCoordDepth, paletteMap, grid, voxels)
			return
		}
		emitVoxel(c, ox, oy, oz, depth, maxCoordDepth, paletteMap, voxels)
		return
	}

	// At coordinate limit with children: check if region has any solids.
	if isRegionEmpty(c) {
		return
	}

	flags := classifyCubeOrRegion(c, paletteMap)
	palIdx := dominantPaletteIndexRecursive(c, paletteMap)

	sizeLog2 := maxCoordDepth - depth
	if sizeLog2 < 0 {
		sizeLog2 = 0
	}
	flags |= byte(sizeLog2 << FlagSizeShift)

	*voxels = append(*voxels, Voxel{
		X:            uint16(ox),
		Y:            uint16(oy),
		Z:            uint16(oz),
		PaletteIndex: palIdx,
		Flags:        flags,
	})
}

// isSkyCube returns true if all 6 faces use the sky texture (skybox boundary).
func isSkyCube(c *maps.Cube) bool {
	for _, t := range c.Texture {
		if t != maps.DEFAULT_SKY {
			return false
		}
	}
	return true
}

// emitVoxel emits a single voxel for an entirely solid leaf cube.
func emitVoxel(c *maps.Cube, ox, oy, oz, depth, maxCoordDepth int, paletteMap map[uint16]uint8, voxels *[]Voxel) {
	if c.IsEmpty() {
		return
	}
	flags := classifyCube(c)
	palIdx := dominantPaletteIndex(c, paletteMap)
	sizeLog2 := maxCoordDepth - depth
	if sizeLog2 < 0 {
		sizeLog2 = 0
	}
	flags |= byte(sizeLog2 << FlagSizeShift)
	*voxels = append(*voxels, Voxel{
		X:            uint16(ox),
		Y:            uint16(oy),
		Z:            uint16(oz),
		PaletteIndex: palIdx,
		Flags:        flags,
	})
}

// isRegionEmpty recursively checks if all descendants are empty.
func isRegionEmpty(c *maps.Cube) bool {
	if len(c.Children) == 0 {
		return c.IsEmpty()
	}
	for _, child := range c.Children {
		if child != nil && !isRegionEmpty(child) {
			return false
		}
	}
	return true
}

func classifyCube(c *maps.Cube) byte {
	var flags byte
	if c.IsEntirelySolid() {
		flags = FlagSolid
	} else {
		flags = FlagNormal
	}

	mat := c.Material
	switch {
	case mat&maps.MAT_DEATH != 0:
		flags |= FlagDeath
	case mat&maps.MATF_VOLUME == maps.MAT_LAVA:
		flags |= FlagLava
	case mat&maps.MATF_VOLUME == maps.MAT_WATER:
		flags |= FlagWater
	case mat&maps.MATF_VOLUME == maps.MAT_GLASS:
		flags |= FlagGlass
	case mat&maps.MATF_CLIP == maps.MAT_CLIP:
		flags |= FlagClip
	}

	return flags
}

// classifyCubeOrRegion returns flags for a leaf cube, or FlagSolid for
// an intermediate node at the coordinate limit.
func classifyCubeOrRegion(c *maps.Cube, paletteMap map[uint16]uint8) byte {
	if len(c.Children) == 0 {
		return classifyCube(c)
	}
	// Intermediate node at coord limit — find first non-empty leaf's flags
	for _, child := range c.Children {
		if child == nil {
			continue
		}
		if len(child.Children) > 0 {
			f := classifyCubeOrRegion(child, paletteMap)
			if f != 0 {
				return f
			}
		} else if !child.IsEmpty() {
			return classifyCube(child)
		}
	}
	return FlagSolid
}

func dominantPaletteIndex(c *maps.Cube, paletteMap map[uint16]uint8) uint8 {
	texCounts := make(map[uint16]int)
	for _, t := range c.Texture {
		if t != maps.DEFAULT_SKY {
			texCounts[t]++
		}
	}
	// If all faces are sky, return 0 (sky marker — renderer skips these)
	if len(texCounts) == 0 {
		return 0
	}
	var dominantTex uint16 = maps.DEFAULT_GEOM
	bestCount := 0
	for t, count := range texCounts {
		if count > bestCount {
			bestCount = count
			dominantTex = t
		}
	}
	palIdx, ok := paletteMap[dominantTex]
	if !ok {
		return 0
	}
	return palIdx
}

// dominantPaletteIndexRecursive finds the palette index for a leaf cube
// or the first non-empty leaf in an intermediate node's subtree.
func dominantPaletteIndexRecursive(c *maps.Cube, paletteMap map[uint16]uint8) uint8 {
	if len(c.Children) == 0 {
		return dominantPaletteIndex(c, paletteMap)
	}
	for _, child := range c.Children {
		if child == nil {
			continue
		}
		if len(child.Children) > 0 {
			p := dominantPaletteIndexRecursive(child, paletteMap)
			if p != 0 {
				return p
			}
		} else if !child.IsEmpty() {
			return dominantPaletteIndex(child, paletteMap)
		}
	}
	return 0
}

