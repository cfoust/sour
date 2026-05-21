package packager

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfoust/sour/pkg/assets"

	"github.com/fxamacker/cbor/v2"
	"github.com/rs/zerolog/log"
)

// Packager builds and tracks assets, bundles, maps, models, and mods.
type Packager struct {
	Outdir   string
	Assets   map[string]struct{}
	Refs     []assets.Asset
	Bundles  []assets.Bundle
	Maps     []assets.GameMap
	Models   []assets.Model
	Mods     []assets.Mod
	Textures []assets.Asset
}

// New creates a new Packager writing to outdir.
func New(outdir string) *Packager {
	return &Packager{
		Outdir: outdir,
		Assets: make(map[string]struct{}),
	}
}

// BuildAsset processes a single file mapping into a hashed asset.
func (p *Packager) BuildAsset(params BuildParams, m Mapping) (*assets.Asset, error) {
	from, to := m.From, m.To

	if from == "nil" {
		return nil, nil
	}

	if IsIDRef(from) {
		id := from[3:]
		asset := &assets.Asset{Path: to, Id: id}
		if params.DownloadAssets {
			ctx := context.Background()
			if err := DownloadAssets(ctx, params.Roots, p.Outdir, []string{id}); err != nil {
				return nil, err
			}
			p.Assets[asset.Id] = struct{}{}
		}
		return asset, nil
	}

	// Strip fs: prefix
	fsPath := StripPrefix(from)

	info, err := os.Stat(fsPath)
	if err != nil || info.IsDir() {
		return nil, nil
	}

	fileHash, err := HashFile(fsPath)
	if err != nil {
		return nil, err
	}

	outFile := filepath.Join(p.Outdir, fileHash)
	asset := &assets.Asset{Path: to, Id: fileHash}

	if fileExists(outFile) {
		p.Assets[asset.Id] = struct{}{}
		return asset, nil
	}

	ext := strings.ToLower(filepath.Ext(fsPath))

	// Check if we should compress this image
	if params.CompressImages && IsImageCompressible(fsPath, minCompressSize) {
		workDir := "working"
		os.MkdirAll(workDir, 0755)

		compressed := filepath.Join(workDir, fileHash+ext)

		if fileExists(compressed) {
			copyFile(compressed, outFile)
			p.Assets[asset.Id] = struct{}{}
			return asset, nil
		}

		if err := CompressImage(fsPath, compressed); err != nil {
			// If compression fails, fall through to direct copy
			log.Warn().Err(err).Msgf("failed to compress %s, using original", fsPath)
		} else if fileExists(compressed) {
			copyFile(compressed, outFile)
			p.Assets[asset.Id] = struct{}{}
			return asset, nil
		}
	}

	if err := copyFile(fsPath, outFile); err != nil {
		return nil, err
	}
	p.Assets[asset.Id] = struct{}{}
	return asset, nil
}

// BuildRef builds an asset and adds it to the refs list.
func (p *Packager) BuildRef(params BuildParams, m Mapping) (*assets.Asset, error) {
	asset, err := p.BuildAsset(params, m)
	if err != nil {
		return nil, err
	}
	if asset == nil {
		return nil, nil
	}
	p.Refs = append(p.Refs, *asset)
	return asset, nil
}

// BuildAssets processes multiple file mappings into assets.
func (p *Packager) BuildAssets(params BuildParams, files []Mapping) ([]assets.Asset, error) {
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

// BuildBundle creates a bundle from a set of file mappings.
func (p *Packager) BuildBundle(ctx context.Context, params BuildParams, files []Mapping) (*assets.Bundle, error) {
	builtAssets, err := p.BuildAssets(params, files)
	if err != nil {
		return nil, err
	}

	// Generate deterministic bundle ID from sorted asset IDs
	ids := make([]string, len(builtAssets))
	for i, a := range builtAssets {
		ids[i] = a.Id
	}
	sort.Strings(ids)
	bundleID := HashString(strings.Join(ids, ""))

	bundle := assets.Bundle{
		Id:      bundleID,
		Assets:  builtAssets,
		Desktop: params.BuildDesktop,
		Web:     params.BuildWeb,
	}

	if params.BuildWeb {
		if _, err := BuildSourBundle(p.Outdir, bundle); err != nil {
			return nil, fmt.Errorf("building sour bundle: %w", err)
		}
	}

	if params.BuildDesktop {
		desktopBundle := bundle
		if params.SkipRoot != "" {
			// Filter out assets that exist in the skip root
			skipResults, err := QueryFiles(ctx, params.Roots, assetPaths(builtAssets))
			if err == nil {
				var newAssets []assets.Asset
				for i, result := range skipResults {
					if result.From == "nil" {
						newAssets = append(newAssets, builtAssets[i])
					}
				}
				desktopBundle.Assets = newAssets
			}
		}
		if err := BuildDesktopBundle(p.Outdir, desktopBundle); err != nil {
			return nil, fmt.Errorf("building desktop bundle: %w", err)
		}
	}

	p.Bundles = append(p.Bundles, bundle)
	return &bundle, nil
}

// BuildMod creates a mod bundle.
func (p *Packager) BuildMod(ctx context.Context, params BuildParams, files []Mapping, name, description string, image string) (*assets.Mod, error) {
	bundle, err := p.BuildBundle(ctx, params, files)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, nil
	}

	mod := assets.Mod{
		Id:          bundle.Id,
		Name:        name,
		Image:       image,
		Description: description,
	}

	p.Mods = append(p.Mods, mod)
	return &mod, nil
}

