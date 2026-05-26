package preview

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"sync"

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

	// Collect liquid volumes (water + lava)
	waterPalIdx := uint8(len(palette))
	palette = append(palette, [3]uint8{30, 100, 200})
	lavaPalIdx := uint8(len(palette))
	palette = append(palette, [3]uint8{255, 80, 10})
	waterVoxels, lavaVoxels := CollectLiquidVoxels(gameMap.WorldRoot, worldSize, waterPalIdx, lavaPalIdx)
	voxels = append(voxels, waterVoxels...)
	voxels = append(voxels, lavaVoxels...)
	log.Debug().Msgf("preview: %d voxels (%d water, %d lava)", len(voxels), len(waterVoxels), len(lavaVoxels))

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
	AddBarriersFromOctree(fillGrid, gameMap.WorldRoot, worldSize)
	reachable, _ := ComputeReachableVolume(fillGrid, entityPositions, worldSize)
	log.Debug().Msgf("preview: %d reachable cells (fill grid %d³)", len(reachable), fillGrid.Size)

	// 3. Find optimal flat clip plane and camera angle
	clipY, cameraYaw, cameraPitch := findOptimalView(
		fillGrid, reachable, voxels, gridSize,
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
// Uses a visual grid (excluding clip/death material) for occlusion checks,
// and viewpoint entropy (Vázquez et al. 2001) for angle selection.
func findOptimalView(
	fillGrid *OccupancyGrid,
	reachable map[gridPos]bool,
	voxels []Voxel,
	gridSize int,
	focusX, focusY, focusZ, focusRadius float64,
) (clipZ int, yaw uint16, pitch uint16) {
	gridMax := fillGrid.Size
	clipZ = gridMax // default: no clip

	if len(reachable) == 0 {
		clipZCoord := gridSize
		yaw, pitch = findBestAngle(voxels, gridSize, clipZCoord, focusX, focusY, focusZ, focusRadius)
		return
	}

	// Build visual grid excluding invisible materials for occlusion check
	visualGrid := BuildVisualGrid(voxels, gridSize)

	// Count how many reachable cells are visually occluded from above
	occluded := 0
	total := len(reachable)
	for p := range reachable {
		for z := p.Z + 1; z < gridMax; z++ {
			if visualGrid.Get(p.X, p.Y, z) {
				occluded++
				break
			}
		}
	}

	occlusionPct := float64(occluded) / float64(total)
	log.Debug().Msgf("preview: %.0f%% of play area occluded from above", occlusionPct*100)

	if occlusionPct > 0.5 {
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

		for testZ := zMax + 3; testZ >= zMin; testZ-- {
			visible := 0
			for p := range reachable {
				if p.Z > testZ {
					continue
				}
				blocked := false
				for z := p.Z + 1; z <= testZ; z++ {
					if visualGrid.Get(p.X, p.Y, z) {
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

	clipZCoord := clipZ << fillGrid.Shift
	yaw, pitch = findBestAngle(voxels, gridSize, clipZCoord, focusX, focusY, focusZ, focusRadius)
	return
}

// findBestAngle uses a two-phase entropy search to find the best camera angle.
// Phase 1 (coarse): 15° yaw steps, 3 pitch values, low-res rays → fast screening.
// Phase 2 (fine): ±10° yaw, ±5° pitch around the best coarse result, higher-res rays.
// Pitch is constrained to 35–55° (the 3/4 isometric sweet spot) so different maps
// get distinct yaw angles rather than all converging on steep overhead views.
func findBestAngle(voxels []Voxel, gridSize, clipZ int, focusX, focusY, focusZ, radius float64) (uint16, uint16) {
	solidLk, _ := buildLookupFromSlice(voxels, clipZ)

	type candidate struct {
		yaw, pitch float64
		entropy    float64
	}

	evalCandidates := func(candidates []candidate, res int) {
		var wg sync.WaitGroup
		for i := range candidates {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				c := &candidates[idx]
				yawRad := c.yaw * math.Pi / 180
				pitchRad := c.pitch * math.Pi / 180
				camX := focusX + math.Cos(pitchRad)*math.Cos(yawRad)*radius
				camY := focusY + math.Cos(pitchRad)*math.Sin(yawRad)*radius
				camZ := focusZ + math.Sin(pitchRad)*radius
				c.entropy = viewpointEntropy(camX, camY, camZ, focusX, focusY, focusZ, solidLk, gridSize, res)
			}(i)
		}
		wg.Wait()
	}

	bestOf := func(candidates []candidate) candidate {
		best := candidates[0]
		for _, c := range candidates[1:] {
			if c.entropy > best.entropy {
				best = c
			}
		}
		return best
	}

	// Phase 1: coarse scan — 24 yaw × 3 pitch = 72 candidates, 24×24 rays
	var coarse []candidate
	for yawDeg := 0.0; yawDeg < 360; yawDeg += 15 {
		for _, pitchDeg := range []float64{35, 45, 55} {
			coarse = append(coarse, candidate{yaw: yawDeg, pitch: pitchDeg})
		}
	}
	evalCandidates(coarse, entropyCoarseRes)
	best := bestOf(coarse)

	// Phase 2: refine — ±10° yaw in 5° steps, ±5° pitch in 5° steps, 32×32 rays
	var fine []candidate
	for yawOff := -10.0; yawOff <= 10; yawOff += 5 {
		for pitchOff := -5.0; pitchOff <= 5; pitchOff += 5 {
			yaw := math.Mod(best.yaw+yawOff+360, 360)
			pitch := best.pitch + pitchOff
			if pitch < 30 || pitch > 55 {
				continue
			}
			fine = append(fine, candidate{yaw: yaw, pitch: pitch})
		}
	}
	evalCandidates(fine, entropyFineRes)
	best = bestOf(fine)

	log.Debug().Msgf("preview: best angle yaw=%.0f pitch=%.0f entropy=%.2f", best.yaw, best.pitch, best.entropy)
	return uint16(best.yaw * 10), uint16(best.pitch * 10)
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
	radius := float32(dists[idx]) * scale * 2.1 // 75% of previous 2.8x
	minRadius := float32(gs) / 4               // sensible minimum: 25% of grid
	if radius < minRadius {
		radius = minRadius
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
