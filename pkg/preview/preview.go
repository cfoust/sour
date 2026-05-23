package preview

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog/log"
)

const (
	// Try depth 6 first, fall back to 5 if too many voxels.
	maxDepthCeiling  = 6
	maxDepthFloor    = 5
	maxVoxelCount    = 60000
)

// Generate creates a .svox preview from a map file.
// roots provides access to game data (textures, configs).
// mapData is the raw .ogz (gzipped) content.
// mapFile is the path to the .ogz within the roots (e.g. "packages/base/complex.ogz").
func Generate(ctx context.Context, roots []assets.Root, mapData []byte, mapFile string) ([]byte, error) {
	gameMap, err := maps.FromGZ(mapData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map: %w", err)
	}

	// Set up the min.Processor to resolve textures.
	processor := min.NewProcessor(roots, gameMap.VSlots)

	// Process default map settings.
	defaultPath := processor.SearchFile(ctx, "data/default_map_settings.cfg")
	if defaultPath != nil {
		if err := processor.ProcessFile(ctx, defaultPath); err != nil {
			log.Warn().Err(err).Msg("preview: failed to process default_map_settings.cfg")
		}
	}

	// Process the map's .cfg file.
	ext := filepath.Ext(mapFile)
	cfgPath := mapFile[:len(mapFile)-len(ext)] + ".cfg"
	for _, root := range roots {
		ref := min.NewReference(root, cfgPath)
		if ref.Exists(ctx) {
			if err := processor.ProcessFile(ctx, ref); err != nil {
				log.Warn().Err(err).Msgf("preview: failed to process map cfg %s", cfgPath)
			}
			break
		}
	}

	// Collect used VSlot indices from octree.
	usedVSlots := CollectUsedVSlots(gameMap.WorldRoot)
	log.Debug().Msgf("preview: %d unique VSlot indices used", len(usedVSlots))

	// Sample texture colors and build palette.
	palette, paletteMap := BuildPalette(ctx, processor, usedVSlots)
	log.Debug().Msgf("preview: palette has %d entries", len(palette))

	// Extract voxels with adaptive depth: try highest first, step down if too large.
	worldSize := int(gameMap.Header.WorldSize)
	maxDepth := maxDepthCeiling
	voxels := ExtractVoxels(gameMap.WorldRoot, worldSize, maxDepth, paletteMap)

	for len(voxels) > maxVoxelCount && maxDepth > maxDepthFloor {
		log.Debug().Msgf("preview: %d voxels at depth %d, reducing to depth %d", len(voxels), maxDepth, maxDepth-1)
		maxDepth--
		voxels = ExtractVoxels(gameMap.WorldRoot, worldSize, maxDepth, paletteMap)
	}

	log.Debug().Msgf("preview: %d voxels at depth %d", len(voxels), maxDepth)

	// Compute ambient occlusion.
	gridSize := 1 << maxDepth
	ComputeAO(voxels, gridSize)

	// Extract entities.
	entities := ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))

	// Extract skybox and lighting colors.
	skyTop, skyHorizon := ExtractSkyboxColors(ctx, processor, gameMap.Vars)
	ambient, sunlight := ExtractLightingColors(gameMap.Vars)

	preview := &MapPreview{
		MaxDepth:   uint8(maxDepth),
		GridSize:   uint16(gridSize),
		WorldSize:  uint32(worldSize),
		Palette:    palette,
		Voxels:     voxels,
		Entities:   entities,
		SkyTop:     skyTop,
		SkyHorizon: skyHorizon,
		Ambient:    ambient,
		Sunlight:   sunlight,
	}

	return Encode(preview)
}