// BuildModel creates a model by extracting its files and bundling them.
func (p *Packager) BuildModel(ctx context.Context, params BuildParams, name string) (*assets.Model, error) {
	modelFiles, err := DumpSour(ctx, "model", name, params.Roots)
	if err != nil {
		return nil, err
	}

	bundle, err := p.BuildBundle(ctx, params, modelFiles)
	if err != nil {
		return nil, err
	}
	if bundle == nil {
		return nil, fmt.Errorf("failed to build bundle for model %s", name)
	}

	model := assets.Model{
		Id:   bundle.Id,
		Name: name,
	}

	p.Models = append(p.Models, model)
	return &model, nil
}

// BuildTexture builds a single texture asset.
func (p *Packager) BuildTexture(ctx context.Context, params BuildParams, file string) (*assets.Asset, error) {
	resolved, err := QueryFiles(ctx, params.Roots, []string{file})
	if err != nil {
		return nil, err
	}
	result, err := p.BuildAssets(params, resolved)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	texture := result[0]
	p.Textures = append(p.Textures, texture)
	return &texture, nil
}

// BuildTextures builds textures in batches.
func (p *Packager) BuildTextures(ctx context.Context, params BuildParams, files []string) error {
	batch := 500
	for i := 0; i < len(files); i += batch {
		end := i + batch
		if end > len(files) {
			end = len(files)
		}
		sub := files[i:end]

		resolved, err := QueryFiles(ctx, params.Roots, sub)
		if err != nil {
			return err
		}
		result, err := p.BuildAssets(params, resolved)
		if err != nil {
			return err
		}
		p.Textures = append(p.Textures, result...)
	}

	return nil
}

// BuildImage builds an image asset and returns "hash.ext" or empty string.
func (p *Packager) BuildImage(ctx context.Context, params BuildParams, file string) (string, error) {
	ext := path.Ext(file)
	resolved, err := QueryFiles(ctx, params.Roots, []string{file})
	if err != nil {
		return "", err
	}
	if len(resolved) == 0 || resolved[0].From == "nil" {
		return "", nil
	}

	imgParams := params
	imgParams.DownloadAssets = true

	asset, err := p.BuildAsset(imgParams, resolved[0])
	if err != nil {
		return "", err
	}
	if asset == nil {
		return "", nil
	}

	image := asset.Id + ext
	src := filepath.Join(p.Outdir, asset.Id)
	dst := filepath.Join(p.Outdir, image)
	if err := copyFile(src, dst); err != nil {
		return "", err
	}

	return image, nil
}

// BuildMap creates a map bundle with all its dependencies.
func (p *Packager) BuildMap(ctx context.Context, params BuildParams, mapFile, name, description string, image string) (*assets.GameMap, error) {
	mapFiles, err := DumpSour(ctx, "map", mapFile, params.Roots)
	if err != nil {
		return nil, err
	}

	builtAssets, err := p.BuildAssets(params, mapFiles)
	if err != nil {
		return nil, err
	}

	// Hash the .ogz and .cfg together for the map ID
	base := strings.TrimSuffix(mapFile, filepath.Ext(mapFile))
	mapHashFiles := []string{mapFile, base + ".cfg"}
	mapHash, err := HashAssets(ctx, params.Roots, mapHashFiles)
	if err != nil {
		return nil, err
	}

	// Find the map image if not provided
	if image == "" {
		for _, ext := range []string{".png", ".jpg"} {
			imagePath := base + ext
			img, err := p.BuildImage(ctx, params, imagePath)
			if err == nil && img != "" {
				image = img
				break
			}
		}
	}

	// Find the .ogz asset ID
	var ogzID string
	for _, asset := range builtAssets {
		if strings.HasSuffix(asset.Path, ".ogz") {
			ogzID = asset.Id
		}
	}

	if ogzID == "" {
		return nil, nil
	}

	bundle, err := p.BuildBundle(ctx, params, mapFiles)
	if err != nil {
		return nil, fmt.Errorf("building map bundle: %w", err)
	}
	if bundle == nil {
		return nil, fmt.Errorf("built bundle was missing")
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

// DumpIndex serializes the packager state to a CBOR .index.source file.
func (p *Packager) DumpIndex(prefix string) error {
	indexFile := fmt.Sprintf("%s.index.source", prefix)

	assetList := make([]string, 0, len(p.Assets))
	for id := range p.Assets {
		assetList = append(assetList, id)
	}
	sort.Strings(assetList)

	// Build lookup: asset ID -> integer index
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

	index := assets.Index{
		Assets:   assetList,
		Textures: p.Textures,
		Refs:     refs,
		Bundles:  p.Bundles,
		Models:   p.Models,
		Maps:     p.Maps,
		Mods:     p.Mods,
	}

	data, err := cbor.Marshal(index)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	outPath := filepath.Join(p.Outdir, indexFile)
	return os.WriteFile(outPath, data, 0644)
}

// Merge incorporates results from another packager (used for parallel building).
func (p *Packager) Merge(other *Packager) {
	for id := range other.Assets {
		p.Assets[id] = struct{}{}
	}
	p.Bundles = append(p.Bundles, other.Bundles...)
	p.Maps = append(p.Maps, other.Maps...)
	p.Models = append(p.Models, other.Models...)
	p.Mods = append(p.Mods, other.Mods...)
	p.Textures = append(p.Textures, other.Textures...)
	p.Refs = append(p.Refs, other.Refs...)
}

func assetPaths(a []assets.Asset) []string {
	paths := make([]string, len(a))
	for i, asset := range a {
		paths[i] = asset.Path
	}
	return paths
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
