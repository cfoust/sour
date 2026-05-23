package main

import (
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/osrs"

	C "github.com/cfoust/sour/pkg/game/constants"
)

// Conversion parameters
//
// Scale derivation:
//   - Sauer player height ≈ 14 units
//   - OSRS player height ≈ 1 tile = 128 game units
//   - So 1 tile must ≈ 14 Sauer units → cubesPerTile = 16 (closest power of 2)
//   - Model scale: 1 OSRS unit = cubesPerTile/128 Sauer units = 0.125
//   - Height scale: same ratio, heightScale = 128/cubesPerTile = 8
//
// Verification with a Door model (240 raw units tall):
//   OBJ height = 240 * 0.125 = 30 units. Door/player = 30/14 = 2.1x ✓ (OSRS: ~2x)
// Verification with a Tree model (520 raw units tall):
//   OBJ height = 520 * 0.125 = 65 units. Tree/player = 65/14 = 4.6x ✓ (OSRS: ~4x)
const (
	cubesPerTile  = 16
	tilesPerChunk = 64
	worldSize     = 2048

	terrainCubes  = tilesPerChunk * cubesPerTile  // 1024
	terrainOffset = (worldSize - terrainCubes) / 2 // 512

	terrainBaseZ = 512
	// Same conversion as models: 1 OSRS game unit = cubesPerTile/128 Sauer units.
	// heightScale = 128/cubesPerTile: dividing OSRS height by this gives cubes.
	heightScale = 128 / cubesPerTile // 8

	wallHeight    = cubesPerTile * 2
	wallThickness = 1

	modelScaleValue = float64(cubesPerTile) / 128.0 // 0.125
)

// OSRS object types
const (
	objWallStraight   = 0
	objWallDiagonal   = 1
	objWallCorner     = 2
	objWallDecoration = 3
	objDecoration1    = 4
	objDecoration2    = 5
	objDecoration3    = 6
	objDecoration4    = 7
	objDecoration5    = 8
	objDiagonal       = 9
	objGameObject     = 10
	objGameObject2    = 11
	objGroundDecor    = 22
)

// convertRegion returns (map, modelCacheIDs, error).
// modelCacheIDs maps OSRS cache model IDs to mapmodel indices for export.
func convertRegion(
	region *osrs.Region,
	defs *osrs.FloorDefs,
	objDefs *osrs.ObjectDefs,
	objects []osrs.PlacedObject,
) (*maps.GameMap, []modelKey, map[modelKey]int, error) {
	m, err := maps.NewMap()
	if err != nil {
		return nil, nil, nil, err
	}
	m.Header.WorldSize = worldSize

	// Build a height grid at cube resolution (interpolated from OSRS tiles)
	heightGrid := buildHeightGrid(region)

	// Build the octree
	root := &maps.Cube{}
	root.Children = make([]*maps.Cube, maps.CUBE_FACTOR)
	halfSize := worldSize / 2
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		ox := (i & 1) * halfSize
		oy := ((i >> 1) & 1) * halfSize
		oz := ((i >> 2) & 1) * halfSize
		root.Children[i] = buildNode(heightGrid, region, defs, ox, oy, oz, halfSize)
	}

	m.WorldRoot = root

	// Walls are rendered as mapmodel entities alongside decorations,
	// since OSRS walls are 3D models, not axis-aligned cube geometry.

	// Set water material on appropriate cubes
	addWater(root, region, heightGrid)

	// Add entities
	cx := float32(worldSize) / 2
	cy := float32(worldSize) / 2
	spawnZ := float32(getTerrainZ(heightGrid, worldSize/2, worldSize/2)) + 4

	m.Entities = append(m.Entities, maps.Entity{
		Position: maps.Vector{X: cx, Y: cy, Z: spawnZ},
		Type:     C.EntityType(maps.ET_PLAYERSTART),
	})

	// Global sunlight
	m.Entities = append(m.Entities, maps.Entity{
		Position: maps.Vector{X: cx, Y: cy, Z: float32(worldSize) - 16},
		Type:     C.EntityType(maps.ET_LIGHT),
		Attr1:    500,
		Attr2:    220,
		Attr3:    220,
		Attr4:    200,
	})

	// Add decorative objects as mapmodels
	modelKeys, modelKeyMap := addDecorations(m, region, objects, objDefs, heightGrid)

	return m, modelKeys, modelKeyMap, nil
}

