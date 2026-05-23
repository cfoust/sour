package preview

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"

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

	// Compute focus point and radius from entity positions.
	focusX, focusY, focusZ, focusRadius := computeFocus(entityPositions, worldSize, gridSize)

	preview := &MapPreview{
		MaxDepth:    coordDepth,
		GridSize:    uint16(gridSize),
		WorldSize:   uint32(worldSize),
		Palette:     palette,
		Voxels:      voxels,
		Entities:    entities,
		SkyTop:      skyTop,
		SkyHorizon:  skyHorizon,
		Ambient:     ambient,
		Sunlight:    sunlight,
		FocusX:      focusX,
		FocusY:      focusY,
		FocusZ:      focusZ,
		FocusRadius: focusRadius,
	}

	return Encode(preview)
}

// computeFocus calculates the orbit target and suggested camera distance
// from entity positions. Uses the centroid as focus and 90th percentile
// distance as radius to exclude outliers.
func computeFocus(positions []worldPos, worldSize, gridSize int) (uint16, uint16, uint16, uint16) {
	gs := uint16(gridSize)
	half := gs / 2

	if len(positions) == 0 {
		return half, half, half, gs
	}

	scale := float32(gridSize) / float32(worldSize)

	// Centroid
	var sx, sy, sz float32
	for _, p := range positions {
		sx += p.X
		sy += p.Y
		sz += p.Z
	}
	n := float32(len(positions))
	cx, cy, cz := sx/n, sy/n, sz/n

	// Distances from centroid
	dists := make([]float64, len(positions))
	for i, p := range positions {
		dx := float64(p.X - cx)
		dy := float64(p.Y - cy)
		dz := float64(p.Z - cz)
		dists[i] = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	sort.Float64s(dists)

	// 90th percentile distance
	idx := int(float64(len(dists)) * 0.9)
	if idx >= len(dists) {
		idx = len(dists) - 1
	}
	radius := float32(dists[idx]) * scale * 1.5 // padding for a nice view
	if radius < 32 {
		radius = 32
	}

	fx := uint16(clampInt(int(cx*scale), 0, int(gs)-1))
	fy := uint16(clampInt(int(cy*scale), 0, int(gs)-1))
	fz := uint16(clampInt(int(cz*scale), 0, int(gs)-1))
	fr := uint16(clampInt(int(radius), 1, int(gs)*2))

	return fx, fy, fz, fr
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
