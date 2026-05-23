package osrs

import (
	"bytes"
	"compress/bzip2"
	"fmt"
	"io"
	"strings"
)

// Archive represents a decoded OSRS FileArchive containing named files.
type Archive struct {
	buffer     []byte
	entries    int
	ids        []int
	extracted  []int
	compressed []int
	offsets    []int
	wholeBuf   bool // true if the archive was decompressed as a whole
}

// DecodeArchive decodes a FileArchive from raw data read from the cache.
func DecodeArchive(data []byte) (*Archive, error) {
	buf := NewBuffer(data)

	decompressedLen, err := buf.ReadTriByte()
	if err != nil {
		return nil, fmt.Errorf("reading decompressed length: %w", err)
	}
	compressedLen, err := buf.ReadTriByte()
	if err != nil {
		return nil, fmt.Errorf("reading compressed length: %w", err)
	}

	var archiveData []byte
	wholeBuf := false

	if compressedLen != decompressedLen {
		// Entire archive is BZip2-compressed
		decompressed, err := decompressBzip2(data[6:6+compressedLen], decompressedLen)
		if err != nil {
			return nil, fmt.Errorf("decompressing archive: %w", err)
		}
		archiveData = decompressed
		buf = NewBuffer(archiveData)
		wholeBuf = true
	} else {
		archiveData = data
	}

	entries, err := buf.ReadUShort()
	if err != nil {
		return nil, fmt.Errorf("reading entry count: %w", err)
	}

	ids := make([]int, entries)
	extracted := make([]int, entries)
	compressed := make([]int, entries)
	offsets := make([]int, entries)

	offset := buf.Position() + entries*10
	for i := 0; i < entries; i++ {
		id, err := buf.ReadInt()
		if err != nil {
			return nil, err
		}
		ids[i] = id

		ext, err := buf.ReadTriByte()
		if err != nil {
			return nil, err
		}
		extracted[i] = ext

		comp, err := buf.ReadTriByte()
		if err != nil {
			return nil, err
		}
		compressed[i] = comp

		offsets[i] = offset
		offset += comp
	}

	return &Archive{
		buffer:     archiveData,
		entries:    entries,
		ids:        ids,
		extracted:  extracted,
		compressed: compressed,
		offsets:    offsets,
		wholeBuf:   wholeBuf,
	}, nil
}

// ReadFile reads a named file from the archive.
func (a *Archive) ReadFile(name string) ([]byte, error) {
	hash := hashName(name)

	for i := 0; i < a.entries; i++ {
		// Compare as int32 to match Java's signed int behavior
		if int32(a.ids[i]) != int32(hash) {
			continue
		}

		output := make([]byte, a.extracted[i])
		if !a.wholeBuf {
			// Individual entry is BZip2-compressed
			decompressed, err := decompressBzip2(
				a.buffer[a.offsets[i]:a.offsets[i]+a.compressed[i]],
				a.extracted[i],
			)
			if err != nil {
				return nil, fmt.Errorf("decompressing entry %s: %w", name, err)
			}
			return decompressed, nil
		}
		// Archive was decompressed as a whole, just copy
		copy(output, a.buffer[a.offsets[i]:a.offsets[i]+a.extracted[i]])
		return output, nil
	}

	return nil, fmt.Errorf("file %q not found in archive", name)
}

// hashName computes the OSRS name hash: (hash * 61 + char) - 32 for each char.
// Uses int32 arithmetic to match Java's 32-bit int overflow behavior.
func hashName(name string) int {
	name = strings.ToUpper(name)
	hash := int32(0)
	for _, c := range name {
		hash = (hash*61 + int32(c)) - 32
	}
	return int(hash)
}

// decompressBzip2 decompresses OSRS BZip2 data.
// OSRS BZip2 data lacks the standard "BZ" magic header, so we prepend it.
func decompressBzip2(data []byte, expectedSize int) ([]byte, error) {
	// Prepend the standard BZip2 header "BZh1" (block size 1)
	header := []byte("BZh1")
	padded := make([]byte, len(header)+len(data))
	copy(padded, header)
	copy(padded[len(header):], data)

	reader := bzip2.NewReader(bytes.NewReader(padded))
	output := make([]byte, expectedSize)
	_, err := io.ReadFull(reader, output)
	if err != nil {
		return nil, err
	}
	return output, nil
}
