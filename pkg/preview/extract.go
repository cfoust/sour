package preview

import (
	"github.com/cfoust/sour/pkg/maps"
)

// ExtractVoxels walks the octree and emits one Voxel per occupied grid cell
// at the target depth. paletteMap maps VSlot texture indices to palette indices.
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
		walkCube(child, ox, oy, oz, 1, maxDepth, paletteMap, &voxels)
	}
	return voxels
}

func walkCube(c *maps.Cube, ox, oy, oz, depth, maxDepth int, paletteMap map[uint16]uint8, voxels *[]Voxel) {
	if c == nil {
		return
	}

	span := 1 << (maxDepth - depth)

	// If the cube has children and we haven't reached maxDepth, recurse.
	if len(c.Children) > 0 && depth < maxDepth {
		half := span / 2
		for i, child := range c.Children {
			if child == nil {
				continue
			}
			cx := ox + (i&1)*half
			cy := oy + ((i>>1)&1)*half
			cz := oz + ((i>>2)&1)*half
			walkCube(child, cx, cy, cz, depth+1, maxDepth, paletteMap, voxels)
		}
		return
	}

	// Leaf node or we've reached maxDepth.
	if c.IsEmpty() {
		return
	}

	// Determine flags and dominant texture.
	var flags byte
	if c.IsEntirelySolid() {
		flags = FlagSolid
	} else {
		flags = FlagNormal
	}

	// Material flags
	mat := c.Material
	switch {
	case mat&maps.MATF_VOLUME == maps.MAT_WATER:
		flags |= FlagWater
	case mat&maps.MATF_VOLUME == maps.MAT_LAVA:
		flags |= FlagLava
	case mat&maps.MATF_VOLUME == maps.MAT_GLASS:
		flags |= FlagGlass
	case mat&maps.MATF_CLIP == maps.MAT_CLIP:
		flags |= FlagClip
	}

	// Pick dominant texture from the cube's 6 faces.
	// Use the most common non-sky texture.
	texCounts := make(map[uint16]int)
	for _, t := range c.Texture {
		if t != maps.DEFAULT_SKY {
			texCounts[t]++
		}
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
		palIdx = 0 // default gray
	}

	// Fill all grid cells this cube spans.
	for dz := 0; dz < span; dz++ {
		for dy := 0; dy < span; dy++ {
			for dx := 0; dx < span; dx++ {
				*voxels = append(*voxels, Voxel{
					X:            uint8(ox + dx),
					Y:            uint8(oy + dy),
					Z:            uint8(oz + dz),
					PaletteIndex: palIdx,
					Flags:        flags,
				})
			}
		}
	}
}