// buildHeightGrid creates a cube-resolution height grid interpolated from OSRS tile heights.
// Returns a (terrainCubes+1) x (terrainCubes+1) grid of Z values in Sauer coordinates.
func buildHeightGrid(region *osrs.Region) [][]int {
	gridSize := terrainCubes + 1
	grid := make([][]int, gridSize)
	for i := range grid {
		grid[i] = make([]int, gridSize)
	}

	for cx := 0; cx <= terrainCubes; cx++ {
		for cy := 0; cy <= terrainCubes; cy++ {
			// Bilinear interpolation between OSRS tile vertices
			// Each tile has cubesPerTile cubes, fractional position within tile
			fx := float64(cx) / float64(cubesPerTile)
			fy := float64(cy) / float64(cubesPerTile)

			tx := int(fx)
			ty := int(fy)
			// Fractional position within the tile
			dx := fx - float64(tx)
			dy := fy - float64(ty)

			if tx >= 64 {
				tx = 63
				dx = 1.0
			}
			if ty >= 64 {
				ty = 63
				dy = 1.0
			}
			tx1 := tx + 1
			ty1 := ty + 1
			if tx1 > 64 {
				tx1 = 64
			}
			if ty1 > 64 {
				ty1 = 64
			}

			// Bilinear interpolation of the 4 surrounding tile vertices
			h00 := float64(region.Heights[0][tx][ty])
			h10 := float64(region.Heights[0][tx1][ty])
			h01 := float64(region.Heights[0][tx][ty1])
			h11 := float64(region.Heights[0][tx1][ty1])

			h := h00*(1-dx)*(1-dy) + h10*dx*(1-dy) + h01*(1-dx)*dy + h11*dx*dy

			// Convert: negate, divide by scale, add base
			z := terrainBaseZ + int(-h)/heightScale
			grid[cx][cy] = z
		}
	}

	return grid
}

// getTerrainZ returns the terrain height at a given world coordinate.
func getTerrainZ(heightGrid [][]int, wx, wy int) int {
	cx := wx - terrainOffset
	cy := wy - terrainOffset
	if cx < 0 {
		cx = 0
	}
	if cy < 0 {
		cy = 0
	}
	if cx >= len(heightGrid) {
		cx = len(heightGrid) - 1
	}
	if cy >= len(heightGrid[0]) {
		cy = len(heightGrid[0]) - 1
	}
	return heightGrid[cx][cy]
}

// buildNode recursively builds octree nodes.
func buildNode(heightGrid [][]int, region *osrs.Region, defs *osrs.FloorDefs, ox, oy, oz, size int) *maps.Cube {
	if size <= 1 {
		return buildLeaf(heightGrid, region, defs, ox, oy, oz)
	}

	// Check if entire node is clearly above or below terrain
	if isEntirelyAbove(heightGrid, ox, oy, oz, size) {
		c := &maps.Cube{}
		c.EmptyFaces()
		return c
	}
	if isEntirelyBelow(heightGrid, ox, oy, oz, size) {
		c := &maps.Cube{}
		c.SolidFaces()
		// Set texture
		texIdx := uint16(1)
		for i := 0; i < 6; i++ {
			c.Texture[i] = texIdx
		}
		return c
	}

	// Subdivide
	c := &maps.Cube{}
	c.Children = make([]*maps.Cube, maps.CUBE_FACTOR)
	half := size / 2
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		cx := ox + (i&1)*half
		cy := oy + ((i>>1)&1)*half
		cz := oz + ((i>>2)&1)*half
		c.Children[i] = buildNode(heightGrid, region, defs, cx, cy, cz, half)
	}
	return c
}

func isEntirelyAbove(heightGrid [][]int, ox, oy, oz, size int) bool {
	// Check if the bottom of this cube is above all terrain in its footprint
	for x := ox; x <= ox+size; x++ {
		for y := oy; y <= oy+size; y++ {
			h := getTerrainZ(heightGrid, x, y)
			if oz < h {
				return false
			}
		}
	}
	return true
}

