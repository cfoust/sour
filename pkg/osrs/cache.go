package osrs

import (
	"compress/gzip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	sectorSize    = 520
	sectorPayload = 512
	sectorHeader  = 8
	indexEntrySize = 6
	numIndices     = 6
)

// Cache reads files from the OSRS file store (main_file_cache.dat + .idx files).
type Cache struct {
	dataFile *os.File
	indices  [numIndices]*os.File
}

// OpenCache opens an OSRS cache from a directory containing
// main_file_cache.dat and main_file_cache.idx0 through .idx5.
func OpenCache(dir string) (*Cache, error) {
	dataPath := filepath.Join(dir, "main_file_cache.dat")
	dataFile, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("opening data file: %w", err)
	}

	c := &Cache{dataFile: dataFile}

	for i := 0; i < numIndices; i++ {
		idxPath := filepath.Join(dir, fmt.Sprintf("main_file_cache.idx%d", i))
		f, err := os.Open(idxPath)
		if err != nil {
			// Not all indices may exist
			continue
		}
		c.indices[i] = f
	}

	return c, nil
}

func (c *Cache) Close() {
	c.dataFile.Close()
	for _, f := range c.indices {
		if f != nil {
			f.Close()
		}
	}
}

// ReadFile reads a file from the given store index (0-based) by file ID.
func (c *Cache) ReadFile(storeIndex, fileID int) ([]byte, error) {
	if storeIndex < 0 || storeIndex >= numIndices || c.indices[storeIndex] == nil {
		return nil, fmt.Errorf("invalid store index %d", storeIndex)
	}

	// The OSRS cache uses 1-based store IDs in sector headers.
	// Index file 0 → store byte 1, index file 4 → store byte 5, etc.
	sectorStoreID := storeIndex + 1

	idxFile := c.indices[storeIndex]

	// Read 6-byte index entry
	entryBuf := make([]byte, indexEntrySize)
	_, err := idxFile.ReadAt(entryBuf, int64(fileID)*indexEntrySize)
	if err != nil {
		return nil, fmt.Errorf("reading index entry %d: %w", fileID, err)
	}

	size := (int(entryBuf[0]&0xff) << 16) + (int(entryBuf[1]&0xff) << 8) + int(entryBuf[2]&0xff)
	sector := (int(entryBuf[3]&0xff) << 16) + (int(entryBuf[4]&0xff) << 8) + int(entryBuf[5]&0xff)

	if sector <= 0 || size <= 0 {
		return nil, fmt.Errorf("invalid index entry: size=%d sector=%d", size, sector)
	}

	buf := make([]byte, size)
	sectorBuf := make([]byte, sectorSize)
	totalRead := 0

	for part := 0; totalRead < size; part++ {
		if sector == 0 {
			return nil, fmt.Errorf("unexpected end of sector chain at part %d", part)
		}

		_, err := c.dataFile.ReadAt(sectorBuf, int64(sector)*sectorSize)
		if err != nil {
			return nil, fmt.Errorf("reading sector %d: %w", sector, err)
		}

		// Parse sector header
		currentID := (int(sectorBuf[0]&0xff) << 8) + int(sectorBuf[1]&0xff)
		currentPart := (int(sectorBuf[2]&0xff) << 8) + int(sectorBuf[3]&0xff)
		nextSector := (int(sectorBuf[4]&0xff) << 16) + (int(sectorBuf[5]&0xff) << 8) + int(sectorBuf[6]&0xff)
		currentStore := int(sectorBuf[7]) & 0xff

		if currentID != fileID || currentPart != part || currentStore != sectorStoreID {
			return nil, fmt.Errorf("sector mismatch: id=%d/%d part=%d/%d store=%d/%d",
				currentID, fileID, currentPart, part, currentStore, sectorStoreID)
		}

		unread := size - totalRead
		if unread > sectorPayload {
			unread = sectorPayload
		}

		copy(buf[totalRead:totalRead+unread], sectorBuf[sectorHeader:sectorHeader+unread])
		totalRead += unread
		sector = nextSector
	}

	return buf, nil
}

// ReadFileGzip reads a gzip-compressed file from the store and decompresses it.
func (c *Cache) ReadFileGzip(storeIndex, fileID int) ([]byte, error) {
	data, err := c.ReadFile(storeIndex, fileID)
	if err != nil {
		return nil, err
	}
	return DecompressGzip(data)
}

// DecompressGzip decompresses gzip data.
func DecompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer r.Close()
	return io.ReadAll(r)
}

// IndexEntryCount returns the number of file entries in the given index.
func (c *Cache) IndexEntryCount(storeIndex int) (int, error) {
	if storeIndex < 0 || storeIndex >= numIndices || c.indices[storeIndex] == nil {
		return 0, fmt.Errorf("invalid store index %d", storeIndex)
	}
	info, err := c.indices[storeIndex].Stat()
	if err != nil {
		return 0, err
	}
	return int(info.Size()) / indexEntrySize, nil
}
