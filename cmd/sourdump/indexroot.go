package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
)

// IndexRootCmd walks a directory, hashes every file, and writes a raw
// .index.source containing only assets and refs (no bundles/maps/mods).
// This is used to publish a game data checkout so that CI and clients
// can fetch individual files by hash without needing the full checkout.
type IndexRootCmd struct {
	Path   string `arg:"" help:"Path to the game data directory (e.g. assets/roots/base)."`
	Outdir string `help:"Output directory for the index and blobs." default:"output/"`
	Prefix string `help:"Prefix for the .index.source filename." default:""`
	Copy   bool   `help:"Copy asset files to outdir named by hash." default:"false"`
}

func (cmd *IndexRootCmd) Run() error {
	absPath, err := filepath.Abs(cmd.Path)
	if err != nil {
		return err
	}

	os.MkdirAll(cmd.Outdir, 0755)

	// Walk the directory and hash every file
	assetSet := make(map[string]struct{})
	var refs []assets.Asset

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(absPath, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes
		rel = strings.ReplaceAll(rel, string(os.PathSeparator), "/")

		hash, err := packager.HashFile(path)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to hash %s", rel)
			return nil
		}

		assetSet[hash] = struct{}{}
		refs = append(refs, assets.Asset{
			Path: rel,
			Id:   hash,
		})

		if cmd.Copy {
			dest := filepath.Join(cmd.Outdir, hash)
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.WriteFile(dest, data, 0644); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory: %w", err)
	}

	// Build sorted asset list
	assetList := make([]string, 0, len(assetSet))
	for id := range assetSet {
		assetList = append(assetList, id)
	}
	sort.Strings(assetList)

	// Build lookup for compact refs
	lookup := make(map[string]int)
	for i, id := range assetList {
		lookup[id] = i
	}

	indexRefs := make([]assets.IndexAsset, 0, len(refs))
	for _, ref := range refs {
		indexRefs = append(indexRefs, assets.IndexAsset{
			Id:   lookup[ref.Id],
			Path: ref.Path,
		})
	}

	index := assets.NewIndex()
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
