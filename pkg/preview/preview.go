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

func Generate(ctx context.Context, roots []assets.Root, mapData []byte, mapFile string) ([]byte, error) {
	gameMap, err := maps.FromGZ(mapData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse map: %w", err)
	}

	processor := min.NewProcessor(roots, gameMap.VSlots)
	defaultPath := processor.SearchFile(ctx, "data/default_map_settings.cfg")
	if defaultPath != nil {
		if err := processor.ProcessFile(ctx, defaultPath); err != nil {
			log.Warn().Err(err).Msg("preview: failed to process default_map_settings.cfg")
		}
	}
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

	usedVSlots := CollectUsedVSlots(gameMap.WorldRoot)
	palette, paletteMap := BuildPalette(ctx, processor, usedVSlots)

	worldSize := int(gameMap.Header.WorldSize)
	entityPositions := extractEntityPositions(gameMap.Entities)

	voxels := ExtractVoxelsAdaptive(gameMap.WorldRoot, worldSize, entityPositions, paletteMap)
	log.Debug().Msgf("preview: %d voxels", len(voxels))

	gridSize := 1 << coordDepth
	occGrid := BuildOccupancyGrid(voxels, gridSize)
	ComputeAO(voxels, occGrid)

	entities := ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))
	skyTop, skyHorizon := ExtractSkyboxColors(ctx, processor, gameMap.Vars)
	ambient, sunlight := ExtractLightingColors(gameMap.Vars)

	// 1. Focus on 90th percentile entity centroid
	focusX, focusY, focusZ, focusRadius := computeFocus(entityPositions, worldSize, gridSize)

	// 2. Flood fill for reachable volume
	fillGrid := BuildFillGrid(voxels, gridSize)
	reachable, _ := ComputeReachableVolume(fillGrid, entityPositions, worldSize)
	log.Debug().Msgf("preview: %d reachable cells (fill grid %d³)", len(reachable), fillGrid.Size)

	// 3. Find optimal flat clip plane and camera angle
	clipY, cameraYaw, cameraPitch := findOptimalView(
		fillGrid, reachable, entityPositions, worldSize,
		float64(focusX), float64(focusY), float64(focusZ), float64(focusRadius),
	)
	// Convert clipY from fill grid Z to coord grid Z
	clipYCoord := uint16(clipY << fillGrid.Shift)

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
		CameraYaw:   cameraYaw,
		CameraPitch: cameraPitch,
		ClipY:       clipYCoord,
	}

	return Encode(preview)
}

// findOptimalView computes the best clip plane and camera angle.
// If >50% of the play area is occluded from above, it searches for the
// clip height that maximizes visible reachable cells. Otherwise no clip.
func findOptimalView(
	grid *OccupancyGrid,
	reachable map[gridPos]bool,
	entityPositions []worldPos,
	worldSize int,
	focusX, focusY, focusZ, focusRadius float64,
) (clipZ int, yaw uint16, pitch uint16) {
	gridMax := grid.Size
	clipZ = gridMax // default: no clip

	if len(reachable) == 0 {
		yaw, pitch = findBestAngle(grid, reachable, gridMax, focusX, focusY, focusZ, focusRadius)
		return
	}

	// Count how many reachable cells are occluded from directly above
	occluded := 0
	total := len(reachable)
	for p := range reachable {
		for z := p.Z + 1; z < gridMax; z++ {
			if grid.Get(p.X, p.Y, z) {
				occluded++
				break
			}
		}
	}

	occlusionPct := float64(occluded) / float64(total)
	log.Debug().Msgf("preview: %.0f%% of play area occluded from above", occlusionPct*100)

	if occlusionPct > 0.5 {
		// Search for the clip height that maximizes visible play area.
		// Scan Z levels in discrete steps through the reachable range.
		zMin, zMax := gridMax, 0
		for p := range reachable {
			if p.Z < zMin {
				zMin = p.Z
			}
			if p.Z > zMax {
				zMax = p.Z
			}
		}

		bestVisible := 0
		bestClip := gridMax

		// Try each Z level from just above the play area down to mid-play area
		for testZ := zMax + 3; testZ >= zMin; testZ-- {
			visible := 0
			for p := range reachable {
				if p.Z > testZ {
					continue // this reachable cell is above the clip
				}
				blocked := false
				for z := p.Z + 1; z <= testZ; z++ {
					if grid.Get(p.X, p.Y, z) {
						blocked = true
						break
					}
				}
				if !blocked {
					visible++
				}
			}
			if visible > bestVisible {
				bestVisible = visible
				bestClip = testZ
			}
		}

		clipZ = bestClip
		log.Debug().Msgf("preview: optimal clip Z=%d reveals %d/%d reachable cells", clipZ, bestVisible, total)
	}

	yaw, pitch = findBestAngle(grid, reachable, clipZ, focusX, focusY, focusZ, focusRadius)
	return
}

func findBestAngle(grid *OccupancyGrid, reachable map[gridPos]bool, clipZ int, focusX, focusY, focusZ, radius float64) (uint16, uint16) {
	bestYaw, bestPitch := 0.0, 30.0
	bestScore := math.MaxInt32

	for yawDeg := 0.0; yawDeg < 360; yawDeg += 5 {
		for _, pitchDeg := range []float64{20, 30, 40, 50} {
			yawRad := yawDeg * math.Pi / 180
			pitchRad := pitchDeg * math.Pi / 180

			camX := focusX + math.Cos(pitchRad)*math.Cos(yawRad)*radius
			camY := focusY + math.Cos(pitchRad)*math.Sin(yawRad)*radius
			camZ := focusZ + math.Sin(pitchRad)*radius

			score := ScoreCameraAngleWithReachable(grid, reachable, clipZ, camX, camY, camZ, focusX, focusY, focusZ)
			if score < bestScore {
				bestScore = score
				bestYaw = yawDeg
				bestPitch = pitchDeg
			}
		}
	}

	return uint16(bestYaw * 10), uint16(bestPitch * 10)
}

func computeFocus(positions []worldPos, worldSize, gridSize int) (uint16, uint16, uint16, uint16) {
	gs := uint16(gridSize)
	half := gs / 2
	if len(positions) == 0 {
		return half, half, half, gs
	}

	scale := float32(gridSize) / float32(worldSize)

	var sx, sy, sz float32
	for _, p := range positions {
		sx += p.X
		sy += p.Y
		sz += p.Z
	}
	n := float32(len(positions))
	cx, cy, cz := sx/n, sy/n, sz/n

	dists := make([]float64, len(positions))
	for i, p := range positions {
		dx := float64(p.X - cx)
		dy := float64(p.Y - cy)
		dz := float64(p.Z - cz)
		dists[i] = math.Sqrt(dx*dx + dy*dy + dz*dz)
	}
	sort.Float64s(dists)

	// 90th percentile
	idx := int(float64(len(dists)) * 0.9)
	if idx >= len(dists) {
		idx = len(dists) - 1
	}
	radius := float32(dists[idx]) * scale * 1.4 // 40% margin
	if radius < 32 {
		radius = 32
	}

	fx := uint16(clampInt(int(cx*scale), 0, int(gs)-1))
	fy := uint16(clampInt(int(cy*scale), 0, int(gs)-1))
	fz := uint16(clampInt(int(cz*scale), 0, int(gs)-1))
	fr := uint16(clampInt(int(radius), 1, int(gs)*2))

	return fx, fy, fz, fr
}

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
