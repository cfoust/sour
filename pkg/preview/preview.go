package preview

import (
	"context"
	"fmt"
	"path/filepath"

	C "github.com/cfoust/sour/pkg/game/constants"
	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog/log"
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

	// Sample texture colors and build palette.
	palette, paletteMap := BuildPalette(ctx, processor, usedVSlots)

	worldSize := int(gameMap.Header.WorldSize)

	// Extract gameplay entity world positions for adaptive depth.
	entityPositions := extractEntityPositions(gameMap.Entities)
	log.Debug().Msgf("preview: %d entity positions for adaptive detail", len(entityPositions))

	// Extract voxels with adaptive depth around entities.
	voxels := ExtractVoxelsAdaptive(gameMap.WorldRoot, worldSize, entityPositions, paletteMap)
	log.Debug().Msgf("preview: %d voxels (adaptive, %d³ coord grid)", len(voxels), 1<<coordDepth)

	// Compute ambient occlusion.
	gridSize := 1 << coordDepth
	ComputeAO(voxels, gridSize)

	// Extract entities (quantized to grid).
	entities := ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))

	// Extract skybox and lighting colors.
	skyTop, skyHorizon := ExtractSkyboxColors(ctx, processor, gameMap.Vars)
	ambient, sunlight := ExtractLightingColors(gameMap.Vars)

	preview := &MapPreview{
		MaxDepth:   coordDepth,
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

// extractEntityPositions returns world-space positions of gameplay-relevant
// entities (the same types used for preview entities).
func extractEntityPositions(entities []maps.Entity) []worldPos {
	var positions []worldPos
	for _, e := range entities {
		switch e.Type {
		case C.EntityTypePlayerStart,
			C.EntityTypeFlag,
			C.EntityTypeBase,
			C.EntityTypeTeleport,
			C.EntityTypeJumpPad,
			C.EntityTypeHealth,
			C.EntityTypeBoost,
			C.EntityTypeGreenArmour,
			C.EntityTypeYellowArmour,
			C.EntityTypeShells,
			C.EntityTypeBullets,
			C.EntityTypeRockets,
			C.EntityTypeRounds,
			C.EntityTypeGrenades,
			C.EntityTypeCartridges,
			C.EntityTypeQuad:
			positions = append(positions, worldPos{
				X: e.Position.X,
				Y: e.Position.Y,
				Z: e.Position.Z,
			})
		}
	}
	return positions
}
