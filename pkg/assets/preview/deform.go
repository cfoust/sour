package preview

import (
	"github.com/cfoust/sour/pkg/maps"
)

// Edge byte layout (from Sauerbraten):
//
//	byte = (end << 4) | start
//	start = byte & 0xF   (low coordinate, 0-8)
//	end   = byte >> 4    (high coordinate, 0-8)
//
// 12 edges indexed as edges[dim*4 + y*2 + x]:
//
//	X-axis (0-3), x=Y/8, y=Z/8:
//	  0: (Y=0, Z=0)  1: (Y=8, Z=0)  2: (Y=0, Z=8)  3: (Y=8, Z=8)
//	Y-axis (4-7), x=Z/8, y=X/8:
//	  4: (X=0, Z=0)  5: (X=0, Z=8)  6: (X=8, Z=0)  7: (X=8, Z=8)
//	Z-axis (8-11), x=X/8, y=Y/8:
//	  8: (X=0, Y=0)  9: (X=8, Y=0)  10: (X=0, Y=8)  11: (X=8, Y=8)

func edgeStart(e byte) float64 { return float64(e & 0xF) }
func edgeEnd(e byte) float64   { return float64(e >> 4) }

func bilinear(v00, v10, v01, v11, t0, t1 float64) float64 {
	return v00*(1-t0)*(1-t1) + v10*t0*(1-t1) + v01*(1-t0)*t1 + v11*t0*t1
}

// isCubeActuallyDeformed checks if any edge is not at full extent (0→8).
func isCubeActuallyDeformed(c *maps.Cube) bool {
	for _, e := range c.Edges {
		start := e & 0xF
		end := e >> 4
		if start != 0 || end != 8 {
			return true
		}
	}
	return false
}

// doesSubCellIntersect checks whether the deformed solid volume has any
// overlap with a sub-cell defined by [xLo,xHi] × [yLo,yHi] × [zLo,zHi]
// in cube-local coordinates [0,8].
//
// For each axis, we compute the min/max of the deformed extent at the
// corners of the other two axes' ranges. If the deformed extent overlaps
// the sub-cell range in ALL three axes, the sub-cell is occupied.
// This is conservative: it never misses thin geometry.
func doesSubCellIntersect(c *maps.Cube, xLo, xHi, yLo, yHi, zLo, zHi float64) bool {
	yTs := [2]float64{yLo / 8, yHi / 8}
	zTs := [2]float64{zLo / 8, zHi / 8}
	xTs := [2]float64{xLo / 8, xHi / 8}

	// X-axis: deformed X range sampled at corners of (Y, Z) sub-cell
	minStart, maxEnd := 8.0, 0.0
	for _, ty := range yTs {
		for _, tz := range zTs {
			s := bilinear(edgeStart(c.Edges[0]), edgeStart(c.Edges[1]),
				edgeStart(c.Edges[2]), edgeStart(c.Edges[3]), ty, tz)
			e := bilinear(edgeEnd(c.Edges[0]), edgeEnd(c.Edges[1]),
				edgeEnd(c.Edges[2]), edgeEnd(c.Edges[3]), ty, tz)
			if s < minStart {
				minStart = s
			}
			if e > maxEnd {
				maxEnd = e
			}
		}
	}
	if maxEnd <= xLo || minStart >= xHi {
		return false
	}

	// Y-axis
	minStart, maxEnd = 8.0, 0.0
	for _, tx := range xTs {
		for _, tz := range zTs {
			s := bilinear(edgeStart(c.Edges[4]), edgeStart(c.Edges[6]),
				edgeStart(c.Edges[5]), edgeStart(c.Edges[7]), tx, tz)
			e := bilinear(edgeEnd(c.Edges[4]), edgeEnd(c.Edges[6]),
				edgeEnd(c.Edges[5]), edgeEnd(c.Edges[7]), tx, tz)
			if s < minStart {
				minStart = s
			}
			if e > maxEnd {
				maxEnd = e
			}
		}
	}
	if maxEnd <= yLo || minStart >= yHi {
		return false
	}

	// Z-axis
	minStart, maxEnd = 8.0, 0.0
	for _, tx := range xTs {
		for _, ty := range yTs {
			s := bilinear(edgeStart(c.Edges[8]), edgeStart(c.Edges[9]),
				edgeStart(c.Edges[10]), edgeStart(c.Edges[11]), tx, ty)
			e := bilinear(edgeEnd(c.Edges[8]), edgeEnd(c.Edges[9]),
				edgeEnd(c.Edges[10]), edgeEnd(c.Edges[11]), tx, ty)
			if s < minStart {
				minStart = s
			}
			if e > maxEnd {
				maxEnd = e
			}
		}
	}
	if maxEnd <= zLo || minStart >= zHi {
		return false
	}

	return true
}

// emitDeformedCube subdivides a deformed cube into sub-cells and emits those
// that intersect the deformed geometry volume. Subdivision is capped at 1
// level (2×2×2 = 8 sub-cells). The post-extraction simplifier (SimplifyVoxels)
// handles further LOD reduction using error-driven octree collapse.
func emitDeformedCube(c *maps.Cube, ox, oy, oz, depth, targetDepth, maxCoordDepth int, paletteMap map[uint16]uint8, grid *OccupancyGrid, voxels *[]Voxel) {
	span := 1 << (maxCoordDepth - depth)
	if span < 1 {
		span = 1
	}

	subdiv := targetDepth - depth
	if subdiv > 3 {
		subdiv = 3 // 8×8×8 matches Sauerbraten's edge resolution
	}
	if subdiv <= 0 {
		flags := classifyCube(c)
		palIdx := dominantPaletteIndex(c, paletteMap)
		sizeLog2 := maxCoordDepth - depth
		if sizeLog2 < 0 {
			sizeLog2 = 0
		}
		flags |= byte(sizeLog2 << FlagSizeShift)
		*voxels = append(*voxels, Voxel{
			X: uint16(ox), Y: uint16(oy), Z: uint16(oz),
			PaletteIndex: palIdx, Flags: flags,
		})
		return
	}

	steps := 1 << subdiv
	cellSize := span / steps
	cubeStep := 8.0 / float64(steps)

	flags := classifyCube(c)
	palIdx := dominantPaletteIndex(c, paletteMap)

	sizeLog2 := 0
	for s := cellSize; s > 1; s >>= 1 {
		sizeLog2++
	}

	for dz := 0; dz < steps; dz++ {
		for dy := 0; dy < steps; dy++ {
			for dx := 0; dx < steps; dx++ {
				xLo := float64(dx) * cubeStep
				xHi := float64(dx+1) * cubeStep
				yLo := float64(dy) * cubeStep
				yHi := float64(dy+1) * cubeStep
				zLo := float64(dz) * cubeStep
				zHi := float64(dz+1) * cubeStep

				if !doesSubCellIntersect(c, xLo, xHi, yLo, yHi, zLo, zHi) {
					continue
				}

				f := flags | byte(sizeLog2<<FlagSizeShift)
				*voxels = append(*voxels, Voxel{
					X:            uint16(ox + dx*cellSize),
					Y:            uint16(oy + dy*cellSize),
					Z:            uint16(oz + dz*cellSize),
					PaletteIndex: palIdx,
					Flags:        f,
				})
			}
		}
	}
}
