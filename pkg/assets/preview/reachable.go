package preview

import "sort"

type gridPos struct {
	X, Y, Z int
}

// canStand checks if a player can be at (x, y, z): the cell must be empty
// and the cell directly below must be solid (floor).
func canStand(grid *OccupancyGrid, x, y, z int) bool {
	if x < 0 || x >= grid.Size || y < 0 || y >= grid.Size || z <= 0 || z >= grid.Size {
		return false
	}
	if grid.Get(x, y, z) {
		return false // cell is solid
	}
	return grid.Get(x, y, z-1) // solid directly below = floor
}

// ComputeReachableVolume does a 3D BFS from player start positions with
// simplified player physics:
//   - Walk horizontally on floors (solid cell directly below)
//   - Step up 1 cell (stairs/ramps)
//   - Fall off edges (walk horizontally, then drop to nearest floor)
//
// The vertical constraint (can only go UP 1 cell at a time from a floor)
// prevents the flood fill from escaping through ceiling gaps.
func ComputeReachableVolume(grid *OccupancyGrid, entityPositions []worldPos, worldSize int) (map[gridPos]bool, int) {
	size := grid.Size
	reachable := make(map[gridPos]bool)

	if len(entityPositions) == 0 {
		return reachable, size
	}

	scale := float32(size) / float32(worldSize)
	var starts []gridPos
	for _, e := range entityPositions {
		gx := clampInt(int(e.X*scale), 0, size-1)
		gy := clampInt(int(e.Y*scale), 0, size-1)
		gz := clampInt(int(e.Z*scale), 0, size-1)

		pos, ok := findStandingPos(grid, gx, gy, gz)
		if ok && !reachable[pos] {
			reachable[pos] = true
			starts = append(starts, pos)
		}
	}

	if len(starts) == 0 {
		return reachable, size
	}

	queue := make([]gridPos, len(starts))
	copy(queue, starts)

	hDirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, d := range hDirs {
			nx, ny := cur.X+d[0], cur.Y+d[1]

			// Walk at same level
			if canStand(grid, nx, ny, cur.Z) {
				p := gridPos{nx, ny, cur.Z}
				if !reachable[p] {
					reachable[p] = true
					queue = append(queue, p)
				}
			}

			// Step up 1 cell
			if canStand(grid, nx, ny, cur.Z+1) {
				p := gridPos{nx, ny, cur.Z + 1}
				if !reachable[p] {
					reachable[p] = true
					queue = append(queue, p)
				}
			}

			// Fall off edge: walk to adjacent column, drop to nearest floor
			if nx >= 0 && nx < size && ny >= 0 && ny < size && !grid.Get(nx, ny, cur.Z) {
				for nz := cur.Z - 1; nz >= 0; nz-- {
					if grid.Get(nx, ny, nz) {
						// Found floor, standing cell is above it
						landZ := nz + 1
						if landZ < size && !grid.Get(nx, ny, landZ) {
							p := gridPos{nx, ny, landZ}
							if !reachable[p] {
								reachable[p] = true
								queue = append(queue, p)
							}
						}
						break
					}
				}
			}
		}

		// Jump straight up 1 cell
		if canStand(grid, cur.X, cur.Y, cur.Z+1) {
			p := gridPos{cur.X, cur.Y, cur.Z + 1}
			if !reachable[p] {
				reachable[p] = true
				queue = append(queue, p)
			}
		}
	}

	clipZ := computeClipZ(reachable, size)
	return reachable, clipZ
}

func findStandingPos(grid *OccupancyGrid, gx, gy, gz int) (gridPos, bool) {
	for z := gz; z >= 0; z-- {
		if canStand(grid, gx, gy, z) {
			return gridPos{gx, gy, z}, true
		}
	}
	for r := 1; r <= 5; r++ {
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				nx, ny := gx+dx, gy+dy
				if nx < 0 || nx >= grid.Size || ny < 0 || ny >= grid.Size {
					continue
				}
				for z := gz + 2; z >= 0; z-- {
					if canStand(grid, nx, ny, z) {
						return gridPos{nx, ny, z}, true
					}
				}
			}
		}
	}
	return gridPos{}, false
}

