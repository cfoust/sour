package dump

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game/constants"
	V "github.com/cfoust/sour/pkg/game/variables"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog/log"
)

// DumpMap parses a .ogz map file and returns all referenced assets.
func DumpMap(ctx context.Context, roots []assets.Root, ref *min.Reference, indexPath string) ([]min.Mapping, error) {
	extension := filepath.Ext(ref.Path)

	if extension != ".ogz" {
		return nil, fmt.Errorf("map must end in .ogz")
	}

	data, err := ref.ReadFile(ctx)
	if err != nil {
		return nil, err
	}

	_map, err := maps.FromGZ(data)
	if err != nil {
		return nil, fmt.Errorf("FromGZ(%s, %d bytes): %w", ref.Path, len(data), err)
	}

	processor := min.NewProcessor(roots, _map.VSlots)

	references := make([]min.Mapping, 0)

	addFile := func(ref *min.Reference) {
		references = append(references, min.Mapping{
			From: ref,
			To:   ref.Path,
		})
	}

	addMapFile := func(ref *min.Reference) {
		if !ref.Exists(ctx) {
			return
		}

		reference := min.Mapping{}
		reference.From = ref
		reference.To = fmt.Sprintf("packages/base/%s", filepath.Base(ref.Path))
		references = append(references, reference)
	}

	addMapFile(ref)

	if skybox, ok := _map.Vars["skybox"]; ok {
		value := string(skybox.(V.StringVariable))
		for _, path := range processor.FindCubemap(ctx, min.NormalizeTexture(value)) {
			addFile(path)
		}
	}

	if cloudlayer, ok := _map.Vars["cloudlayer"]; ok {
		value := string(cloudlayer.(V.StringVariable))
		resolved := processor.FindTexture(ctx, min.NormalizeTexture(value))

		if resolved != nil {
			addFile(resolved)
		}
	}

	if cloudbox, ok := _map.Vars["cloudbox"]; ok {
		value := string(cloudbox.(V.StringVariable))
		for _, path := range processor.FindCubemap(ctx, min.NormalizeTexture(value)) {
			addFile(path)
		}
	}

	modelRefs := make(map[int16]int)
	for _, entity := range _map.Entities {
		if entity.Type != maps.ET_MAPMODEL {
			continue
		}

		modelRefs[entity.Attr2] += 1
	}

	defaultPath := processor.SearchFile(ctx, "data/default_map_settings.cfg")
	if defaultPath == nil {
		return nil, fmt.Errorf("root with data/default_map_settings.cfg not provided")
	}

	err = processor.ProcessFile(ctx, defaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to process default map settings: %w", err)
	}

	cfg := min.ReplaceExtension(ref, "cfg")
	if cfg.Exists(ctx) {
		err = processor.ProcessFile(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to process map cfg: %w", err)
		}

		addMapFile(cfg)
	}

	for _, extension := range []string{"png", "jpg"} {
		shotName := min.ReplaceExtension(ref, extension)
		addMapFile(shotName)
	}

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(ctx, path.Name)
			if texture != nil {
				addFile(texture)
			}
		}
	}

	for _, file := range processor.Files {
		addFile(file)
	}

	for _, sound := range processor.Sounds {
		addFile(sound)
	}

	for i, model := range processor.Models {
		if _, ok := modelRefs[int16(i)]; ok {
			name := model.Name
			if name == "" {
				continue
			}
			err := processor.ProcessModel(ctx, name)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to process model %s", name)
				continue
			}

			for _, path := range processor.ModelFiles {
				addFile(path)
			}
		}
	}

	textureRefs := min.GetChildTextures(_map.World, processor.VSlots)

	for i, slot := range processor.Slots {
		if _, ok := textureRefs[int32(i)]; ok {
			for _, path := range slot.Sts {
				texture := processor.SearchFile(ctx, min.NormalizeTexture(path.Name))
				if texture == nil {
					log.Warn().Msgf("unable to find texture %s", path.Name)
					continue
				}
				addFile(texture)
			}
		}
	}

	if len(indexPath) > 0 {
		err = processor.SaveTextureIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to save texture index: %w", err)
		}
	}

	return references, nil
}

// DeriveGameModes analyzes map entities to determine supported game modes.
func DeriveGameModes(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	_map, err := maps.FromGZ(data)
	if err != nil {
		return nil, err
	}

	hasFlags := false
	hasBases := false

	team0 := false
	team1 := false

	for _, ent := range _map.Entities {
		switch constants.EntityType(ent.Type) {
		case constants.EntityTypePlayerStart:
			if ent.Attr2 == 0 {
				team0 = true
			} else if ent.Attr2 == 1 {
				team1 = true
			}
		case constants.EntityTypeFlag:
			hasFlags = true
		case constants.EntityTypeBase:
			hasBases = true
		}
	}

	hasTeamSpawns := team0 && team1

	modes := []string{"ffa"}
	if hasTeamSpawns || hasFlags || hasBases {
		modes = append(modes, "tdm")
	}
	if hasFlags {
		modes = append(modes, "ctf")
	}
	if hasBases {
		modes = append(modes, "capture")
	}

	return modes, nil
}

// DumpModel extracts all files referenced by a model.
func DumpModel(ctx context.Context, roots []assets.Root, name string) ([]min.Mapping, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	err := processor.ProcessModel(ctx, name)
	modelFiles := processor.ModelFiles
	if err != nil || modelFiles == nil {
		return nil, fmt.Errorf("error processing model %s", name)
	}

	references := make([]min.Mapping, 0)
	for _, file := range modelFiles {
		references = append(references, min.Mapping{
			From: file,
			To:   file.Path,
		})
	}

	return references, nil
}