func isEntirelyBelow(heightGrid [][]int, ox, oy, oz, size int) bool {
	// Check if the top of this cube is below all terrain in its footprint
	top := oz + size
	for x := ox; x <= ox+size; x++ {
		for y := oy; y <= oy+size; y++ {
			h := getTerrainZ(heightGrid, x, y)
			if top > h {
				return false
			}
		}
	}
	return true
}

func buildLeaf(heightGrid [][]int, region *osrs.Region, defs *osrs.FloorDefs, wx, wy, wz int) *maps.Cube {
	c := &maps.Cube{}

	h := getTerrainZ(heightGrid, wx, wy)

	if wz >= h {
		// Above terrain: empty
		c.EmptyFaces()
		return c
	}

	if wz+1 <= h-1 {
		// Fully below terrain surface: solid
		c.SolidFaces()
		texIdx := getTextureForTile(region, defs, wx, wy)
		for i := 0; i < 6; i++ {
			c.Texture[i] = texIdx
		}
		return c
	}

	// Surface cube: use edge data to shape the top face
	// Get heights at the four corners of this cube
	h00 := getTerrainZ(heightGrid, wx, wy)
	h10 := getTerrainZ(heightGrid, wx+1, wy)
	h01 := getTerrainZ(heightGrid, wx, wy+1)
	h11 := getTerrainZ(heightGrid, wx+1, wy+1)

	// Convert heights to edge values (0-8 scale within this cube)
	// Edge value 8 = top of cube, 0 = bottom of cube
	e00 := clampEdge(h00 - wz)
	e10 := clampEdge(h10 - wz)
	e01 := clampEdge(h01 - wz)
	e11 := clampEdge(h11 - wz)

	// Set edges for a heightfield cube
	// Dimension 0 (X): edges along X axis
	// Dimension 1 (Y): edges along Y axis
	// Dimension 2 (Z): edges along Z axis - this is what we modify for heightmaps
	//
	// cubeedge(c, d, x, y) = edges[d*4 + y*2 + x]
	// For Z dimension (d=2): edges[8], edges[9], edges[10], edges[11]
	// Each edge byte: low nibble = bottom, high nibble = top
	// For heightfield: bottom is always 0, top varies

	// X and Y dimensions stay solid (full extent)
	for i := 0; i < 8; i++ {
		c.Edges[i] = 0x80 // edgemake(0, 8) = solid
	}

	// Z dimension edges encode the height at each corner
	// cubeedge(c, 2, 0, 0) = edges[8]  → corner (0,0)
	// cubeedge(c, 2, 1, 0) = edges[9]  → corner (1,0)
	// cubeedge(c, 2, 0, 1) = edges[10] → corner (0,1)
	// cubeedge(c, 2, 1, 1) = edges[11] → corner (1,1)
	c.Edges[8] = edgeMake(0, e00)
	c.Edges[9] = edgeMake(0, e10)
	c.Edges[10] = edgeMake(0, e01)
	c.Edges[11] = edgeMake(0, e11)

	texIdx := getTextureForTile(region, defs, wx, wy)
	for i := 0; i < 6; i++ {
		c.Texture[i] = texIdx
	}

	return c
}

func edgeMake(a, b int) byte {
	return byte((b << 4) | a)
}

func clampEdge(v int) int {
	if v < 0 {
		return 0
	}
	if v > 8 {
		return 8
	}
	return v
}

func getTextureForTile(region *osrs.Region, defs *osrs.FloorDefs, wx, wy int) uint16 {
	// Convert world coords to tile coords
	tx := (wx - terrainOffset) / cubesPerTile
	ty := (wy - terrainOffset) / cubesPerTile
	if tx < 0 || tx >= 64 || ty < 0 || ty >= 64 {
		return 1 // default
	}

	// Check overlay first (higher priority)
	// Overlay IDs are 1-based in the region data.
	// In the cfg after texturereset: slots 0..149 are underlays, 150..323 are overlays.
	// So overlay ID n (1-based) → slot len(underlays) + (n-1)
	overlayID := int(region.Overlays[0][tx][ty])
	if overlayID > 0 && overlayID <= len(defs.Overlays) {
		return uint16(len(defs.Underlays) + overlayID - 1)
	}

	// Use underlay (1-based ID maps directly to slot index)
	underlayID := int(region.Underlays[0][tx][ty])
	if underlayID > 0 && underlayID <= len(defs.Underlays) {
		return uint16(underlayID)
	}

	return 1 // default
}

