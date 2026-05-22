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

// RunManifest executes a manifest, building all specified assets.
func RunManifest(ctx context.Context, manifest *Manifest, cliRoots []string, cache assets.Store) error {
	manifest.MergeRoots(cliRoots)

	if manifest.Output == "" {
		manifest.Output = "output/"
	}
	os.MkdirAll(manifest.Output, 0755)

	// Build root strings for LoadRoots
	rootStrings := make([]string, len(manifest.Roots))
	copy(rootStrings, manifest.Roots)

	assetRoots, err := assets.LoadRoots(ctx, cache, rootStrings, false)
	if err != nil {
		return fmt.Errorf("loading roots: %w", err)
	}

	// Enumerate all files in roots
	files, err := GetRootFiles(ctx, assetRoots)
	if err != nil {
		return fmt.Errorf("enumerating root files: %w", err)
	}

	p := New(manifest.Output)

	params := BuildParams{
		Roots:          assetRoots,
		RootStrings:    rootStrings,
		SkipRoot:       manifest.Skip,
		CompressImages: manifest.Compress,
		DownloadAssets: manifest.Download,
		BuildWeb:       manifest.Web,
		BuildDesktop:   manifest.Desktop,
	}

	// Build mods
	for _, mod := range manifest.Mods {
		if err := buildManifestMod(ctx, p, params, mod, assetRoots, files); err != nil {
			log.Warn().Err(err).Msgf("failed to build mod %s", mod.Name)
		}
	}

	// Build individual models
	if manifest.Models != nil {
		allModelNames := EnumerateModelNames(files)
		selected := FilterStrings(*manifest.Models, allModelNames)
		log.Info().Msgf("building %d models", len(selected))
		for _, name := range selected {
			_, err := p.BuildModel(ctx, params, name)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to build model %s", name)
			}
		}
	}

	// Build individual textures
	if manifest.Textures != nil {
		var textureFiles []string
		for _, f := range files {
			if strings.HasSuffix(f, ".jpg") || strings.HasSuffix(f, ".png") {
				textureFiles = append(textureFiles, f)
			}
		}
		selected := FilterStrings(*manifest.Textures, textureFiles)
		log.Info().Msgf("building %d textures", len(selected))
		if err := p.BuildTextures(ctx, params, selected); err != nil {
			return fmt.Errorf("building textures: %w", err)
		}
	}

	// Build maps in parallel
	if manifest.Maps != nil {
		allMapNames := EnumerateMapNames(files)
		selected := FilterStrings(*manifest.Maps, allMapNames)
		if len(selected) > 0 {
			log.Info().Msgf("building %d maps", len(selected))
			var mu sync.Mutex
			g, gctx := errgroup.WithContext(ctx)
			g.SetLimit(runtime.NumCPU())

			for _, name := range selected {
				name := name
				g.Go(func() error {
					mapFile := fmt.Sprintf("packages/base/%s.ogz", name)
					sub := New(manifest.Output)
					_, err := sub.BuildMap(gctx, params, mapFile, name, "", "")
					if err != nil {
						log.Warn().Err(err).Msgf("failed to build map %s", name)
						return nil
					}
					mu.Lock()
					p.Merge(sub)
					mu.Unlock()
					return nil
				})
			}

			if err := g.Wait(); err != nil {
				return err
			}
		}
	}

	// Dump index
	if err := p.DumpIndex(manifest.Prefix); err != nil {
		return fmt.Errorf("dumping index: %w", err)
	}

	// Generate catalog if maps were built
	if len(p.Maps) > 0 {
		if err := generateCatalog(manifest.Output, p); err != nil {
			log.Warn().Err(err).Msg("failed to generate catalog")
		}
	}

	log.Info().Msgf("built %d maps, %d mods, %d models, %d textures",
		len(p.Maps), len(p.Mods), len(p.Models), len(p.Textures))

	return nil
}

func buildManifestMod(
	ctx context.Context,
	p *Packager,
	params BuildParams,
	mod ManifestMod,
	roots []assets.Root,
	allFiles []string,
) error {
	log.Info().Msgf("building mod %s", mod.Name)

	var mappings []Mapping
	mounted := make(map[string]bool)

	addMappings := func(ms []Mapping) {
		for _, m := range ms {
			if mounted[m.To] {
				continue
			}
			mounted[m.To] = true
			mappings = append(mappings, m)
		}
	}

	// files: pattern-matched root-relative paths
	if mod.Files != nil {
		selected := FilterStrings(*mod.Files, allFiles)
		if len(selected) > 0 {
			resolved, err := QueryFiles(ctx, roots, selected)
			if err != nil {
				return fmt.Errorf("resolving mod files: %w", err)
			}
			addMappings(resolved)
		}
	}

	// models: pattern-matched model names
	if mod.Models != nil {
		allModelNames := EnumerateModelNames(allFiles)
		selected := FilterStrings(*mod.Models, allModelNames)
		for _, name := range selected {
			modelFiles, err := DumpSour(ctx, "model", name, roots)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to dump model %s", name)
				continue
			}
			addMappings(modelFiles)
		}
	}

	// maps: pattern-matched map names
	if mod.Maps != nil {
		allMapNames := EnumerateMapNames(allFiles)
		selected := FilterStrings(*mod.Maps, allMapNames)
		for _, name := range selected {
			mapFile := fmt.Sprintf("packages/base/%s.ogz", name)
			mapFiles, err := DumpSour(ctx, "map", mapFile, roots)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to dump map %s", name)
				continue
			}
			addMappings(mapFiles)
		}
	}

	// textures: pattern-matched file paths
	if mod.Textures != nil {
		var textureFiles []string
		for _, f := range allFiles {
			if strings.HasSuffix(f, ".jpg") || strings.HasSuffix(f, ".png") {
				textureFiles = append(textureFiles, f)
			}
		}
		selected := FilterStrings(*mod.Textures, textureFiles)
		if len(selected) > 0 {
			resolved, err := QueryFiles(ctx, roots, selected)
			if err != nil {
				return fmt.Errorf("resolving mod textures: %w", err)
			}
			addMappings(resolved)
		}
	}

	if len(mappings) == 0 {
		log.Warn().Msgf("mod %s has no files", mod.Name)
		return nil
	}

	modParams := params
	modParams.DownloadAssets = true
	modParams.BuildWeb = true

	_, err := p.BuildMod(ctx, modParams, mappings, mod.Name, mod.Description, mod.Image)
	return err
}

func generateCatalog(outdir string, p *Packager) error {
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

	catalog := map[string]interface{}{"maps": catalogMaps}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}

	catalogPath := filepath.Join(catalogDir, "catalog.json")
	if err := os.WriteFile(catalogPath, data, 0644); err != nil {
		return err
	}

	log.Info().Msgf("wrote catalog with %d maps to %s", len(catalogMaps), catalogDir)
	return nil
}
