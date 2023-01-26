package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func DumpMap(roots []assets.Root, ref *min.Reference, indexPath string) ([]min.Mapping, error) {
	extension := filepath.Ext(ref.Path)

	if extension != ".ogz" {
		return nil, fmt.Errorf("map must end in .ogz")
	}

	data, err := ref.ReadFile()
	if err != nil {
		return nil, err
	}

	_map, err := maps.FromGZ(data)

	if err != nil {
		return nil, err
	}

	processor := min.NewProcessor(roots, _map.VSlots)

	references := make([]min.Mapping, 0)

	var addFile func(ref *min.Reference)
	addFile = func(ref *min.Reference) {
		references = append(references, min.Mapping{
			From: ref,
			To:   ref.Path,
		})
	}

	// Map files can be mapped into packages/base/
	addMapFile := func(ref *min.Reference) {
		if !ref.Exists() {
			return
		}

		reference := min.Mapping{}
		reference.From = ref
		reference.To = fmt.Sprintf("packages/base/%s", filepath.Base(ref.Path))
		references = append(references, reference)
	}

	addMapFile(ref)

	// Some variables contain textures
	if skybox, ok := _map.Vars["skybox"]; ok {
		value := string(skybox.(game.StringVariable))
		for _, path := range processor.FindCubemap(min.NormalizeTexture(value)) {
			addFile(path)
		}
	}

	if cloudlayer, ok := _map.Vars["cloudlayer"]; ok {
		value := string(cloudlayer.(game.StringVariable))
		resolved := processor.FindTexture(min.NormalizeTexture(value))

		if resolved != nil {
			addFile(resolved)
		}
	}

	if cloudbox, ok := _map.Vars["cloudbox"]; ok {
		value := string(cloudbox.(game.StringVariable))
		for _, path := range processor.FindCubemap(min.NormalizeTexture(value)) {
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

	// Always load the default map settings
	defaultPath := processor.SearchFile("data/default_map_settings.cfg")

	if defaultPath == nil {
		log.Fatal().Msg("Root with data/default_map_settings.cfg not provided")
	}

	err = processor.ProcessFile(defaultPath)
	if err != nil {
		log.Fatal().Err(err)
	}

	cfg := min.ReplaceExtension(ref, "cfg")
	if cfg.Exists() {
		err = processor.ProcessFile(cfg)
		if err != nil {
			log.Fatal().Err(err)
		}

		addMapFile(cfg)
	}

	for _, extension := range []string{"png", "jpg"} {
		shotName := min.ReplaceExtension(ref, extension)
		addMapFile(shotName)
	}

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(path.Name)
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
			err := processor.ProcessModel(name)
			if err != nil {
				log.Fatal().Err(err).Msgf("Failed to process model %s", name)
				continue
			}

			for _, path := range processor.ModelFiles {
				addFile(path)
			}
		}
	}

	textureRefs := min.GetChildTextures(_map.C, processor.VSlots)

	for i, slot := range processor.Slots {
		if _, ok := textureRefs[int32(i)]; ok {
			for _, path := range slot.Sts {
				texture := processor.SearchFile(min.NormalizeTexture(path.Name))
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
		log.Fatal().Err(err)
	}

	return references, nil
}

const MODEL_DIR = "packages/models"

func DumpModel(roots []assets.Root, name string) ([]min.Mapping, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	err := processor.ProcessModel(name)
	modelFiles := processor.ModelFiles
	if err != nil || modelFiles == nil {
		return nil, fmt.Errorf("Error processing model")
	}

	references := make([]min.Mapping, 0)

	var addFile func(ref *min.Reference)
	addFile = func(ref *min.Reference) {
		references = append(references, min.Mapping{
			From: ref,
			To:   ref.Path,
		})
	}

	for _, file := range modelFiles {
		addFile(file)
	}

	return references, nil
}

func DumpCFG(roots []assets.Root, ref *min.Reference, indexPath string) ([]min.Mapping, error) {
	extension := filepath.Ext(ref.Path)

	if extension != ".cfg" {
		return nil, fmt.Errorf("cfg must end in .cfg")
	}

	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	err := processor.ProcessFile(ref)
	if err != nil {
		return nil, fmt.Errorf("error processing file")
	}

	references := make([]min.Mapping, 0)

	var addFile func(ref *min.Reference)
	addFile = func(ref *min.Reference) {
		references = append(references, min.Mapping{
			From: ref,
			To:   ref.Path,
		})
	}

	addFile(ref)

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(path.Name)
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
		err := processor.ProcessModel(name)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to process model %s", name)
			continue
		}

		for _, path := range processor.ModelFiles {
			addFile(path)
		}
	}

	for _, slot := range processor.Slots {
		for _, path := range slot.Sts {
			texture := processor.SearchFile(path.Name)
			if texture != nil {
				addFile(texture)
			}
		}
	}

	if len(indexPath) > 0 {
		err = processor.SaveTextureIndex(indexPath)
		log.Fatal().Err(err)
	}

	return references, nil
}

func resolveTarget(roots []assets.Root, target string) (*min.Reference, error) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	// Base case is a file on the FS, does not need to be in root
	if assets.FileExists(target) {
		return &min.Reference{
			Path: target,
			Root: nil,
		}, nil
	}

	// Just try the file
	ref := processor.SearchFile(target)
	if ref != nil {
		return ref, nil
	}

	// Or a file in a source
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

