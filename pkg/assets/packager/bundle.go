package packager

import (
	"archive/zip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
)

// sourFileEntry represents a file in the .sour metadata JSON.
type sourFileEntry struct {
	Filename string `json:"filename"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
}

// sourMetadata is the metadata JSON in the .sour format.
type sourMetadata struct {
	Files             []sourFileEntry `json:"files"`
	RemotePackageSize int             `json:"remote_package_size"`
}

// buildDirectoryList returns the directory creation entries needed for a set of file paths.
// Each entry is ["parent", "dirname", true] matching the Emscripten file_packager format.
func buildDirectoryList(paths []string) [][]interface{} {
	seen := make(map[string]bool)
	var result [][]interface{}

	for _, p := range paths {
		dir := path.Dir(p)
		if dir == "." || dir == "" {
			continue
		}

		// Walk up the directory tree
		parts := strings.Split(dir, "/")
		for i := 0; i < len(parts); i++ {
			fullDir := strings.Join(parts[:i+1], "/")
			if seen[fullDir] {
				continue
			}
			seen[fullDir] = true

			parent := ""
			if i > 0 {
				parent = strings.Join(parts[:i], "/")
			}

			result = append(result, []interface{}{parent, parts[i], true})
		}
	}

	// Sort for deterministic output
	sort.Slice(result, func(i, j int) bool {
		a := fmt.Sprintf("%s/%s", result[i][0], result[i][1])
		b := fmt.Sprintf("%s/%s", result[j][0], result[j][1])
		return a < b
	})

	return result
}

// BuildSourBundle creates a .sour web bundle from assets.
// The format matches what client/src/assets/worker.ts:unpackBundle() expects:
//   - 4 bytes big-endian: directories JSON length
//   - directories JSON
//   - 4 bytes big-endian: metadata JSON length
//   - metadata JSON with {files: [{filename, start, end}...], remote_package_size}
//   - concatenated raw file data
func BuildSourBundle(outdir string, bundle assets.Bundle) (string, error) {
	target := filepath.Join(outdir, fmt.Sprintf("%s.sour", bundle.Id))

	if fileExists(target) {
		return bundle.Id, nil
	}

	// Collect file paths for directory list
	var filePaths []string
	for _, asset := range bundle.Assets {
		filePaths = append(filePaths, asset.Path)
	}

	directories := buildDirectoryList(filePaths)

	// Build the concatenated data and file entries
	var dataChunks [][]byte
	var files []sourFileEntry
	offset := 0

	for _, asset := range bundle.Assets {
		srcPath := filepath.Join(outdir, asset.Id)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("reading asset %s: %w", asset.Id, err)
		}

		files = append(files, sourFileEntry{
			Filename: asset.Path,
			Start:    offset,
			End:      offset + len(data),
		})

		dataChunks = append(dataChunks, data)
		offset += len(data)
	}

	metadata := sourMetadata{
		Files:             files,
		RemotePackageSize: offset,
	}

	dirJSON, err := json.Marshal(directories)
	if err != nil {
		return "", fmt.Errorf("marshaling directories: %w", err)
	}

	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshaling metadata: %w", err)
	}

	// Write the .sour file
	out, err := os.Create(target)
	if err != nil {
		return "", fmt.Errorf("creating bundle file: %w", err)
	}
	defer out.Close()

	// Directory JSON length + data
	if err := binary.Write(out, binary.BigEndian, int32(len(dirJSON))); err != nil {
		return "", err
	}
	if _, err := out.Write(dirJSON); err != nil {
		return "", err
	}

	// Metadata JSON length + data
	if err := binary.Write(out, binary.BigEndian, int32(len(metaJSON))); err != nil {
		return "", err
	}
	if _, err := out.Write(metaJSON); err != nil {
		return "", err
	}

	// Raw concatenated data
	for _, chunk := range dataChunks {
		if _, err := out.Write(chunk); err != nil {
			return "", err
		}
	}

	return bundle.Id, nil
}

// BuildDesktopBundle creates a .desktop ZIP bundle.
func BuildDesktopBundle(outdir string, bundle assets.Bundle) error {
	zipPath := filepath.Join(outdir, fmt.Sprintf("%s.desktop", bundle.Id))

	if fileExists(zipPath) {
		return nil
	}

	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	added := make(map[string]bool)

	for _, asset := range bundle.Assets {
		if added[asset.Path] {
			continue
		}

		srcPath := filepath.Join(outdir, asset.Id)
		src, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("opening asset %s: %w", asset.Id, err)
		}

		header := &zip.FileHeader{
			Name:   asset.Path,
			Method: zip.Deflate,
		}

		dst, err := w.CreateHeader(header)
		if err != nil {
			src.Close()
			return err
		}

		_, err = io.Copy(dst, src)
		src.Close()
		if err != nil {
			return err
		}

		added[asset.Path] = true
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