// DumpCFG processes a .cfg file and returns all referenced assets.
func DumpCFG(ctx context.Context, roots []assets.Root, ref *min.Reference, indexPath string) ([]min.Mapping, error) {
	extension := filepath.Ext(ref.Path)

	if extension != ".cfg" {
		return nil, fmt.Errorf("cfg must end in .cfg")
	}

	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	err := processor.ProcessFile(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error processing file: %w", err)
	}

	references := make([]min.Mapping, 0)

	addFile := func(ref *min.Reference) {
		references = append(references, min.Mapping{
			From: ref,
			To:   ref.Path,
		})
	}

	addFile(ref)

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(ctx, path.Name)
			if texture != nil {
				addFile(texture)
			}
		}
	}

	for _, file := range processor.Files {
		addFile(file)
	}

	for _, sound := range processor.Sounds {
		addFile(sound)
	}

	for _, model := range processor.Models {
		name := model.Name
		err := processor.ProcessModel(ctx, name)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to process model %s", name)
			continue
		}

		for _, path := range processor.ModelFiles {
			addFile(path)
		}
	}

	for _, slot := range processor.Slots {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(ctx, path.Name)
			if texture != nil {
				addFile(texture)
			}
		}
	}

	if len(indexPath) > 0 {
		err = processor.SaveTextureIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to save texture index: %w", err)
		}
	}

	return references, nil
}

// ResolveTarget resolves a target string to a min.Reference.
func ResolveTarget(ctx context.Context, roots []assets.Root, target string) (*min.Reference, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	if assets.FileExists(target) {
		return &min.Reference{
			Path: target,
			Root: nil,
		}, nil
	}

	ref := processor.SearchFile(ctx, target)
	if ref != nil {
		return ref, nil
	}

	parts := strings.Split(target, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid target reference, must be index:path")
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, err
	}

	if index < 0 || index >= len(roots) {
		return nil, fmt.Errorf("index not a root")
	}

	return &min.Reference{
		Path: parts[1],
		Root: roots[index],
	}, nil
}

// Dump resolves and prints all assets for a given target.
func Dump(ctx context.Context, roots []assets.Root, type_ string, indexPath string, target string) error {
	var err error
	var references []min.Mapping

	if type_ == "model" {
		references, err = DumpModel(ctx, roots, target)
	} else {
		reference, resolveErr := ResolveTarget(ctx, roots, target)
		if resolveErr != nil {
			return fmt.Errorf("could not resolve target: %w", resolveErr)
		}

		switch type_ {
		case "map":
			references, err = DumpMap(ctx, roots, reference, indexPath)
		case "cfg":
			references, err = DumpCFG(ctx, roots, reference, indexPath)
		default:
			return fmt.Errorf("invalid type %s", type_)
		}
	}

	if err != nil {
		return fmt.Errorf("could not parse file: %w", err)
	}

	if references == nil {
		return fmt.Errorf("no references found")
	}

	references = min.CrunchReferences(references)

	for _, path := range references {
		resolved, err := path.From.Resolve(ctx)
		if err != nil {
			return fmt.Errorf("could not resolve asset %s: %w", path.From.String(), err)
		}
		fmt.Printf("%s->%s\n", resolved, path.To)
	}

	return nil
}

// Download fetches assets from remote roots and saves them to outDir.
func Download(ctx context.Context, roots []assets.Root, outDir string, targets []string) error {
	outCache := assets.FSStore(outDir)

	for _, target := range targets {
		found := false

		for _, root := range roots {
			remoteRoot, ok := root.(*assets.PackagedRoot)
			if !ok {
				continue
			}

			data, err := remoteRoot.ReadAsset(ctx, target)
			if err == assets.Missing {
				continue
			}
			if err != nil {
				return fmt.Errorf("could not resolve asset %s: %w", target, err)
			}

			err = outCache.Set(ctx, target, data)
			if err != nil {
				return fmt.Errorf("could not save asset %s: %w", target, err)
			}

			found = true
			break
		}

		if !found {
			return fmt.Errorf("could not find asset '%s'", target)
		}
	}

	return nil
}

// List returns all files from packaged roots.
func List(roots []assets.Root) []string {
	var result []string
	for _, root := range roots {
		remoteRoot, ok := root.(*assets.PackagedRoot)
		if !ok {
			continue
		}

		for file := range remoteRoot.FS {
			result = append(result, file)
		}
	}
	return result
}

// QueryResult holds the result of a file query.
type QueryResult struct {
	Target   string
	Resolved string
}

// Query resolves file paths against roots.
func Query(ctx context.Context, roots []assets.Root, targets []string) ([]QueryResult, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	var results []QueryResult
	for _, target := range targets {
		ref := processor.SearchFile(ctx, target)

		resolved := "nil"
		if ref != nil {
			r, err := ref.Resolve(ctx)
			if err != nil {
				return nil, fmt.Errorf("could not resolve asset %s: %w", target, err)
			}
			resolved = r
		}

		results = append(results, QueryResult{
			Target:   target,
			Resolved: resolved,
		})
	}

	return results, nil
}

// Hash computes a combined SHA-256 hash of the given assets.
func Hash(ctx context.Context, roots []assets.Root, targets []string) (string, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))
	hash := sha256.New()

	for _, target := range targets {
		ref := processor.SearchFile(ctx, target)
		if ref == nil {
			continue
		}

		data, err := ref.ReadFile(ctx)
		if err != nil {
			return "", fmt.Errorf("could not read asset %s: %w", target, err)
		}

		hash.Write(data)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
