package osrs

import "fmt"

// MapIndex maps region coordinates to cache file IDs.
type MapIndex struct {
	areas      []int
	mapFiles   []int
	landscapes []int
}

// LoadMapIndex parses the map_index file from a config archive.
func LoadMapIndex(data []byte) (*MapIndex, error) {
	buf := NewBuffer(data)

	size, err := buf.ReadUShort()
	if err != nil {
		return nil, fmt.Errorf("reading map index size: %w", err)
	}

	m := &MapIndex{
		areas:      make([]int, size),
		mapFiles:   make([]int, size),
		landscapes: make([]int, size),
	}

	for i := 0; i < size; i++ {
		area, err := buf.ReadUShort()
		if err != nil {
			return nil, err
		}
		m.areas[i] = area

		mapFile, err := buf.ReadUShort()
		if err != nil {
			return nil, err
		}
		m.mapFiles[i] = mapFile

		landscape, err := buf.ReadUShort()
		if err != nil {
			return nil, err
		}
		m.landscapes[i] = landscape
	}

	return m, nil
}

// TerrainFileID returns the cache file ID for terrain data at the given region.
// regionX and regionY are in region coordinates (tile / 64).
// Returns -1 if not found.
func (m *MapIndex) TerrainFileID(regionX, regionY int) int {
	id := (regionX << 8) | regionY
	for i, area := range m.areas {
		if area == id {
			if m.mapFiles[i] > 3535 {
				return -1
			}
			return m.mapFiles[i]
		}
	}
	return -1
}

// ObjectFileID returns the cache file ID for object/landscape data at the given region.
// regionX and regionY are in region coordinates (tile / 64).
// Returns -1 if not found.
func (m *MapIndex) ObjectFileID(regionX, regionY int) int {
	id := (regionX << 8) | regionY
	for i, area := range m.areas {
		if area == id {
			if m.landscapes[i] > 3535 {
				return -1
			}
			return m.landscapes[i]
		}
	}
	return -1
}

// RegionEntry describes a map region available in the cache.
type RegionEntry struct {
	AreaID      int
	TerrainFile int
	ObjectFile  int
}

// Regions returns all available region entries.
func (m *MapIndex) Regions() []RegionEntry {
	entries := make([]RegionEntry, len(m.areas))
	for i := range m.areas {
		entries[i] = RegionEntry{
			AreaID:      m.areas[i],
			TerrainFile: m.mapFiles[i],
			ObjectFile:  m.landscapes[i],
		}
	}
	return entries
}
