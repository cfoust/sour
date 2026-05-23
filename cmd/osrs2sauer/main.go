package main

import (
	"encoding/binary"
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
			fmt.Printf("area %d: terrain=%d objects=%d\n", r.AreaID, r.TerrainFile, r.ObjectFile)
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

// ConvertCmd converts an OSRS region to a Sauerbraten map.
type ConvertCmd struct {
	Region int    `arg:"" help:"Region area ID to convert."`
	Output string `help:"Output .ogz file path." default:"output/osrs.ogz"`
}

func (cmd *ConvertCmd) Run() error {
	c, idx, err := loadCacheAndIndex()
	if err != nil {
		return err
	}
	defer c.Close()

	defs, err := loadFloorDefs(c)
	if err != nil {
		return err
	}

	// Find region
	terrainFile := idx.TerrainFileID(0, cmd.Region)
	if terrainFile < 0 {
		return fmt.Errorf("region %d not found in map index", cmd.Region)
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
	objectFile := idx.ObjectFileID(0, cmd.Region)
	var objects []osrs.PlacedObject
	if objectFile > 0 {
		objData, err := c.ReadFileGzip(4, objectFile)
		if err == nil {
			objects, _ = osrs.LoadObjects(objData, 0, 0)
		}
	}

	log.Info().Msgf("region %d: terrain loaded, %d objects", cmd.Region, len(objects))

	// Convert to Sauerbraten map
	gameMap, err := convertRegion(region, defs, objects)
	if err != nil {
		return fmt.Errorf("converting region: %w", err)
	}

	os.MkdirAll(filepath.Dir(cmd.Output), 0755)
	if err := maps.ToFile(gameMap, cmd.Output); err != nil {
		return fmt.Errorf("writing map: %w", err)
	}

	log.Info().Msgf("wrote %s", cmd.Output)
	return nil
}

// convertRegion converts OSRS terrain data to a Sauerbraten GameMap.
func convertRegion(region *osrs.Region, defs *osrs.FloorDefs, objects []osrs.PlacedObject) (*maps.GameMap, error) {
	// Scale: 1 OSRS tile = 2x2 Sauer cubes at gridpower 3
	// 64 tiles * 2 = 128 cubes per axis → worldSize 256
	const (
		tilesPerChunk = 64
		cubesPerTile  = 2
		worldSize     = 256
		cubeSize      = worldSize / (tilesPerChunk * cubesPerTile) // = 2, but in world units
	)

	m, err := maps.NewMap()
	if err != nil {
		return nil, err
	}
	m.Header.WorldSize = worldSize

	// Build the octree: worldSize 256 = root has 8 children each of size 128
	// We need to subdivide down to individual cubes
	root := buildTerrainOctree(region, defs, worldSize)
	m.WorldRoot = root

	// Add a player spawn at the center
	m.Entities = append(m.Entities, maps.Entity{
		Position: maps.Vector{
			X: float32(worldSize) / 2,
			Y: float32(worldSize) / 2,
			Z: float32(worldSize) / 2,
		},
		Type: maps.ET_PLAYERSTART,
	})

	// Add a sunlight
	m.Entities = append(m.Entities, maps.Entity{
		Position: maps.Vector{
			X: float32(worldSize) / 2,
			Y: float32(worldSize) / 2,
			Z: float32(worldSize) - 10,
		},
		Type:  maps.ET_LIGHT,
		Attr1: 200, // radius
		Attr2: 200, // R
		Attr3: 200, // G
		Attr4: 200, // B
	})

	_ = objects // TODO: convert objects to cubes/entities

	return m, nil
}

// buildTerrainOctree creates an octree from OSRS heightmap data.
// The octree represents a worldSize^3 cube. We subdivide it recursively
// and fill in terrain cubes based on the heightmap.
func buildTerrainOctree(region *osrs.Region, defs *osrs.FloorDefs, worldSize int) *maps.Cube {
	root := &maps.Cube{}
	root.Children = make([]*maps.Cube, maps.CUBE_FACTOR)
	halfSize := worldSize / 2

	for i := 0; i < maps.CUBE_FACTOR; i++ {
		// Octree child order: xyz bits (x=bit0, y=bit1, z=bit2)
		ox := (i & 1) * halfSize
		oy := ((i >> 1) & 1) * halfSize
		oz := ((i >> 2) & 1) * halfSize
		root.Children[i] = buildOctreeNode(region, defs, ox, oy, oz, halfSize)
	}

	return root
}

func buildOctreeNode(region *osrs.Region, defs *osrs.FloorDefs, ox, oy, oz, size int) *maps.Cube {
	// At gridpower 0 (size=1), we're at a leaf cube
	if size <= 1 {
		return buildLeafCube(region, defs, ox, oy, oz)
	}

	// Check if this entire node is above or below the terrain
	// and can be simplified to solid or empty
	halfSize := size / 2

	c := &maps.Cube{}
	c.Children = make([]*maps.Cube, maps.CUBE_FACTOR)
	for i := 0; i < maps.CUBE_FACTOR; i++ {
		cx := ox + (i&1)*halfSize
		cy := oy + ((i>>1)&1)*halfSize
		cz := oz + ((i>>2)&1)*halfSize
		c.Children[i] = buildOctreeNode(region, defs, cx, cy, cz, halfSize)
	}

	return c
}

func buildLeafCube(region *osrs.Region, defs *osrs.FloorDefs, x, y, z int) *maps.Cube {
	c := &maps.Cube{}

	// Map cube coordinates to OSRS tile coordinates
	// 2 cubes per tile, 64 tiles = 128 cubes, worldSize=256
	// But our grid is 128 cubes in x/y out of 256, so cubes 0-127 map to tiles 0-63
	tileX := x / 2
	tileY := y / 2

	if tileX < 0 || tileX >= 64 || tileY < 0 || tileY >= 64 {
		c.EmptyFaces()
		return c
	}

	// Get terrain height (OSRS heights are negative, 0 = sea level)
	// Convert to Sauer Z coordinates: higher Z = higher up
	h := region.Heights[0][tileX][tileY]
	// OSRS heights range roughly -720 to 0
	// Map to Sauer Z range: we want terrain at roughly Z=64 to Z=192
	// in a 256-unit world
	terrainZ := 128 + h/4 // rough mapping

	if z < terrainZ {
		// Below terrain: solid
		c.SolidFaces()

		// Set texture based on underlay
		texIdx := uint16(1) // default
		underlayID := int(region.Underlays[0][tileX][tileY])
		if underlayID > 0 && underlayID <= len(defs.Underlays) {
			texIdx = uint16(underlayID) // will need proper slot mapping
		}
		for i := 0; i < 6; i++ {
			c.Texture[i] = texIdx
		}
	} else {
		// Above terrain: empty
		c.EmptyFaces()
	}

	return c
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

// Ensure binary import is used (needed for Entity serialization)
var _ = binary.LittleEndian
