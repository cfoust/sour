package main

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
	"github.com/cfoust/sour/pkg/assets/packager"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// BaseCmd builds base game assets.
type BaseCmd struct {
	Outdir       string   `help:"Output directory." default:"output/"`
	Textures     bool     `help:"Include all textures."`
	Models       bool     `help:"Include all models."`
	Download     bool     `help:"Download remote assets."`
	Mobile       bool     `help:"Compress textures for mobile."`
	Prefix       string   `help:"Index file prefix." default:""`
	Root         string   `help:"Base game root." required:"" name:"game-root"`
	PlayerModels bool     `help:"Include only player models." name:"player-models"`
	Maps         []string `arg:"" optional:"" help:"Specific map names to build."`
}

func (cmd *BaseCmd) Run() error {
	ctx := context.Background()

	roots := append(CLI.Root, cmd.Root)
	rootStrings := []string{"sour", cmd.Root}

	cache := assets.FSStore(CLI.Cache)
	os.MkdirAll(CLI.Cache, 0755)

	assetRoots, err := assets.LoadRoots(ctx, cache, rootStrings, false)
	if err != nil {
		return fmt.Errorf("failed to load roots: %w", err)
	}

	_ = roots

	os.MkdirAll(cmd.Outdir, 0755)

	p := packager.New(cmd.Outdir)

	params := packager.BuildParams{
		Roots:          assetRoots,
		RootStrings:    rootStrings,
		SkipRoot:       cmd.Root,
		CompressImages: cmd.Mobile,
		DownloadAssets: cmd.Download,
		BuildWeb:       false,
		BuildDesktop:   false,
	}

	// Get all root files
	files, err := packager.GetRootFiles(ctx, assetRoots)
	if err != nil {
		return fmt.Errorf("getting root files: %w", err)
	}

	// Determine maps to build
	var mapFiles []string
	if len(cmd.Maps) > 0 {
		if cmd.Maps[0] == "none" {
			mapFiles = nil
		} else {
			for _, m := range cmd.Maps {
				mapFiles = append(mapFiles, fmt.Sprintf("packages/base/%s.ogz", m))
			}
		}
	} else {
		for _, f := range files {
			if strings.HasSuffix(f, ".ogz") {
				mapFiles = append(mapFiles, f)
			}
		}
	}
	// Always include xmwhub
	mapFiles = append(mapFiles, "packages/base/xmwhub.ogz")

	if cmd.Textures || (!cmd.Textures && !cmd.Models && !cmd.PlayerModels && len(cmd.Maps) == 0) {
		log.Info().Msg("building textures")
		var textures []string
		for _, f := range files {
			if strings.HasSuffix(f, ".jpg") || strings.HasSuffix(f, ".png") {
				textures = append(textures, f)
			}
		}
		if err := p.BuildTextures(ctx, params, textures); err != nil {
			return fmt.Errorf("building textures: %w", err)
		}
	}

	fpsModels := append([]string{}, BASE_MODELS...)
	fpsModels = append(fpsModels, SNOUT_MODELS...)
	if !cmd.Mobile {
		fpsModels = append(fpsModels, OTHER_MODELS...)
	}

	if cmd.Models || (!cmd.Textures && !cmd.Models && !cmd.PlayerModels && len(cmd.Maps) == 0) {
		log.Info().Msg("building models")
		modelTypes := []string{"md2", "md3", "md5", "obj", "smd", "iqm"}

		var modelIDs []string
		seen := make(map[string]bool)
		for _, f := range files {
			if !strings.HasPrefix(f, "packages/models") {
				continue
			}
			isModel := false
			for _, t := range modelTypes {
				if strings.HasSuffix(f, t+".cfg") || strings.HasSuffix(f, "tris."+t) {
					isModel = true
					break
				}
			}
			if !isModel {
				continue
			}

			id := filepath.Dir(f[len("packages/models/"):])
			if seen[id] {
				continue
			}
			seen[id] = true

			// Skip FPS models (built separately)
			isFPS := false
			for _, fm := range fpsModels {
				if id == fm {
					isFPS = true
					break
				}
			}
			if isFPS {
				continue
			}

			modelIDs = append(modelIDs, id)
		}

		for _, id := range modelIDs {
			_, err := p.BuildModel(ctx, params, id)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to build model %s", id)
			}
		}
	}

	// Build base mod
	log.Info().Msg("building base mod")
	baseFiles := strings.Split(strings.TrimSpace(baseListData), "\n")
	baseMappings, err := packager.QueryFiles(ctx, assetRoots, baseFiles)
	if err != nil {
		return fmt.Errorf("querying base files: %w", err)
	}

	baseParams := params
	baseParams.DownloadAssets = true
	baseParams.BuildWeb = true
	_, err = p.BuildMod(ctx, baseParams, baseMappings, "base", "Everything the base game needs.", "")
	if err != nil {
		return fmt.Errorf("building base mod: %w", err)
	}

	// Build fps mod
	log.Info().Msg("building fps mod")
	var fpsMappings []packager.Mapping
	fpsMounted := make(map[string]bool)
	for _, model := range fpsModels {
		modelFiles, err := packager.DumpSour(ctx, "model", model, assetRoots)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to dump model %s", model)
			continue
		}
		for _, m := range modelFiles {
			if fpsMounted[m.To] {
				continue
			}
			fpsMounted[m.To] = true
			fpsMappings = append(fpsMappings, m)
		}
	}

	fpsParams := params
	fpsParams.DownloadAssets = true
	fpsParams.BuildWeb = true
	_, err = p.BuildMod(ctx, fpsParams, fpsMappings, "fps", "All of the base game FPS models.", "")
	if err != nil {
		return fmt.Errorf("building fps mod: %w", err)
	}

	// Build maps in parallel
	if len(mapFiles) > 0 {
		log.Info().Msgf("building %d maps", len(mapFiles))
		var mu sync.Mutex
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(runtime.NumCPU())

		for _, mapFile := range mapFiles {
			mapFile := mapFile
			g.Go(func() error {
				sub := packager.New(cmd.Outdir)
				base := strings.TrimSuffix(filepath.Base(mapFile), filepath.Ext(mapFile))
				desc := fmt.Sprintf("Base game map %s as it appeared in game version r6481.", base)

				_, err := sub.BuildMap(gctx, params, mapFile, base, desc, "")
				if err != nil {
					log.Warn().Err(err).Msgf("failed to build map %s", mapFile)
					return nil // Don't fail the whole batch
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

	// Dump index
	if err := p.DumpIndex(cmd.Prefix); err != nil {
		return fmt.Errorf("dumping index: %w", err)
	}

	// Generate base catalog
	catalogDir := filepath.Join(cmd.Outdir, "catalog")
	os.MkdirAll(catalogDir, 0755)

	catalogMaps := make(map[string]interface{})
	for _, gameMap := range p.Maps {
		entry := map[string]interface{}{
			"author":      "Sauerbraten",
			"description": gameMap.Description,
		}
		if gameMap.Image != "" {
			imgSrc := filepath.Join(cmd.Outdir, gameMap.Image)
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
	catalogJSON, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	catalogPath := filepath.Join(catalogDir, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogJSON, 0644); err != nil {
		return err
	}
	log.Info().Msgf("wrote base catalog with %d maps to %s", len(catalogMaps), catalogDir)

	return nil
}
