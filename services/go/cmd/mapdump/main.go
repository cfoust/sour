package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	var roots min.RootFlags

	flag.Var(&roots, "root", "Specify an explicit asset root directory. Roots are searched in order of appearance.")
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
		log.Fatal().Err(err)
	}

	extension := filepath.Ext(filename)

	if extension != ".ogz" {
		log.Fatal().Msg("Map must end in .ogz")
	}

	_map, err := maps.FromFile(filename)

	if err != nil {
		log.Fatal().Msg("Failed to parse map file")
	}

	processor := min.NewProcessor(absoluteRoots, _map.VSlots)

	references := make([]min.Reference, 0)

	// File paths are strange in Sauer: certain types of assets omit the
	// packages/, others are relative to the config file (models), and this
	// program also accepts map files not inside of a Sauer directory
	// structure. On top of that, we ultimately need to map assets into the
	// game's filesystem correctly. This function normalizes all paths so
	// we can do that more easily.
	var addFile func(file string)
	addFile = func(file string) {
		reference := min.Reference{}

		if filepath.IsAbs(file) {
			reference.Absolute = file

			relative := processor.GetRootRelative(file)

			if opt.IsNone(relative) {
				log.Printf("File absolute but not in root: %s", file)
				return
			}

			reference.Relative = relative.Value
			references = append(references, reference)
			return
		}

		// This might just be a file (like a config) that was specified with a relative path
		absolute, err := filepath.Abs(file)
		if err != nil {
			log.Fatal().Err(err)
		}

		if min.FileExists(absolute) {
			addFile(absolute)
			return
		}

		// If it's relative, it must be inside of a root
		resolved := processor.SearchFile(file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find relative file in roots: %s", file)
			return
		}

		// Sometimes a file was specified without packages/ so we need
		// to normalize it
		addFile(resolved.Value)
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
		value := string(skybox.(maps.StringVariable))
		for _, path := range processor.FindCubemap(min.NormalizeTexture(value)) {
			addFile(path)
		}
	}

	if cloudlayer, ok := _map.Vars["cloudlayer"]; ok {
		value := string(cloudlayer.(maps.StringVariable))
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

	references = processor.CrunchReferences(references)

	for _, path := range references {
		fmt.Printf("%s->%s\n", path.Absolute, path.Relative)
	}
}
