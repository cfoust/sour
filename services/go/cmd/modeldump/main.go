package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"strings"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"

	"github.com/cfoust/sour/pkg/maps"
	"github.com/cfoust/sour/pkg/min"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const MODEL_DIR = "packages/models"

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

	if extension != ".cfg" {
		log.Fatal().Msg("Model must end in .cfg")
	}

	processor := min.NewProcessor(absoluteRoots, make([]*maps.VSlot, 0))

	normalized := processor.NormalizeFile(filename)
	if opt.IsNone(normalized) {
		log.Fatal().Msg("Could not normalize model path")
	}

	relativePath := normalized.Value.Relative
	if !strings.HasPrefix(relativePath, MODEL_DIR) {
		log.Fatal().Msg("Model not in model directory")
	}

	modelName := filepath.Dir(relativePath[len(MODEL_DIR):])

	modelFiles, err := processor.ProcessModel(modelName)
	if err != nil || opt.IsNone(modelFiles) {
		log.Fatal().Err(err).Msg("Error processing model")
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

	references = processor.CrunchReferences(references)

	for _, path := range references {
		fmt.Printf("%s->%s\n", path.Absolute, path.Relative)
	}
}