// addWalls converts OSRS wall objects to solid cube geometry in the octree.
func addWalls(root *maps.Cube, objects []osrs.PlacedObject, objDefs *osrs.ObjectDefs, heightGrid [][]int) {
	for _, obj := range objects {
		if obj.Type > objWallCorner {
			continue // only wall types 0, 1, 2
		}

		def := objDefs.Get(obj.ID)
		if !def.Solid {
			continue
		}

		// Convert OSRS tile position to world coordinates
		wx := obj.LocalX*cubesPerTile + terrainOffset
		wy := obj.LocalY*cubesPerTile + terrainOffset
		baseZ := getTerrainZ(heightGrid, wx, wy)

		switch obj.Type {
		case objWallStraight:
			// Place a thin wall along one edge of the tile
			var dx, dy int
			switch obj.Rotation {
			case 0: // West wall
				dx, dy = 0, 0
			case 1: // North wall
				dx, dy = 0, 0
			case 2: // East wall
				dx, dy = cubesPerTile - wallThickness, 0
			case 3: // South wall
				dx, dy = 0, cubesPerTile - wallThickness
			}

			for h := 0; h < wallHeight; h++ {
				for t := 0; t < cubesPerTile; t++ {
					var x, y int
					if obj.Rotation == 0 || obj.Rotation == 2 {
						// NS wall: extends along Y
						x = wx + dx
						y = wy + t
					} else {
						// EW wall: extends along X
						x = wx + t
						y = wy + dy
					}
					z := baseZ + h
					setCubeAt(root, x, y, z, worldSize, true, 1)
				}
			}

		case objWallCorner:
			// Place a pillar at the corner
			for h := 0; h < wallHeight; h++ {
				setCubeAt(root, wx, wy, baseZ+h, worldSize, true, 1)
			}
		}
	}
}

// setCubeAt sets a cube at the given world position to solid or empty.
func setCubeAt(root *maps.Cube, x, y, z, worldSize int, solid bool, texIdx uint16) {
	if x < 0 || y < 0 || z < 0 || x >= worldSize || y >= worldSize || z >= worldSize {
		return
	}

	// Navigate the octree to the leaf
	node := root
	size := worldSize
	for size > 1 {
		if len(node.Children) != maps.CUBE_FACTOR {
			// Need to subdivide
			old := *node
			node.Children = make([]*maps.Cube, maps.CUBE_FACTOR)
			for i := 0; i < maps.CUBE_FACTOR; i++ {
				child := &maps.Cube{}
				child.Edges = old.Edges
				child.Texture = old.Texture
				child.Material = old.Material
				node.Children[i] = child
			}
		}

		half := size / 2
		idx := 0
		if x >= half {
			idx |= 1
			x -= half
		}
		if y >= half {
			idx |= 2
			y -= half
		}
		if z >= half {
			idx |= 4
			z -= half
		}
		node = node.Children[idx]
		size = half
	}

	if solid {
		node.SolidFaces()
		for i := 0; i < 6; i++ {
			node.Texture[i] = texIdx
		}
		node.Children = nil
	} else {
		node.EmptyFaces()
		node.Children = nil
	}
}

// addWater sets MAT_WATER on cubes at sea level (Z <= terrainBaseZ) that are empty.
func addWater(root *maps.Cube, region *osrs.Region, heightGrid [][]int) {
	// Simple approach: any empty cube at or below the water line gets water material
	waterZ := terrainBaseZ
	setWaterRecursive(root, 0, 0, 0, worldSize, waterZ)
}

func setWaterRecursive(c *maps.Cube, ox, oy, oz, size, waterZ int) {
	if c == nil {
		return
	}

	if len(c.Children) == maps.CUBE_FACTOR {
		half := size / 2
		for i := 0; i < maps.CUBE_FACTOR; i++ {
			cx := ox + (i&1)*half
			cy := oy + ((i>>1)&1)*half
			cz := oz + ((i>>2)&1)*half
			setWaterRecursive(c.Children[i], cx, cy, cz, half, waterZ)
		}
		return
	}

	// Leaf cube
	if oz < waterZ && isEmptyCube(c) {
		c.Material = maps.MAT_WATER
	}
}

func isEmptyCube(c *maps.Cube) bool {
	for _, e := range c.Edges {
		if e != 0 {
			return false
		}
	}
	return true
}

