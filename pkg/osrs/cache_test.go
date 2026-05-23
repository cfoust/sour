package osrs

import (
	"os"
	"testing"
)

const testCacheDir = "/Users/cfoust/Map-Viewer"

func skipIfNoCache(t *testing.T) {
	if _, err := os.Stat(testCacheDir); os.IsNotExist(err) {
		t.Skip("OSRS cache not found at", testCacheDir)
	}
}

func TestOpenCache(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Check that we can count index entries
	for i := 0; i < numIndices; i++ {
		count, err := c.IndexEntryCount(i)
		if err != nil {
			continue
		}
		t.Logf("index %d: %d entries", i, count)
	}
}

func TestReadFile(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Try reading files from index 1 (models, largest index)
	count, err := c.IndexEntryCount(1)
	if err != nil {
		t.Fatal(err)
	}

	read := 0
	var lastErr error
	for id := 1; id < count && read < 5; id++ {
		data, err := c.ReadFile(1, id)
		if err != nil {
			lastErr = err
			continue
		}
		t.Logf("index 1, file %d: %d bytes", id, len(data))
		read++
	}

	if read == 0 {
		t.Fatalf("could not read any files from index 1 (last error: %v)", lastErr)
	}
}

func TestArchive(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Archive 1 from index 0 should contain config data
	data, err := c.ReadFile(0, 2)
	if err != nil {
		t.Fatal("reading archive:", err)
	}
	t.Logf("archive raw size: %d bytes", len(data))

	archive, err := DecodeArchive(data)
	if err != nil {
		t.Fatal("decoding archive:", err)
	}
	t.Logf("archive has %d entries", archive.entries)

	// List all entries by hash
	t.Logf("entry hashes: %v", archive.ids)

	// Check what the expected hash of flo.dat is
	t.Logf("hash of 'flo.dat': %d", hashName("flo.dat"))

	// Try known config file names across all archive files
	names := []string{"flo.dat", "flo.idx", "loc.dat", "loc.idx", "map_index", "obj.dat", "obj.idx", "npc.dat"}
	for fileID := 0; fileID < 9; fileID++ {
		data, err := c.ReadFile(0, fileID)
		if err != nil {
			continue
		}
		a, err := DecodeArchive(data)
		if err != nil {
			t.Logf("archive %d: decode error: %v", fileID, err)
			continue
		}
		t.Logf("archive %d: %d entries, hashes=%v", fileID, a.entries, a.ids[:min(5, len(a.ids))])
		for _, name := range names {
			f, err := a.ReadFile(name)
			if err == nil {
				t.Logf("  found %s: %d bytes", name, len(f))
			}
		}
	}

	// Also check expected hashes
	for _, name := range names {
		t.Logf("  hash(%q) = %d", name, hashName(name))
	}
}

func TestReadFileGzip(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Terrain files in index 4 are gzip-compressed
	count, err := c.IndexEntryCount(4)
	if err != nil {
		t.Fatal(err)
	}

	read := 0
	for id := 0; id < count && read < 3; id++ {
		data, err := c.ReadFileGzip(4, id)
		if err != nil {
			continue
		}
		t.Logf("index 4, file %d: %d bytes (decompressed)", id, len(data))
		read++
	}
}

func TestMapIndex(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Load config archive (file 5 from index 0 had map_index)
	data, err := c.ReadFile(0, 5)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := DecodeArchive(data)
	if err != nil {
		t.Fatal(err)
	}

	mapData, err := archive.ReadFile("map_index")
	if err != nil {
		t.Fatal(err)
	}

	idx, err := LoadMapIndex(mapData)
	if err != nil {
		t.Fatal(err)
	}

	regions := idx.Regions()
	t.Logf("map index has %d regions", len(regions))

	// Check a few known regions have valid file IDs
	valid := 0
	for _, r := range regions {
		if r.TerrainFile > 0 && r.TerrainFile <= 3535 {
			valid++
		}
	}
	t.Logf("%d regions have valid terrain files", valid)

	if valid == 0 {
		t.Fatal("no valid terrain files found")
	}
}

