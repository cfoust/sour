package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/osrs"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var CLI struct {
	Debug bool   `help:"Enable debug logging."`
	Cache string `help:"Path to OSRS cache directory." default:"~/Map-Viewer" env:"OSRS_CACHE"`

	Regions  RegionsCmd  `cmd:"" help:"List available map regions."`
	Textures TexturesCmd `cmd:"" help:"Extract floor textures as PNG."`
	Convert  ConvertCmd  `cmd:"" help:"Convert an OSRS region to a Sauerbraten .ogz map."`
}

// RegionsCmd lists available regions in the cache.
type RegionsCmd struct{}

func (cmd *RegionsCmd) Run() error {
	c, idx, err := loadCacheAndIndex()
	if err != nil {
		return err
	}
	defer c.Close()

	regions := idx.Regions()
	valid := 0
	for _, r := range regions {
		if r.TerrainFile > 0 && r.TerrainFile <= 3535 {
			regionX := r.AreaID >> 8
			regionY := r.AreaID & 0xFF
			fmt.Printf("%d,%d (area %d): terrain=%d objects=%d\n",
				regionX, regionY, r.AreaID, r.TerrainFile, r.ObjectFile)
			valid++
		}
	}
	fmt.Printf("\n%d regions with valid terrain\n", valid)
	return nil
}

// TexturesCmd extracts OSRS floor textures as PNGs.
type TexturesCmd struct {
	Outdir string `help:"Output directory." default:"output/osrs-textures"`
}

func (cmd *TexturesCmd) Run() error {
	c, err := openCache()
	if err != nil {
		return err
	}
	defer c.Close()

	defs, err := loadFloorDefs(c)
	if err != nil {
		return err
	}

	os.MkdirAll(cmd.Outdir, 0755)

	// Generate solid-color PNGs for each underlay
	for i, f := range defs.Underlays {
		col := f.Color()
		if col.R == 0 && col.G == 0 && col.B == 0 {
			continue
		}
		path := filepath.Join(cmd.Outdir, fmt.Sprintf("underlay_%d.png", i))
		if err := writeSolidPNG(path, col, 64); err != nil {
			log.Warn().Err(err).Msgf("writing underlay %d", i)
		}
	}

	// Generate solid-color PNGs for each overlay
	for i, f := range defs.Overlays {
		col := f.Color()
		if col.R == 0 && col.G == 0 && col.B == 0 && f.Texture < 0 {
			continue
		}
		path := filepath.Join(cmd.Outdir, fmt.Sprintf("overlay_%d.png", i))
		if err := writeSolidPNG(path, col, 64); err != nil {
			log.Warn().Err(err).Msgf("writing overlay %d", i)
		}
	}

	log.Info().Msgf("wrote %d underlays + %d overlays to %s",
		len(defs.Underlays), len(defs.Overlays), cmd.Outdir)

	// Write package.cfg
	cfgPath := filepath.Join(cmd.Outdir, "package.cfg")
	f, err := os.Create(cfgPath)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "// OSRS floor textures (auto-generated)\n")
	for i := range defs.Underlays {
		fmt.Fprintf(f, "setshader stdworld\n")
		fmt.Fprintf(f, "texture 0 \"packages/osrs/underlay_%d.png\"\n", i)
	}
	for i := range defs.Overlays {
		fmt.Fprintf(f, "setshader stdworld\n")
		fmt.Fprintf(f, "texture 0 \"packages/osrs/overlay_%d.png\"\n", i)
	}

	return nil
}

// ConvertCmd converts an OSRS region to a Sauerbraten root directory.
type ConvertCmd struct {
	RegionX int    `arg:"" help:"Region X coordinate (tile / 64)."`
	RegionY int    `arg:"" help:"Region Y coordinate (tile / 64)."`
	Outdir  string `help:"Output root directory." default:"input/roots/osrs"`
}