// addDecorations converts OSRS game objects to Sauerbraten mapmodel entities.
// Returns the set of OSRS model IDs that need to be exported.
// modelKey identifies a unique (objectID, rotation) pair.
type modelKey struct {
	objID    int
	rotation int
}

func addDecorations(m *maps.GameMap, region *osrs.Region, objects []osrs.PlacedObject, objDefs *osrs.ObjectDefs, heightGrid [][]int) ([]modelKey, map[modelKey]int) {
	// Each unique (objectID, rotation) pair needs its own model export,
	// because OSRS applies rotation to the model vertices directly.
	keyToMapmodel := make(map[modelKey]int)
	var keyOrder []modelKey

	for _, obj := range objects {
		if obj.Type == objGroundDecor || obj.Type == objWallDecoration {
			continue
		}
		def := objDefs.Get(obj.ID)
		if len(def.ModelIDs) == 0 {
			continue
		}
		key := modelKey{obj.ID, obj.Rotation}
		if _, ok := keyToMapmodel[key]; !ok {
			keyToMapmodel[key] = len(keyOrder)
			keyOrder = append(keyOrder, key)
		}
	}

	// Second pass: place entities
	for _, obj := range objects {
		if obj.Type == objGroundDecor || obj.Type == objWallDecoration {
			continue
		}
		key := modelKey{obj.ID, obj.Rotation}
		mmIdx, ok := keyToMapmodel[key]
		if !ok {
			continue
		}

		def := objDefs.Get(obj.ID)
		sizeX := def.SizeX
		sizeY := def.SizeY
		if obj.Rotation == 1 || obj.Rotation == 3 {
			sizeX, sizeY = sizeY, sizeX
		}

		// Match OSRS: xPos = xLoc*128 + 64*sizeX, yPos = yLoc*128 + 64*sizeY
		// Convert to Sauer: multiply by cubesPerTile/128 and add terrainOffset
		osrsX := float64(obj.LocalX*128 + 64*sizeX)
		osrsY := float64(obj.LocalY*128 + 64*sizeY)
		wx := float32(osrsX*float64(cubesPerTile)/128.0) + float32(terrainOffset)
		wy := float32(osrsY*float64(cubesPerTile)/128.0) + float32(terrainOffset)

		// Height: use the object's PLANE (0-3) for correct vertical placement.
		// OSRS has 4 planes: 0=ground, 1=first floor, 2=second floor, 3=roof.
		// Each plane has its own height map.
		plane := obj.Plane
		if plane < 0 || plane > 3 {
			plane = 0
		}

		tx := obj.LocalX
		ty := obj.LocalY
		if tx < 0 || tx >= 64 || ty < 0 || ty >= 64 {
			continue
		}
		editX := tx + (sizeX >> 1)
		editY := ty + (sizeY >> 1)
		editX2 := tx + ((1 + sizeX) >> 1)
		editY2 := ty + ((1 + sizeY) >> 1)
		if editX > 64 { editX = 64 }
		if editY > 64 { editY = 64 }
		if editX2 > 64 { editX2 = 64 }
		if editY2 > 64 { editY2 = 64 }

		// Check for bridge tiles: if plane 1 has BRIDGE flag, reduce plane by 1
		actualPlane := plane
		if plane > 0 && tx < 104 && ty < 104 {
			if region.Flags[1][tx][ty]&2 == 2 {
				actualPlane--
			}
		}
		if actualPlane < 0 {
			actualPlane = 0
		}

		h00 := region.Heights[actualPlane][editX][editY]
		h10 := region.Heights[actualPlane][editX2][editY]
		h01 := region.Heights[actualPlane][editX][editY2]
		h11 := region.Heights[actualPlane][editX2][editY2]
		meanH := (h00 + h10 + h01 + h11) / 4
		wz := float32(terrainBaseZ + (-meanH)/heightScale)

		// Rotation is baked into the model vertices (matching OSRS),
		// so entity yaw is 0.
		m.Entities = append(m.Entities, maps.Entity{
			Position: maps.Vector{X: wx, Y: wy, Z: wz},
			Type:     C.EntityType(maps.ET_MAPMODEL),
			Attr1:    0,
			Attr2:    int16(mmIdx),
		})
	}

	return keyOrder, keyToMapmodel
}