func computeClipZ(reachable map[gridPos]bool, gridSize int) int {
	if len(reachable) == 0 {
		return gridSize
	}
	zValues := make([]int, 0, len(reachable))
	for p := range reachable {
		zValues = append(zValues, p.Z)
	}
	sort.Ints(zValues)
	idx := int(float64(len(zValues)) * 0.95)
	if idx >= len(zValues) {
		idx = len(zValues) - 1
	}
	clipZ := zValues[idx] + 3
	if clipZ > gridSize {
		clipZ = gridSize
	}
	return clipZ
}

func BuildCutawayHeightmap(reachable map[gridPos]bool, gridSize int, margin int) []uint8 {
	hmap := make([]uint8, gridSize*gridSize)
	hasAny := make([]bool, gridSize*gridSize)
	for p := range reachable {
		idx := p.Y*gridSize + p.X
		hasAny[idx] = true
		z := uint8(clampInt(p.Z+margin, 0, 255))
		if z > hmap[idx] {
			hmap[idx] = z
		}
	}
	globalMax := uint8(0)
	for _, h := range hmap {
		if h > globalMax {
			globalMax = h
		}
	}
	for i := range hmap {
		if !hasAny[i] {
			hmap[i] = globalMax
		}
	}
	return hmap
}

// BuildCutawayHeightmapDownsampled builds the heightmap at a lower resolution
// than the fill grid. Maps fill grid coords to output grid coords.
func BuildCutawayHeightmapDownsampled(reachable map[gridPos]bool, fillSize, outSize, margin int) []uint8 {
	hmap := make([]uint8, outSize*outSize)
	hasAny := make([]bool, outSize*outSize)
	ratio := fillSize / outSize

	for p := range reachable {
		ox := p.X / ratio
		oy := p.Y / ratio
		if ox >= outSize {
			ox = outSize - 1
		}
		if oy >= outSize {
			oy = outSize - 1
		}
		idx := oy*outSize + ox
		hasAny[idx] = true
		z := uint8(clampInt(p.Z/ratio+margin, 0, 255))
		if z > hmap[idx] {
			hmap[idx] = z
		}
	}

	globalMax := uint8(0)
	for _, h := range hmap {
		if h > globalMax {
			globalMax = h
		}
	}
	for i := range hmap {
		if !hasAny[i] {
			hmap[i] = globalMax
		}
	}

	return hmap
}

func ScoreCameraAngleWithReachable(grid *OccupancyGrid, reachable map[gridPos]bool, clipZ int, camX, camY, camZ, focusX, focusY, focusZ float64) int {
	dx := focusX - camX
	dy := focusY - camY
	dz := focusZ - camZ
	dist := sqrt(dx*dx + dy*dy + dz*dz)
	if dist < 1 {
		return 0
	}
	steps := int(dist) + 1
	if steps > 200 {
		steps = 200
	}
	score := 0
	gx := int(camX) >> grid.Shift
	gy := int(camY) >> grid.Shift
	gz := int(camZ) >> grid.Shift
	if grid.Get(gx, gy, gz) {
		score += 10000
	}
	if gz > clipZ {
		score += 5000
	}
	stepX := dx / float64(steps)
	stepY := dy / float64(steps)
	stepZ := dz / float64(steps)
	for i := 0; i < steps; i++ {
		px := camX + stepX*float64(i)
		py := camY + stepY*float64(i)
		pz := camZ + stepZ*float64(i)
		gx := int(px) >> grid.Shift
		gy := int(py) >> grid.Shift
		gz := int(pz) >> grid.Shift
		if gz > clipZ {
			continue
		}
		if grid.Get(gx, gy, gz) && !reachable[gridPos{gx, gy, gz}] {
			weight := steps - i
			score += weight
		}
	}
	return score
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	r := x
	for i := 0; i < 20; i++ {
		r = (r + x/r) / 2
	}
	return r
}
