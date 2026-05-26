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

// GenerateDebugReachable creates a .svox file that visualizes the reachable
// volume as green voxels overlaid on the map geometry.
func GenerateDebugReachable(ctx context.Context, roots []assets.Root, mapData []byte, mapFile string) ([]byte, error) {
	gameMap, err := maps.FromGZ(mapData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map: %w", err)
	}

	processor := min.NewProcessor(roots, gameMap.VSlots)
	defaultPath := processor.SearchFile(ctx, "data/default_map_settings.cfg")
	if defaultPath != nil {
		processor.ProcessFile(ctx, defaultPath)
	}
	ext := filepath.Ext(mapFile)
	cfgPath := mapFile[:len(mapFile)-len(ext)] + ".cfg"
	for _, root := range roots {
		ref := min.NewReference(root, cfgPath)
		if ref.Exists(ctx) {
			processor.ProcessFile(ctx, ref)
			break
		}
	}

	usedVSlots := CollectUsedVSlots(gameMap.WorldRoot)
	palette, paletteMap := BuildPalette(ctx, processor, usedVSlots)

	worldSize := int(gameMap.Header.WorldSize)
	entityPositions := extractDebugEntityPositions(gameMap.Entities)

	voxels := ExtractVoxelsAdaptive(gameMap.WorldRoot, worldSize, entityPositions, paletteMap)

	gridSize := 1 << coordDepth
	occGrid := BuildOccupancyGrid(voxels, gridSize)
	ComputeAO(voxels, occGrid)

	fillGrid := BuildFillGrid(voxels, gridSize)
	AddBarriersFromOctree(fillGrid, gameMap.WorldRoot, worldSize)

	reachable, clipZ := ComputeReachableVolume(fillGrid, entityPositions, worldSize)
	log.Info().Msgf("debug: %d reachable cells, clip Z=%d (fill grid %d³, shift=%d)", len(reachable), clipZ, fillGrid.Size, fillGrid.Shift)

	greenIdx := uint8(len(palette))
	palette = append(palette, [3]uint8{0, 255, 80})

	sizeLog2 := fillGrid.Shift
	for p := range reachable {
		if p.Z > 0 && !fillGrid.Get(p.X, p.Y, p.Z-1) {
			continue
		}
		voxels = append(voxels, Voxel{
			X:            uint16(p.X << fillGrid.Shift),
			Y:            uint16(p.Y << fillGrid.Shift),
			Z:            uint16(p.Z << fillGrid.Shift),
			PaletteIndex: greenIdx,
			Flags:        FlagNormal | byte(sizeLog2<<FlagSizeShift),
			AO:           255,
		})
	}

	entities := ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))
	skyTop, skyHorizon := ExtractSkyboxColors(ctx, processor, gameMap.Vars)
	ambient, sunlight := ExtractLightingColors(gameMap.Vars)
	focusX, focusY, focusZ, focusRadius := computeFocus(entityPositions, worldSize, gridSize)
	defaultClipY := uint16(clipZ << fillGrid.Shift)

	cameraYaw, cameraPitch := FindBestCameraAngleReachable(
		fillGrid, reachable, clipZ,
		float64(focusX), float64(focusY), float64(focusZ),
		float64(focusRadius),
	)

	p := &MapPreview{
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
		CameraYaw:   cameraYaw,
		CameraPitch: cameraPitch,
		ClipY:       defaultClipY,
	}

	return Encode(p)
}

func extractDebugEntityPositions(entities []maps.Entity) []worldPos {
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
