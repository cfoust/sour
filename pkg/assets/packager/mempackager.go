package packager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfoust/sour/pkg/assets"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
)

// MemPackager builds assets entirely in memory.
type MemPackager struct {
	Store    *MemStore
	Assets   map[string]struct{}
	Bundles  []assets.Bundle
	Maps     []assets.GameMap
	Models   []assets.Model
	Mods     []assets.Mod
	Textures []assets.Asset
	Refs     []assets.Asset
}

func NewMemPackager() *MemPackager {
	return &MemPackager{
		Store:    NewMemStore(),
		Assets:   make(map[string]struct{}),
		Bundles:  []assets.Bundle{},
		Maps:     []assets.GameMap{},
		Models:   []assets.Model{},
		Mods:     []assets.Mod{},
		Textures: []assets.Asset{},
		Refs:     []assets.Asset{},
	}
}

// BuildAsset reads a file, hashes it, stores in memory.
func (p *MemPackager) BuildAsset(params BuildParams, m Mapping) (*assets.Asset, error) {
	from, to := m.From, m.To

	if from == "nil" {
		return nil, nil
	}

	if IsIDRef(from) {
		id := from[3:]
		asset := &assets.Asset{Path: to, Id: id}
		if !p.Store.Has(id) {
			// Download from roots
			for _, root := range params.Roots {
				pr, ok := root.(*assets.PackagedRoot)
				if !ok {
					continue
				}
				data, err := pr.ReadAsset(context.Background(), id)
				if err == nil {
					p.Store.Set(id, data)
					break
				}
			}
		}
		p.Assets[id] = struct{}{}
		return asset, nil
	}

	fsPath := StripPrefix(from)

	info, err := os.Stat(fsPath)
	if err != nil || info.IsDir() {
		return nil, nil
	}

	data, err := os.ReadFile(fsPath)
	if err != nil {
		return nil, err
	}

	fileHash := HashBytes(data)
	asset := &assets.Asset{Path: to, Id: fileHash}

	if !p.Store.Has(fileHash) {
		p.Store.Set(fileHash, data)
	}
	p.Assets[fileHash] = struct{}{}
	return asset, nil
}

func (p *MemPackager) BuildAssets(params BuildParams, files []Mapping) ([]assets.Asset, error) {
	var result []assets.Asset
	for _, file := range files {
		asset, err := p.BuildAsset(params, file)
		if err != nil {
			return nil, err
		}
		if asset == nil {
			continue
		}
		result = append(result, *asset)
	}
	return result, nil
}

func (p *MemPackager) BuildBundle(ctx context.Context, params BuildParams, files []Mapping) (*assets.Bundle, error) {
	builtAssets, err := p.BuildAssets(params, files)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(builtAssets))
	for i, a := range builtAssets {
		ids[i] = a.Id
	}
	sort.Strings(ids)
	bundleID := HashString(strings.Join(ids, ""))

	bundle := assets.Bundle{
		Id:      bundleID,
		Assets:  builtAssets,
		Desktop: false,
		Web:     true,
	}

	// Build .sour bundle in memory
	sourData, err := BuildSourBundleBytes(p.Store, bundle)
	if err != nil {
		return nil, fmt.Errorf("building sour bundle: %w", err)
	}
	p.Store.Set(bundleID+".sour", sourData)

	p.Bundles = append(p.Bundles, bundle)
	return &bundle, nil
}

func (p *MemPackager) BuildMap(ctx context.Context, params BuildParams, mapFile, name, description string) (*assets.GameMap, error) {
	mapFiles, err := DumpSour(ctx, "map", mapFile, params.Roots)
	if err != nil {
		return nil, err
	}

	builtAssets, err := p.BuildAssets(params, mapFiles)
	if err != nil {
		return nil, err
	}

	base := strings.TrimSuffix(mapFile, filepath.Ext(mapFile))
	mapHashFiles := []string{mapFile, base + ".cfg"}
	mapHash, err := HashAssets(ctx, params.Roots, mapHashFiles)
	if err != nil {
		return nil, err
	}

	var ogzID string
	for _, asset := range builtAssets {
		if strings.HasSuffix(asset.Path, ".ogz") {
			ogzID = asset.Id
		}
	}
	if ogzID == "" {
		return nil, nil
	}

	// Find mapshot
	var image string
	for _, ext := range []string{".png", ".jpg"} {
		imagePath := base + ext
		resolved, err := QueryFiles(ctx, params.Roots, []string{imagePath})
		if err != nil || len(resolved) == 0 || resolved[0].From == "nil" {
			continue
		}
		imgAsset, err := p.BuildAsset(params, resolved[0])
		if err != nil || imgAsset == nil {
			continue
		}
		image = imgAsset.Id + ext
		// Copy the data under the image filename too
		if data, err := p.Store.Get(imgAsset.Id); err == nil {
			p.Store.Set(image, data)
		}
		break
	}

	bundle, err := p.BuildBundle(ctx, params, mapFiles)
	if err != nil {
		return nil, fmt.Errorf("building map bundle: %w", err)
	}

	gameMap := assets.GameMap{
		Id:          mapHash,
		Name:        name,
		Bundle:      bundle.Id,
		Ogz:         ogzID,
		Assets:      builtAssets,
		Image:       image,
		Description: description,
	}

	p.Maps = append(p.Maps, gameMap)
	return &gameMap, nil
}

