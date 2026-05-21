package packager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/dump"
	"github.com/cfoust/sour/pkg/min"
)

// DumpSour extracts file references for a map, model, or cfg.
// Returns Mapping pairs where From is "fs:/path" or "id:hash" and To is the game path.
func DumpSour(ctx context.Context, type_ string, target string, roots []assets.Root) ([]Mapping, error) {
	var references []min.Mapping
	var err error

	if type_ == "model" {
		references, err = dump.DumpModel(ctx, roots, target)
	} else {
		ref, resolveErr := dump.ResolveTarget(ctx, roots, target)
		if resolveErr != nil {
			return nil, resolveErr
		}

		switch type_ {
		case "map":
			references, err = dump.DumpMap(ctx, roots, ref, "")
		case "cfg":
			references, err = dump.DumpCFG(ctx, roots, ref, "")
		default:
			return nil, fmt.Errorf("invalid type %s", type_)
		}
	}

	if err != nil {
		return nil, err
	}

	references = min.CrunchReferences(references)

	var result []Mapping
	for _, ref := range references {
		resolved, err := ref.From.Resolve(ctx)
		if err != nil {
			continue
		}

		result = append(result, Mapping{
			From: resolved,
			To:   ref.To,
		})
	}

	return result, nil
}

// QueryFiles resolves file paths using roots, returning Mapping pairs.
func QueryFiles(ctx context.Context, roots []assets.Root, files []string) ([]Mapping, error) {
	results, err := dump.Query(ctx, roots, files)
	if err != nil {
		return nil, err
	}

	var mappings []Mapping
	for _, r := range results {
		mappings = append(mappings, Mapping{
			From: r.Resolved,
			To:   r.Target,
		})
	}

	return mappings, nil
}

// HashAssets computes a combined SHA-256 hash of assets resolved through roots.
func HashAssets(ctx context.Context, roots []assets.Root, targets []string) (string, error) {
	return dump.Hash(ctx, roots, targets)
}

// GetRootFiles returns all file paths available across roots.
func GetRootFiles(ctx context.Context, roots []assets.Root) ([]string, error) {
	var files []string

	for _, root := range roots {
		remoteRoot, ok := root.(*assets.PackagedRoot)
		if ok {
			for file := range remoteRoot.FS {
				files = append(files, file)
			}
			continue
		}

		fsRoot, ok := root.(assets.FSRoot)
		if ok {
			rootPath := string(fsRoot)
			err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				relative := path[len(rootPath)+1:]
				files = append(files, relative)
				return nil
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return files, nil
}

// GetMapFiles returns all files referenced by a map, as Mapping pairs.
func GetMapFiles(ctx context.Context, mapFile string, roots []assets.Root) ([]Mapping, error) {
	return DumpSour(ctx, "map", mapFile, roots)
}

// DownloadAssets downloads assets from remote roots to outdir.
func DownloadAssets(ctx context.Context, roots []assets.Root, outdir string, assetIDs []string) error {
	return dump.Download(ctx, roots, outdir, assetIDs)
}

// SearchFile searches for a file across roots and returns its resolved path.
func SearchFile(ctx context.Context, roots []assets.Root, file string) string {
	results, err := QueryFiles(ctx, roots, []string{file})
	if err != nil || len(results) == 0 {
		return ""
	}
	if results[0].From == "nil" {
		return ""
	}
	return results[0].From
}

// IsIDRef returns true if the reference is an asset ID (not a filesystem path).
func IsIDRef(ref string) bool {
	return strings.HasPrefix(ref, "id:")
}

// IsFSRef returns true if the reference is a filesystem path.
func IsFSRef(ref string) bool {
	return strings.HasPrefix(ref, "fs:")
}

// StripPrefix removes the "fs:" or "id:" prefix from a reference.
func StripPrefix(ref string) string {
	if strings.HasPrefix(ref, "fs:") {
		return ref[3:]
	}
	if strings.HasPrefix(ref, "id:") {
		return ref[3:]
	}
	return ref
}