func TestFloorDefs(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	data, err := c.ReadFile(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := DecodeArchive(data)
	if err != nil {
		t.Fatal(err)
	}

	floData, err := archive.ReadFile("flo.dat")
	if err != nil {
		t.Fatal(err)
	}

	defs, err := LoadFloorDefs(floData)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("underlays: %d, overlays: %d", len(defs.Underlays), len(defs.Overlays))

	// Print a few floors
	for i := 0; i < min(5, len(defs.Underlays)); i++ {
		f := defs.Underlays[i]
		c := f.Color()
		t.Logf("  underlay %d: RGB=#%06x (%d,%d,%d)", i, f.RGB, c.R, c.G, c.B)
	}
	for i := 0; i < min(5, len(defs.Overlays)); i++ {
		f := defs.Overlays[i]
		t.Logf("  overlay %d: RGB=#%06x tex=%d occlude=%v", i, f.RGB, f.Texture, f.Occlude)
	}
}

func TestObjectDefs(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	data, err := c.ReadFile(0, 2)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := DecodeArchive(data)
	if err != nil {
		t.Fatal(err)
	}

	locDat, err := archive.ReadFile("loc.dat")
	if err != nil {
		t.Fatal(err)
	}
	locIdx, err := archive.ReadFile("loc.idx")
	if err != nil {
		t.Fatal(err)
	}

	defs, err := LoadObjectDefs(locDat, locIdx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("object definitions: %d", defs.Count())

	// Print a few known objects
	for _, id := range []int{0, 1, 2, 10, 100, 1000, 2000} {
		if id >= defs.Count() {
			continue
		}
		def := defs.Get(id)
		t.Logf("  obj %d: name=%q size=%dx%d solid=%v walkable=%v",
			id, def.Name, def.SizeX, def.SizeY, def.Solid, def.Walkable)
	}
}

func TestRegionTerrain(t *testing.T) {
	skipIfNoCache(t)

	c, err := OpenCache(testCacheDir)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// Load map index
	data, err := c.ReadFile(0, 5)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := DecodeArchive(data)
	if err != nil {
		t.Fatal(err)
	}
	mapData, err := archive.ReadFile("map_index")
	if err != nil {
		t.Fatal(err)
	}
	idx, err := LoadMapIndex(mapData)
	if err != nil {
		t.Fatal(err)
	}

	// Find a region with interesting terrain (varied heights)
	regions := idx.Regions()
	var terrainData []byte
	var regionEntry RegionEntry
	for _, r := range regions {
		if r.TerrainFile <= 0 || r.TerrainFile > 3535 {
			continue
		}
		td, err := c.ReadFileGzip(4, r.TerrainFile)
		if err != nil {
			continue
		}
		// Parse quickly to check if it has height variation
		testReg := &Region{}
		if testReg.LoadTerrain(td, 0, 0) != nil {
			continue
		}
		hasVariation := false
		for x := 0; x < 64 && !hasVariation; x++ {
			for y := 0; y < 64 && !hasVariation; y++ {
				if testReg.Heights[0][x][y] != 0 {
					hasVariation = true
				}
			}
		}
		if !hasVariation {
			continue
		}
		terrainData = td
		regionEntry = r
		break
	}

	if terrainData == nil {
		t.Fatal("could not read any terrain data")
	}

	t.Logf("loading terrain for area %d (file %d): %d bytes", regionEntry.AreaID, regionEntry.TerrainFile, len(terrainData))

	region := &Region{}
	err = region.LoadTerrain(terrainData, 0, 0)
	if err != nil {
		t.Fatal("loading terrain:", err)
	}

	// Print height stats
	minH, maxH := region.Heights[0][0][0], region.Heights[0][0][0]
	underlayCount := 0
	overlayCount := 0
	for x := 0; x < RegionSize; x++ {
		for y := 0; y < RegionSize; y++ {
			h := region.Heights[0][x][y]
			if h < minH {
				minH = h
			}
			if h > maxH {
				maxH = h
			}
			if region.Underlays[0][x][y] != 0 {
				underlayCount++
			}
			if region.Overlays[0][x][y] != 0 {
				overlayCount++
			}
		}
	}
	t.Logf("plane 0: height range [%d, %d]", minH, maxH)
	t.Logf("plane 0: %d tiles with underlays, %d with overlays", underlayCount, overlayCount)

	// Load objects if available
	if regionEntry.ObjectFile > 0 && regionEntry.ObjectFile <= 3535 {
		objData, err := c.ReadFileGzip(4, regionEntry.ObjectFile)
		if err == nil {
			objects, err := LoadObjects(objData, 0, 0)
			if err != nil {
				t.Fatal("loading objects:", err)
			}
			t.Logf("objects: %d placed", len(objects))

			// Count by type
			typeCounts := make(map[int]int)
			for _, obj := range objects {
				typeCounts[obj.Type]++
			}
			for typ, count := range typeCounts {
				t.Logf("  type %d: %d objects", typ, count)
			}
		}
	}
}