// DumpIndex returns the CBOR index as bytes.
func (p *MemPackager) DumpIndex() ([]byte, error) {
	assetList := make([]string, 0, len(p.Assets))
	for id := range p.Assets {
		assetList = append(assetList, id)
	}
	sort.Strings(assetList)

	lookup := make(map[string]int)
	for i, id := range assetList {
		lookup[id] = i
	}

	replaceAsset := func(a assets.Asset) assets.IndexAsset {
		return assets.IndexAsset{
			Id:   lookup[a.Id],
			Path: a.Path,
		}
	}

	refs := make([]assets.IndexAsset, 0, len(p.Refs))
	for _, ref := range p.Refs {
		refs = append(refs, replaceAsset(ref))
	}

	index := assets.NewIndex()
	index.Assets = assetList
	index.Textures = p.Textures
	index.Refs = refs
	index.Bundles = p.Bundles
	index.Models = p.Models
	index.Maps = p.Maps
	index.Mods = p.Mods

	return cbor.Marshal(index)
}

// Merge incorporates results from another MemPackager.
func (p *MemPackager) Merge(other *MemPackager) {
	for id := range other.Assets {
		p.Assets[id] = struct{}{}
	}
	// Merge store data
	for _, key := range other.Store.Keys() {
		if !p.Store.Has(key) {
			data, _ := other.Store.Get(key)
			p.Store.Set(key, data)
		}
	}
	p.Bundles = append(p.Bundles, other.Bundles...)
	p.Maps = append(p.Maps, other.Maps...)
}

// BuildSourBundleBytes creates a .sour bundle entirely in memory.
func BuildSourBundleBytes(store *MemStore, bundle assets.Bundle) ([]byte, error) {
	var filePaths []string
	for _, asset := range bundle.Assets {
		filePaths = append(filePaths, asset.Path)
	}

	directories := buildDirectoryList(filePaths)

	var dataChunks [][]byte
	var files []sourFileEntry
	offset := 0

	for _, asset := range bundle.Assets {
		data, err := store.Get(asset.Id)
		if err != nil {
			return nil, fmt.Errorf("asset %s not in store: %w", asset.Id, err)
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

	return encodeSourBundle(directories, metadata, dataChunks)
}

// BuildMapDirs scans directories for .ogz files and builds them in memory.
// extraRoots are additional asset root strings (e.g. existing asset sources)
// that provide base game files like default_map_settings.cfg.
func BuildMapDirs(ctx context.Context, dirs []string, extraRoots ...string) (*MemPackager, error) {
	p := NewMemPackager()

	// Collect all .ogz files
	type mapEntry struct {
		path string
		root string
	}
	var maps []mapEntry

	for _, dir := range dirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to resolve map dir: %s", dir)
			continue
		}

		err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(path, ".ogz") {
				maps = append(maps, mapEntry{path: path, root: absDir})
			}
			return nil
		})
		if err != nil {
			log.Warn().Err(err).Msgf("failed to walk map dir: %s", dir)
		}
	}

	if len(maps) == 0 {
		return p, nil
	}

	log.Info().Msgf("building %d maps from %d directories", len(maps), len(dirs))

	// Build roots: map dirs + any extra roots (for base game cfgs/textures)
	cache := assets.FSStore("/tmp/sour-mapdirs-cache")
	os.MkdirAll("/tmp/sour-mapdirs-cache", 0755)

	var rootStrings []string
	for _, dir := range dirs {
		absDir, _ := filepath.Abs(dir)
		rootStrings = append(rootStrings, absDir)
	}
	rootStrings = append(rootStrings, extraRoots...)

	// Check if any root provides default_map_settings.cfg. If not, add the
	// canonical Sauerbraten base game as a remote root so that map building
	// can resolve texture slots and other base game resources.
	const baseGameRoot = "https://static.sourga.me/blobs/6481/.index.source"
	needsBase := true
	for _, rs := range rootStrings {
		if rs == baseGameRoot {
			needsBase = false
			break
		}
	}
	if needsBase {
		// Quick check: can any existing root resolve default_map_settings.cfg?
		testRoots, _ := assets.LoadRoots(ctx, cache, rootStrings, false)
		found := false
		for _, r := range testRoots {
			if r.Exists(ctx, "data/default_map_settings.cfg") {
				found = true
				break
			}
		}
		if !found {
			log.Info().Msg("mapDirs: fetching base game files from static.sourga.me (needed for texture/model resolution)")
			rootStrings = append(rootStrings, baseGameRoot)
		}
	}

	roots, err := assets.LoadRoots(ctx, cache, rootStrings, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load roots: %w", err)
	}

	params := BuildParams{
		Roots:       roots,
		RootStrings: rootStrings,
	}

	for _, m := range maps {
		relPath, err := filepath.Rel(m.root, m.path)
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(filepath.Base(m.path), ".ogz")

		sub := NewMemPackager()
		_, err = sub.BuildMap(ctx, params, relPath, name, "")
		if err != nil {
			log.Warn().Err(err).Msgf("failed to build map %s", name)
			continue
		}

		p.Merge(sub)
		log.Info().Msgf("built map %s", name)
	}

	log.Info().Msgf("built %d maps in memory (%d MB)", len(p.Maps), p.Store.Size()/(1024*1024))

	return p, nil
}
