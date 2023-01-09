package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/game"
	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func DumpMap(roots []string, filename string) ([]min.Reference, error) {
	extension := filepath.Ext(filename)

	if extension != ".ogz" {
		return nil, fmt.Errorf("map must end in .ogz")
	}

	_map, err := maps.FromFile(filename)

	if err != nil {
		return nil, err
	}

	processor := min.NewProcessor(roots, _map.VSlots)

	references := make([]min.Reference, 0)

	var addFile func(file string)
	addFile = func(file string) {
		normalized := processor.NormalizeFile(file)
		if opt.IsNone(normalized) {
			return
		}
		references = append(references, normalized.Value)
	}

	// Map files can be mapped into packages/base/
	addMapFile := func(file string) {
		target := file

		if filepath.IsAbs(file) {
			absolute, err := filepath.Abs(file)

			if err != nil {
				log.Fatal().Err(err)
			}
			target = absolute
		}

		if !min.FileExists(target) {
			return
		}

		relative := processor.GetRootRelative(target)

		if opt.IsSome(relative) {
			addFile(relative.Value)
			return
		}

		reference := min.Reference{}
		reference.Absolute = target
		reference.Relative = fmt.Sprintf("packages/base/%s", filepath.Base(file))
		references = append(references, reference)
	}

	addMapFile(filename)

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

		if opt.IsSome(resolved) {
			addFile(resolved.Value)
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

	if opt.IsNone(defaultPath) {
		log.Fatal().Msg("Root with data/default_map_settings.cfg not provided")
	}

	err = processor.ProcessFile(defaultPath.Value)
	if err != nil {
		log.Fatal().Err(err)
	}

	cfgName := min.ReplaceExtension(filename, "cfg")
	if min.FileExists(cfgName) {
		err = processor.ProcessFile(cfgName)
		if err != nil {
			log.Fatal().Err(err)
		}

		addMapFile(cfgName)
	}

	for _, extension := range []string{"png", "jpg"} {
		shotName := min.ReplaceExtension(filename, extension)
		addMapFile(shotName)
	}

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			addFile(path.Name)
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
			for _, path := range model.Paths {
				addFile(path)
			}
		}
	}

	textureRefs := min.GetChildTextures(_map.WorldRoot.Children, processor.VSlots)

	for i, slot := range processor.Slots {
		if _, ok := textureRefs[int32(i)]; ok {
			for _, path := range slot.Sts {
				addFile(path.Name)
			}
		}
	}

	return references, nil
}

const MODEL_DIR = "packages/models"

func DumpModel(roots []string, filename string) ([]min.Reference, error) {
	extension := filepath.Ext(filename)

	if extension != ".cfg" {
		return nil, fmt.Errorf("Model must end in .cfg")
	}

	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	normalized := processor.NormalizeFile(filename)
	if opt.IsNone(normalized) {
		return nil, fmt.Errorf("Could not normalize model path")
	}

	relativePath := normalized.Value.Relative
	if !strings.HasPrefix(relativePath, MODEL_DIR) {
		return nil, fmt.Errorf("Model not in model directory")
	}

	modelName := filepath.Dir(relativePath[len(MODEL_DIR):])

	modelFiles, err := processor.ProcessModel(modelName)
	if err != nil || opt.IsNone(modelFiles) {
		return nil, fmt.Errorf("Error processing model")
	}

	references := make([]min.Reference, 0)

	var addFile func(file string)
	addFile = func(file string) {
		normalized := processor.NormalizeFile(file)
		if opt.IsNone(normalized) {
			return
		}
		references = append(references, normalized.Value)
	}

	for _, file := range modelFiles.Value {
		addFile(file)
	}

	return references, nil
}

func DumpCFG(roots []string, filename string) ([]min.Reference, error) {
	extension := filepath.Ext(filename)

	if extension != ".cfg" {
		return nil, fmt.Errorf("cfg must end in .cfg")
	}

	processor := min.NewProcessor(roots, make([]*maps.VSlot, 0))

	err := processor.ProcessFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error processing file")
	}

	references := make([]min.Reference, 0)

	var addFile func(file string)
	addFile = func(file string) {
		normalized := processor.NormalizeFile(file)
		if opt.IsNone(normalized) {
			return
		}
		references = append(references, normalized.Value)
	}

	addFile(filename)

	for _, slot := range processor.Materials {
		for _, path := range slot.Sts {
			addFile(path.Name)
		}
	}

	for _, file := range processor.Files {
		addFile(file)
	}

	for _, sound := range processor.Sounds {
		addFile(sound)
	}

	for _, model := range processor.Models {
		for _, path := range model.Paths {
			addFile(path)
		}
	}

	for _, slot := range processor.Slots {
		for _, path := range slot.Sts {
			addFile(path.Name)
		}
	}

	return references, nil
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	var roots min.RootFlags

	flag.Var(&roots, "root", "Specify an explicit asset root directory. Roots are searched in order of appearance.")
	parseType := flag.String("type", "map", "The type of the asset to parse, one of 'map', 'model', 'cfg'.")
	flag.Parse()

	absoluteRoots := fp.Map[string, string](func(root string) string {
		absolute, err := filepath.Abs(root)
		if err != nil {
			log.Fatal().Err(err)
		}
		return absolute
	})(roots)

	args := flag.Args()

	if len(args) != 1 {
		log.Fatal().Msg("You must provide only a single argument.")
	}

	filename, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatal().Err(err).Msg("failed to make filename absolute")
	}

	var references []min.Reference

	switch *parseType {
	case "map":
		references, err = DumpMap(absoluteRoots, filename)
	case "model":
		references, err = DumpModel(absoluteRoots, filename)
	case "cfg":
		references, err = DumpCFG(absoluteRoots, filename)
	default:
		log.Fatal().Msgf("invalid type %s", *parseType)
	}

	if err != nil || references == nil {
		log.Fatal().Err(err).Msg("could not parse file")
	}

	references = min.CrunchReferences(references)

	for _, path := range references {
		fmt.Printf("%s->%s\n", path.Absolute, path.Relative)
	}
}