func (cmd *ConvertCmd) Run() error {
	c, idx, err := loadCacheAndIndex()
	if err != nil {
		return err
	}
	defer c.Close()

	floorDefs, err := loadFloorDefs(c)
	if err != nil {
		return err
	}

	objDefs, err := loadObjectDefs(c)
	if err != nil {
		return err
	}

	// Find region
	terrainFile := idx.TerrainFileID(cmd.RegionX, cmd.RegionY)
	if terrainFile < 0 {
		return fmt.Errorf("region %d,%d not found in map index", cmd.RegionX, cmd.RegionY)
	}

	terrainData, err := c.ReadFileGzip(4, terrainFile)
	if err != nil {
		return fmt.Errorf("reading terrain data: %w", err)
	}

	region := &osrs.Region{}
	if err := region.LoadTerrain(terrainData, 0, 0); err != nil {
		return fmt.Errorf("parsing terrain: %w", err)
	}

	// Load objects if available
	objectFile := idx.ObjectFileID(cmd.RegionX, cmd.RegionY)
	var objects []osrs.PlacedObject
	if objectFile > 0 {
		objData, err := c.ReadFileGzip(4, objectFile)
		if err == nil {
			objects, _ = osrs.LoadObjects(objData, 0, 0)
		}
	}

	log.Info().Msgf("region %d,%d: terrain loaded, %d objects", cmd.RegionX, cmd.RegionY, len(objects))

	// Load OSRS model textures
	osrsTextures, texErr := osrs.LoadTextures(c)
	if texErr != nil {
		log.Warn().Err(texErr).Msg("failed to load OSRS textures, models will use color atlas only")
	}

	gameMap, modelKeys, _, err := convertRegion(region, floorDefs, objDefs, objects)
	if err != nil {
		return fmt.Errorf("converting region: %w", err)
	}

	mapName := fmt.Sprintf("osrs_%d_%d", cmd.RegionX, cmd.RegionY)

	// Create Sauerbraten root directory structure
	baseDir := filepath.Join(cmd.Outdir, "packages", "base")
	osrsDir := filepath.Join(cmd.Outdir, "packages", "osrs")
	modelsDir := filepath.Join(cmd.Outdir, "packages", "models")
	os.MkdirAll(baseDir, 0755)
	os.MkdirAll(osrsDir, 0755)

	// Write the .ogz map file
	ogzPath := filepath.Join(baseDir, mapName+".ogz")
	if err := maps.ToFile(gameMap, ogzPath); err != nil {
		return fmt.Errorf("writing map: %w", err)
	}

	// Export OSRS models as OBJ files for each unique (object, rotation) pair
	mapmodelLines := exportModels(c, modelsDir, modelKeys, objDefs, osrsTextures)
	_ = osrsTextures

	// Write texture PNGs
	writeTexturePNGs(osrsDir, floorDefs)

	// Write the map .cfg with texture slots and mapmodel registrations
	// Texture slots must match the indices used on cube faces.
	// Slot 0 = sky (registered by engine), so our slots start at 1.
	cfgPath := filepath.Join(baseDir, mapName+".cfg")
	cfg := fmt.Sprintf("// Auto-generated OSRS region %d,%d\n\n", cmd.RegionX, cmd.RegionY)
	cfg += "texturereset\nmapmodelreset\n\n"

	// Slot 0 after reset is the first texture registered.
	// Sauer's texture command prepends "packages/" internally,
	// so paths should be relative to packages/.
	cfg += "// slot 0: default\nsetshader stdworld\ntexture 0 \"osrs/underlay_0.png\"\n\n"

	for i := 1; i < len(floorDefs.Underlays); i++ {
		cfg += fmt.Sprintf("setshader stdworld\ntexture 0 \"osrs/underlay_%d.png\"\n", i)
	}
	for i := range floorDefs.Overlays {
		cfg += fmt.Sprintf("setshader stdworld\ntexture 0 \"osrs/overlay_%d.png\"\n", i)
	}
	cfg += "\n"

	for _, line := range mapmodelLines {
		cfg += line + "\n"
	}
	os.WriteFile(cfgPath, []byte(cfg), 0644)

	log.Info().Msgf("wrote %s (entities: %d, models: %d, map: %s)",
		cmd.Outdir, len(gameMap.Entities), len(modelKeys), mapName)
	return nil
}

// Helper functions

func openCache() (*osrs.Cache, error) {
	cacheDir := CLI.Cache
	if cacheDir[:2] == "~/" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, cacheDir[2:])
	}
	return osrs.OpenCache(cacheDir)
}

func loadCacheAndIndex() (*osrs.Cache, *osrs.MapIndex, error) {
	c, err := openCache()
	if err != nil {
		return nil, nil, err
	}

	// Load config archive (file 5 from index 0 contains map_index)
	data, err := c.ReadFile(0, 5)
	if err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("reading config archive: %w", err)
	}

	archive, err := osrs.DecodeArchive(data)
	if err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("decoding config archive: %w", err)
	}

	mapData, err := archive.ReadFile("map_index")
	if err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("reading map_index: %w", err)
	}

	idx, err := osrs.LoadMapIndex(mapData)
	if err != nil {
		c.Close()
		return nil, nil, fmt.Errorf("parsing map_index: %w", err)
	}

	return c, idx, nil
}

func loadObjectDefs(c *osrs.Cache) (*osrs.ObjectDefs, error) {
	data, err := c.ReadFile(0, 2)
	if err != nil {
		return nil, fmt.Errorf("reading config archive: %w", err)
	}

	archive, err := osrs.DecodeArchive(data)
	if err != nil {
		return nil, fmt.Errorf("decoding config archive: %w", err)
	}

	locDat, err := archive.ReadFile("loc.dat")
	if err != nil {
		return nil, fmt.Errorf("reading loc.dat: %w", err)
	}

	locIdx, err := archive.ReadFile("loc.idx")
	if err != nil {
		return nil, fmt.Errorf("reading loc.idx: %w", err)
	}

	return osrs.LoadObjectDefs(locDat, locIdx)
}

