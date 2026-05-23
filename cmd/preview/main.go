package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"
	"github.com/cfoust/sour/pkg/preview"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	if len(os.Args) >= 2 && os.Args[1] == "bench" {
		benchDepths()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "stats" {
		octreeStats()
		return
	}

	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: preview <outdir> <root1> [root2...] -- <map1.ogz> [map2.ogz...]\n")
		fmt.Fprintf(os.Stderr, "       preview bench <root1> [root2...] -- <map1.ogz> [map2.ogz...]\n")
		os.Exit(1)
	}

	outdir := os.Args[1]
	args := os.Args[2:]

	dashIdx := findDash(args)
	if dashIdx < 0 {
		fmt.Fprintf(os.Stderr, "Error: separate roots and maps with --\n")
		os.Exit(1)
	}

	roots := makeRoots(args[:dashIdx])
	mapFiles := args[dashIdx+1:]
	os.MkdirAll(outdir, 0755)
	ctx := context.Background()

	for _, mapFile := range mapFiles {
		mapData := readFromRoots(ctx, roots, mapFile)
		if mapData == nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))
		fmt.Fprintf(os.Stderr, "Generating preview for %s...\n", name)

		data, err := preview.Generate(ctx, roots, mapData, mapFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		outPath := filepath.Join(outdir, name+".svox")
		os.WriteFile(outPath, data, 0644)
		fmt.Fprintf(os.Stderr, "Wrote %s (%d bytes)\n", outPath, len(data))
	}
}

func benchDepths() {
	args := os.Args[2:]
	dashIdx := findDash(args)
	if dashIdx < 0 {
		fmt.Fprintf(os.Stderr, "Error: separate roots and maps with --\n")
		os.Exit(1)
	}

	roots := makeRoots(args[:dashIdx])
	mapFiles := args[dashIdx+1:]
	ctx := context.Background()

	fmt.Printf("%-12s", "Map")
	for d := 4; d <= 8; d++ {
		fmt.Printf("  Depth %d (voxels / KB)", d)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 120))

	for _, mapFile := range mapFiles {
		mapData := readFromRoots(ctx, roots, mapFile)
		if mapData == nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))

		gameMap, err := maps.FromGZ(mapData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", name, err)
			continue
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

		usedVSlots := preview.CollectUsedVSlots(gameMap.WorldRoot)
		palette, paletteMap := preview.BuildPalette(ctx, processor, usedVSlots)

		fmt.Printf("%-12s", name)
		for d := 4; d <= 8; d++ {
			worldSize := int(gameMap.Header.WorldSize)
			voxels := preview.ExtractVoxels(gameMap.WorldRoot, worldSize, d, paletteMap)
			gridSize := 1 << d
			preview.ComputeAO(voxels, gridSize)
			entities := preview.ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))

			// Estimate size: header(32) + palette(n*3) + voxels(n*6) + entities(n*4)
			size := 32 + len(palette)*3 + len(voxels)*6 + len(entities)*4
			fmt.Printf("  %s / %s", padLeft(strconv.Itoa(len(voxels)), 8), padLeft(fmt.Sprintf("%.1f", float64(size)/1024), 6))
		}
		fmt.Println()
	}
}

func findDash(args []string) int {
	for i, arg := range args {
		if arg == "--" {
			return i
		}
	}
	return -1
}

func makeRoots(paths []string) []assets.Root {
	roots := make([]assets.Root, len(paths))
	for i, p := range paths {
		abs, _ := filepath.Abs(p)
		r := assets.FSRoot(abs)
		roots[i] = r
	}
	return roots
}

func readFromRoots(ctx context.Context, roots []assets.Root, mapFile string) []byte {
	for _, root := range roots {
		data, err := root.ReadFile(ctx, mapFile)
		if err == nil {
			return data
		}
	}
	fmt.Fprintf(os.Stderr, "Error: could not find %s in any root\n", mapFile)
	return nil
}

func padLeft(s string, width int) string {
	for len(s) < width {
		s = " " + s
	}
	return s
}

func octreeStats() {
	args := os.Args[2:]
	dashIdx := findDash(args)
	if dashIdx < 0 {
		fmt.Fprintf(os.Stderr, "Error: separate roots and maps with --\n")
		os.Exit(1)
	}
	roots := makeRoots(args[:dashIdx])
	mapFiles := args[dashIdx+1:]
	ctx := context.Background()

	for _, mapFile := range mapFiles {
		mapData := readFromRoots(ctx, roots, mapFile)
		if mapData == nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))
		gameMap, err := maps.FromGZ(mapData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}
		stats := preview.CollectOctreeStats(gameMap.WorldRoot, int(gameMap.Header.WorldSize))
		fmt.Printf("\n%s (worldSize=%d, maxOctreeDepth=%d):\n", name, gameMap.Header.WorldSize, stats.MaxDepth)
		fmt.Printf("  %-8s %10s %10s %10s\n", "Depth", "CubeSize", "Solid", "Empty")
		for d := 1; d <= stats.MaxDepth; d++ {
			cubeSize := int(gameMap.Header.WorldSize) >> d
			fmt.Printf("  %-8d %10d %10d %10d\n", d, cubeSize, stats.LeafCounts[d], stats.EmptyCounts[d])
		}
	}
}
