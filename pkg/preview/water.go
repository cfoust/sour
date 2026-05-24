package preview

import "github.com/cfoust/sour/pkg/maps"

// CollectLiquidVoxels walks the octree and emits voxels for water and lava
// material cubes. These are geometrically empty but have material flags,
// so they're skipped by the normal voxel extraction.
func CollectLiquidVoxels(root *maps.Cube, worldSize int, waterPalIdx, lavaPalIdx uint8) (water []Voxel, lava []Voxel) {
	if root == nil || len(root.Children) == 0 {
		return
	}
	gridSize := 1 << coordDepth
	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkLiquids(child, ox, oy, oz, 1, waterPalIdx, lavaPalIdx, &water, &lava)
	}
	return
}

func walkLiquids(c *maps.Cube, ox, oy, oz, depth int, waterPalIdx, lavaPalIdx uint8, water, lava *[]Voxel) {
	if c == nil {
		return
	}

	span := 1 << (coordDepth - depth)

	if len(c.Children) > 0 && depth < coordDepth {
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			nx := ox + (i&1)*half
			ny := oy + ((i>>1)&1)*half
			nz := oz + ((i>>2)&1)*half
			walkLiquids(child, nx, ny, nz, depth+1, waterPalIdx, lavaPalIdx, water, lava)
		}
		return
	}

	sizeLog2 := coordDepth - depth
	if sizeLog2 < 0 {
		sizeLog2 = 0
	}

	vol := c.Material & maps.MATF_VOLUME
	switch vol {
	case maps.MAT_WATER:
		*water = append(*water, Voxel{
			X: uint16(ox), Y: uint16(oy), Z: uint16(oz),
			PaletteIndex: waterPalIdx,
			Flags:        FlagWater | byte(sizeLog2<<FlagSizeShift),
			AO:           255,
		})
	case maps.MAT_LAVA:
		*lava = append(*lava, Voxel{
			X: uint16(ox), Y: uint16(oy), Z: uint16(oz),
			PaletteIndex: lavaPalIdx,
			Flags:        FlagLava | byte(sizeLog2<<FlagSizeShift),
			AO:           255,
		})
	}
}