func Dump(cache assets.Cache, roots []assets.Root, type_ string, indexPath string, target string) {
	var err error
	var references []min.Mapping

	if type_ == "model" {
		references, err = DumpModel(roots, target)
	} else {
		reference, err := resolveTarget(roots, target)
		if err != nil {
			log.Fatal().Err(err).Msg("could not resolve target")
		}

		switch type_ {
		case "map":
			references, err = DumpMap(roots, reference, indexPath)
		case "cfg":
			references, err = DumpCFG(roots, reference, indexPath)
		default:
			log.Fatal().Msgf("invalid type %s", type_)
		}
	}

	if err != nil || references == nil {
		log.Fatal().Err(err).Msg("could not parse file")
	}

	references = min.CrunchReferences(references)

	for _, path := range references {
		// TODO segfault?
		resolved, err := path.From.Resolve()
		if err != nil {
			log.Fatal().Err(err).Msgf("could not resolve asset %s", path.From.String())
			return
		}
		fmt.Printf("%s->%s\n", resolved, path.To)
	}
}

func Download(cache assets.Cache, roots []assets.Root, outDir string, targets []string) {
	outCache := assets.FSCache(outDir)

	for _, target := range targets {
		for _, root := range roots {
			remoteRoot, ok := root.(*assets.RemoteRoot)
			if !ok {
				continue
			}

			data, err := remoteRoot.ReadAsset(target)
			if err == assets.Missing {
				continue
			}
			if err != nil {
				log.Fatal().Err(err).Msgf("could not resolve asset %s", target)
			}

			err = outCache.Set(target, data)
			if err != nil {
				log.Fatal().Err(err).Msgf("could not save asset %s", target)
			}
		}
	}
}

func List(cache assets.Cache, roots []assets.Root) {
	for _, root := range roots {
		remoteRoot, ok := root.(*assets.RemoteRoot)
		if !ok {
			continue
		}

		for file := range remoteRoot.FS {
			fmt.Printf("%s\n", file)
		}
	}
}

func Query(cache assets.Cache, roots []assets.Root, targets []string) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	for _, target := range targets {
		ref := processor.SearchFile(target)

		to := "nil"
		if ref != nil {
			resolved, err := ref.Resolve()
			if err != nil {
				log.Fatal().Err(err).Msgf("could not resolve asset %s", target)
			}
			to = resolved
		}

		fmt.Printf("%s->%s\n", target, to)
	}
}

func Hash(cache assets.Cache, roots []assets.Root, targets []string) {
	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))
	hash := sha256.New()

	for _, target := range targets {
		ref := processor.SearchFile(target)
		// We ignore missing assets when hashing
		if ref == nil {
			continue
		}

		data, err := ref.ReadFile()
		if err != nil {
			log.Fatal().Err(err).Msgf("could not read asset %s", target)
		}

		hash.Write(data)
	}

	fmt.Printf("%x", hash.Sum(nil))
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	var roots min.RootFlags

	flag.Var(&roots, "root", "Specify a source for assets. Roots are searched in order of appearance.")
	cpuProfile := flag.String("cpu", "", "Write cpu profile to `file`.")
	cacheDir := flag.String("cache", "cache/", "The directory in which to cache assets from remote sources.")

	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)
	parseType := dumpCmd.String("type", "map", "The type of the asset to parse, one of 'map', 'model', 'cfg'.")
	indexPath := dumpCmd.String("index", "", "Where to save the index of all texture calls.")
	flag.Parse()

	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	outDir := downloadCmd.String("outdir", "output/", "The directory in which to save the assets.")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	queryCmd := flag.NewFlagSet("query", flag.ExitOnError)
	hashCmd := flag.NewFlagSet("hash", flag.ExitOnError)

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	args := flag.Args()

	if len(args) == 0 {
		log.Fatal().Msg("You must provide at least one argument.")
	}

	cache := assets.FSCache(*cacheDir)
	assetRoots, err := assets.LoadRoots(cache, roots, false)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load roots")
	}

	switch args[0] {
	case "dump":
		dumpCmd.Parse(args[1:])
		args := dumpCmd.Args()
		if len(args) != 1 {
			log.Fatal().Msg("You must provide only a single argument.")
		}
		Dump(cache, assetRoots, *parseType, *indexPath, args[0])
	case "download":
		downloadCmd.Parse(args[1:])
		args := downloadCmd.Args()
		if len(args) == 0 {
			log.Fatal().Msg("You must provide at least one asset.")
		}
		Download(cache, assetRoots, *outDir, args)
	case "list":
		listCmd.Parse(args[1:])
		args := listCmd.Args()
		if len(args) != 0 {
			log.Fatal().Msg("`list` takes no arguments.")
		}
		List(cache, assetRoots)
	case "query":
		queryCmd.Parse(args[1:])
		args := queryCmd.Args()
		if len(args) == 0 {
			log.Fatal().Msg("You must provide at least one path to query.")
		}
		Query(cache, assetRoots, args)
	case "hash":
		hashCmd.Parse(args[1:])
		args := hashCmd.Args()
		if len(args) == 0 {
			log.Fatal().Msg("You must provide at least one path to hash.")
		}
		Hash(cache, assetRoots, args)
	}
}
