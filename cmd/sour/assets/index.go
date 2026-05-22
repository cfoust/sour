package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	pkgassets "github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
)

// IndexCmd indexes a directory into a raw .index.source.
type IndexCmd struct {
	Path   string `arg:"" help:"Path to directory to index."`
	Outdir string `help:"Output directory." default:"output/"`
	Prefix string `help:"Prefix for .index.source filename." default:""`
	Copy   bool   `help:"Copy files to outdir named by hash."`
}

func (cmd *IndexCmd) Run() error {
	absPath, err := filepath.Abs(cmd.Path)
	if err != nil {
		return err
	}
	os.MkdirAll(cmd.Outdir, 0755)

	assetSet := make(map[string]struct{})
	var refs []pkgassets.Asset

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(absPath, path)
		if err != nil {
			return err
		}
		rel = strings.ReplaceAll(rel, string(os.PathSeparator), "/")

		hash, err := packager.HashFile(path)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to hash %s", rel)
			return nil
		}
		assetSet[hash] = struct{}{}
		refs = append(refs, pkgassets.Asset{Path: rel, Id: hash})

		if cmd.Copy {
			dest := filepath.Join(cmd.Outdir, hash)
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				return os.WriteFile(dest, data, 0644)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	assetList := make([]string, 0, len(assetSet))
	for id := range assetSet {
		assetList = append(assetList, id)
	}
	sort.Strings(assetList)

	lookup := make(map[string]int)
	for i, id := range assetList {
		lookup[id] = i
	}

	indexRefs := make([]pkgassets.IndexAsset, 0, len(refs))
	for _, ref := range refs {
		indexRefs = append(indexRefs, pkgassets.IndexAsset{Id: lookup[ref.Id], Path: ref.Path})
	}

	index := pkgassets.NewIndex()
	index.Assets = assetList
	index.Refs = indexRefs

	data, err := cbor.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	indexFile := fmt.Sprintf("%s.index.source", cmd.Prefix)
	outPath := filepath.Join(cmd.Outdir, indexFile)
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return err
	}

	log.Info().Msgf("indexed %d files (%d unique hashes) to %s", len(refs), len(assetList), outPath)
	return nil
}
