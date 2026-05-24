package preview

import "github.com/cfoust/sour/pkg/maps"

// CollectWaterVoxels walks the octree and emits voxels for water material
// cubes. Water cubes are geometrically empty but have the water material flag,
// so they're skipped by the normal voxel extraction.
func CollectWaterVoxels(root *maps.Cube, worldSize int, waterPaletteIdx uint8) []Voxel {
	if root == nil || len(root.Children) == 0 {
		return nil
	}
	gridSize := 1 << coordDepth
	var voxels []Voxel
	for i, child := range root.Children {
		if child == nil {
			continue
		}
		ox := (i & 1) * (gridSize / 2)
		oy := ((i >> 1) & 1) * (gridSize / 2)
		oz := ((i >> 2) & 1) * (gridSize / 2)
		walkWater(child, ox, oy, oz, 1, waterPaletteIdx, &voxels)
	}
	return voxels
}

func walkWater(c *maps.Cube, ox, oy, oz, depth int, waterPalIdx uint8, voxels *[]Voxel) {
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
			walkWater(child, nx, ny, nz, depth+1, waterPalIdx, voxels)
		}
		return
	}

	// Check if this cube has water material
	if c.Material&maps.MATF_VOLUME != maps.MAT_WATER {
		return
	}

	sizeLog2 := coordDepth - depth
	if sizeLog2 < 0 {
		sizeLog2 = 0
	}

	*voxels = append(*voxels, Voxel{
		X:            uint16(ox),
		Y:            uint16(oy),
		Z:            uint16(oz),
		PaletteIndex: waterPalIdx,
		Flags:        FlagWater | byte(sizeLog2<<FlagSizeShift),
		AO:           255,
	})
}
