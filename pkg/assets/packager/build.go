package packager

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/cfoust/sour/pkg/assets"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// BuildMapDirsToDisk scans directories for .ogz files, builds them using the
// disk-based Packager, and writes output to a temp directory. Returns the
// output directory path.
func BuildMapDirsToDisk(ctx context.Context, dirs []string, extraRoots ...string) (string, error) {
	// Collect .ogz files
	type mapEntry struct {
		path string
		root string
	}
	var maps []mapEntry

	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to resolve map dir: %s", dir)
			continue
		}

		filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".ogz") {
				maps = append(maps, mapEntry{path: path, root: absDir})
			}
			return nil
		})
	}

	if len(maps) == 0 {
		return "", nil
	}

	log.Info().Msgf("building %d maps from %d directories", len(maps), len(dirs))

	// Set up roots
	cache := assets.FSStore("/tmp/sour-mapdirs-cache")
	os.MkdirAll("/tmp/sour-mapdirs-cache", 0755)

	var rootStrings []string
	for _, dir := range dirs {
		absDir, _ := filepath.Abs(dir)
		rootStrings = append(rootStrings, absDir)
	}
	rootStrings = append(rootStrings, extraRoots...)

	// Check if we need the remote base game root
	const baseGameRoot = "https://static.sourga.me/blobs/6481/.index.source"
	needsBase := true
	for _, rs := range rootStrings {
		if rs == baseGameRoot {
			needsBase = false
			break
		}
	}
	if needsBase {
		testRoots, _ := assets.LoadRoots(ctx, cache, rootStrings, false)
		found := false
		for _, r := range testRoots {
			if r.Exists(ctx, "data/default_map_settings.cfg") {
				found = true
				break
			}
		}
		if !found {
			log.Info().Msg("mapDirs: fetching base game files from static.sourga.me (needed for texture/model resolution)")
			rootStrings = append(rootStrings, baseGameRoot)
		}
	}

	roots, err := assets.LoadRoots(ctx, cache, rootStrings, false)
	if err != nil {
		return "", fmt.Errorf("failed to load roots: %w", err)
	}

	// Create temp output directory
	outdir, err := os.MkdirTemp("", "sour-mapdirs-*")
	if err != nil {
		return "", err
	}

	p := New(outdir)
	params := BuildParams{
		Roots:          roots,
		RootStrings:    rootStrings,
		BuildWeb:       true,
		DownloadAssets: true,
	}

	// Build maps in parallel
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, m := range maps {
		m := m
		g.Go(func() error {
			relPath, err := filepath.Rel(m.root, m.path)
			if err != nil {
				return nil
			}
			name := strings.TrimSuffix(filepath.Base(m.path), ".ogz")

			sub := New(outdir)
			_, err = sub.BuildMap(gctx, params, relPath, name, "", "")
			if err != nil {
				log.Warn().Err(err).Msgf("failed to build map %s", name)
				return nil
			}

			mu.Lock()
			p.Merge(sub)
			mu.Unlock()

			log.Info().Msgf("built map %s", name)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	if len(p.Maps) == 0 {
		os.RemoveAll(outdir)
		return "", nil
	}

	// Dump index
	if err := p.DumpIndex(""); err != nil {
		return "", fmt.Errorf("dumping index: %w", err)
	}

	// Generate catalog
	catalogDir := filepath.Join(outdir, "catalog")
	os.MkdirAll(catalogDir, 0755)
	catalogMaps := make(map[string]interface{})
	for _, gameMap := range p.Maps {
		entry := map[string]interface{}{
			"description": gameMap.Description,
		}
		if gameMap.Image != "" {
			imgSrc := filepath.Join(outdir, gameMap.Image)
			if _, err := os.Stat(imgSrc); err == nil {
				dest := filepath.Join(catalogDir, gameMap.Image)
				if _, err := os.Stat(dest); os.IsNotExist(err) {
					data, _ := os.ReadFile(imgSrc)
					os.WriteFile(dest, data, 0644)
				}
				entry["image"] = gameMap.Image
			}
		}
		catalogMaps[gameMap.Name] = entry
	}
	catJSON, _ := json.MarshalIndent(map[string]interface{}{"maps": catalogMaps}, "", "  ")
	os.WriteFile(filepath.Join(catalogDir, "catalog.json"), catJSON, 0644)

	log.Info().Msgf("built %d maps to %s", len(p.Maps), outdir)

	return outdir, nil
}