func loadFloorDefs(c *osrs.Cache) (*osrs.FloorDefs, error) {
	data, err := c.ReadFile(0, 2)
	if err != nil {
		return nil, fmt.Errorf("reading floor archive: %w", err)
	}

	archive, err := osrs.DecodeArchive(data)
	if err != nil {
		return nil, fmt.Errorf("decoding floor archive: %w", err)
	}

	floData, err := archive.ReadFile("flo.dat")
	if err != nil {
		return nil, fmt.Errorf("reading flo.dat: %w", err)
	}

	return osrs.LoadFloorDefs(floData)
}

// exportModels exports OSRS models as OBJ files and returns mapmodel cfg lines.
// modelIDs maps cache model ID → mapmodel index.
func exportModels(c *osrs.Cache, modelsDir string, keys []modelKey, objDefs *osrs.ObjectDefs, textures []*image.RGBA) []string {
	lines := make([]string, 0, len(keys))

	for _, key := range keys {
		def := objDefs.Get(key.objID)
		if len(def.ModelIDs) == 0 {
			lines = append(lines, "mmodel osrs/placeholder")
			continue
		}

		cacheID := def.ModelIDs[0]
		modelName := fmt.Sprintf("osrs_%d_r%d", key.objID, key.rotation)
		modelDir := filepath.Join(modelsDir, modelName)
		os.MkdirAll(modelDir, 0755)

		data, err := c.ReadFileGzip(1, cacheID)
		if err != nil {
			log.Warn().Err(err).Msgf("reading model %d", cacheID)
			lines = append(lines, "mmodel osrs/placeholder")
			continue
		}

		model, err := osrs.DecodeModel(data)
		if err != nil {
			log.Warn().Err(err).Msgf("decoding model %d", cacheID)
			lines = append(lines, "mmodel osrs/placeholder")
			continue
		}

		// Apply ObjectDefinition transforms matching Java ObjectDefinition.model():
		// 1. rotate90Degrees() called orientation times
		// 2. scale(scaleX, scaleZ, scaleY)
		// 3. translate(translateX, translateY, translateZ)
		for r := 0; r < key.rotation; r++ {
			model.Rotate90()
		}
		model.ApplyObjectDef(def)

		// Sanity check
		maxCoord := 0
		for i := range model.VertexX {
			for _, v := range []int{model.VertexX[i], model.VertexY[i], model.VertexZ[i]} {
				if v < 0 {
					v = -v
				}
				if v > maxCoord {
					maxCoord = v
				}
			}
		}
		if maxCoord > 10000 {
			log.Warn().Msgf("skipping model %d_r%d: too large (%d)", key.objID, key.rotation, maxCoord)
			lines = append(lines, "mmodel osrs/placeholder")
			continue
		}

		objContent, _ := model.ToOBJ(modelName, modelScaleValue)
		os.WriteFile(filepath.Join(modelDir, "tris.obj"), []byte(objContent), 0644)

		// Determine skin texture: use actual OSRS texture if model has one
		skinName := "skin.png"
		dominantTexID := model.DominantTexture()
		if dominantTexID >= 0 && textures != nil && dominantTexID < len(textures) && textures[dominantTexID] != nil {
			// Use the actual OSRS texture
			skinPath := filepath.Join(modelDir, skinName)
			af, err := os.Create(skinPath)
			if err == nil {
				png.Encode(af, textures[dominantTexID])
				af.Close()
			}
		} else {
			// Fall back to color atlas
			atlas := model.ColorAtlas()
			skinPath := filepath.Join(modelDir, skinName)
			af, err := os.Create(skinPath)
			if err == nil {
				png.Encode(af, atlas)
				af.Close()
			}
		}

		objCfg := "objload tris.obj\nobjskin * skin.png\nmdlscale 400\nmdlambient 80\nmdlshadow 1\n"
		os.WriteFile(filepath.Join(modelDir, "obj.cfg"), []byte(objCfg), 0644)

		lines = append(lines, fmt.Sprintf("mmodel %s", modelName))
	}

	return lines
}

func writeTexturePNGs(outdir string, defs *osrs.FloorDefs) {
	os.MkdirAll(outdir, 0755)

	for i, f := range defs.Underlays {
		col := f.Color()
		if col.R == 0 && col.G == 0 && col.B == 0 {
			col = color.RGBA{R: 128, G: 128, B: 128, A: 255}
		}
		path := filepath.Join(outdir, fmt.Sprintf("underlay_%d.png", i))
		writeSolidPNG(path, col, 64)
	}

	for i, f := range defs.Overlays {
		col := f.Color()
		if col.R == 0 && col.G == 0 && col.B == 0 && f.Texture < 0 {
			col = color.RGBA{R: 96, G: 96, B: 96, A: 255}
		}
		path := filepath.Join(outdir, fmt.Sprintf("overlay_%d.png", i))
		writeSolidPNG(path, col, 64)
	}
}

func writeSolidPNG(path string, col color.RGBA, size int) error {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, col)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func main() {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = log.Output(consoleWriter)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ctx := kong.Parse(&CLI,
		kong.Name("osrs2sauer"),
		kong.Description("Convert OSRS maps to Sauerbraten .ogz format."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	if CLI.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

