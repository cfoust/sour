package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
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
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if len(os.Args) >= 2 && os.Args[1] == "bench" {
		benchDepths()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "stats" {
		octreeStats()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "debug" {
		debugReachable()
		return
	}

	if len(os.Args) >= 2 && os.Args[1] == "render" {
		renderPreviews()
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
			occGrid := preview.BuildOccupancyGrid(voxels, gridSize)
			preview.ComputeAO(voxels, occGrid)
			entities := preview.ExtractEntities(gameMap.Entities, gameMap.Header.WorldSize, uint16(gridSize))

			// Estimate size: header(32) + palette(n*3) + voxels(n*6) + entities(n*4)
			size := 32 + len(palette)*3 + len(voxels)*6 + len(entities)*4
			fmt.Printf("  %s / %s", padLeft(strconv.Itoa(len(voxels)), 8), padLeft(fmt.Sprintf("%.1f", float64(size)/1024), 6))
		}
		fmt.Println()
	}
}

func debugReachable() {
	args := os.Args[2:]
	dashIdx := findDash(args)
	if dashIdx < 0 {
		fmt.Fprintf(os.Stderr, "Error: separate roots and maps with --\n")
		os.Exit(1)
	}
	roots := makeRoots(args[:dashIdx])
	mapFiles := args[dashIdx+1:]
	outdir := "."
	if dashIdx > 0 {
		outdir = args[0]
		roots = makeRoots(args[1:dashIdx])
	}
	os.MkdirAll(outdir, 0755)
	ctx := context.Background()

	for _, mapFile := range mapFiles {
		mapData := readFromRoots(ctx, roots, mapFile)
		if mapData == nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))
		fmt.Fprintf(os.Stderr, "Generating debug reachable for %s...\n", name)

		data, err := preview.GenerateDebugReachable(ctx, roots, mapData, mapFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		outPath := filepath.Join(outdir, name+"_debug.svox")
		os.WriteFile(outPath, data, 0644)
		fmt.Fprintf(os.Stderr, "Wrote %s (%d bytes)\n", outPath, len(data))
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

func renderPreviews() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: preview render <outdir> <file1.svox> [file2.svox...]\n")
		os.Exit(1)
	}
	outdir := os.Args[2]
	files := os.Args[3:]
	os.MkdirAll(outdir, 0755)

	size := 2048

	// Try GPU renderer first
	glRenderer := preview.NewGLRenderer(size, size)
	if glRenderer != nil {
		fmt.Fprintf(os.Stderr, "Using GPU renderer\n")
		defer glRenderer.Close()
	} else {
		fmt.Fprintf(os.Stderr, "GPU unavailable, using software renderer\n")
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", file, err)
			continue
		}
		p, err := preview.Decode(data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding %s: %v\n", file, err)
			continue
		}

		name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		fmt.Fprintf(os.Stderr, "Rendering %s...\n", name)

		var img *image.RGBA
		if glRenderer != nil {
			img = glRenderer.RenderGL(p)
		} else {
			img = preview.Render(p, preview.RenderConfig{Width: size, Height: size})
		}

		outPath := filepath.Join(outdir, name+".png")
		f, err := os.Create(outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", outPath, err)
			continue
		}
		png.Encode(f, img)
		f.Close()
		fmt.Fprintf(os.Stderr, "Wrote %s\n", outPath)
	}
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
