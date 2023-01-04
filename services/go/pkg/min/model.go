package min

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/repeale/fp-go"
	"github.com/repeale/fp-go/option"

	"github.com/rs/zerolog/log"
)

// This is slightly different from the other Normalize because models
// specifically use relative paths for some stuff
func (p *Processor) NormalizeModelPath(modelDir string, path string) string {
	return filepath.Clean(filepath.Join(modelDir, path))
}

func (p *Processor) ResolveRelative(modelDir string, file string) opt.Option[string] {
	path := p.NormalizeModelPath(modelDir, file)
	resolved := p.SearchFile(path)

	if opt.IsSome(resolved) {
		return resolved
	}

	// Also check the parent dir (Cube does this, too)
	parent := filepath.Join(
		filepath.Dir(path),
		"..",
		filepath.Base(path),
	)
	return p.SearchFile(parent)
}

func (p *Processor) ProcessModelFile(modelDir string, modelType string, path string) (opt.Option[[]string], error) {
	results := make([]string, 0)

	addFile := func(file string) {
		results = append(results, file)
	}

	addRootFile := func(file string) {
		resolved := p.SearchFile(file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find root-relative model path %s", file)
			return
		}

		results = append(results, resolved.Value)
	}

	// Some references are relative to the model config
	addRelative := func(file string) {
		resolved := p.ResolveRelative(modelDir, file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find cfg-relative model path %s (%s)", file, path)
			return
		}

		results = append(results, resolved.Value)
	}

	// Model textures tend to also come with a DDS counterpart
	expandTexture := func(texture string) []string {
		normalized := NormalizeTexture(texture)

		hasDDS := fp.Some(
			func(x []string) bool {
				return x[1] == "dds"
			},
		)(TEXTURE_COMMAND_REGEX.FindAllStringSubmatch(texture, -1))

		if hasDDS {
			extension := filepath.Ext(normalized)
			ddsPath := fmt.Sprintf(
				"%s.dds",
				normalized[:len(normalized)-len(extension)],
			)
			return []string{normalized, ddsPath}
		}

		return []string{normalized}
	}

	addTexture := func(texture string) {
		for _, file := range expandTexture(texture) {
			addRelative(file)
		}
	}

	addFile(path)

	src, err := os.ReadFile(path)
	if err != nil {
		return opt.None[[]string](), errors.New(fmt.Sprintf("Failed to read %s", path))
	}

	for _, line := range strings.Split(string(src), "\n") {
		args := ParseLine(line)

		if len(args) == 0 {
			continue
		}

		command := args[0]

		if strings.HasPrefix(command, modelType) {
			command = command[len(modelType):]
		}

		switch command {
		case "anim":
			if len(args) < 3 {
				break
			}

			// `anim` uses anim indices and files, so no need to
			// error if it's not found
			for i := 2; i < len(args); i++ {
				resolved := p.ResolveRelative(modelDir, args[i])
				if opt.IsNone(resolved) {
					continue
				}

				addTexture(args[i])
			}

		case "bumpmap":
			if len(args) < 3 {
				break
			}

			addTexture(args[2])

		case "exec":
			if len(args) != 2 {
				break
			}
			execPath := args[1]

			resolved := p.SearchFile(execPath)

			if opt.IsNone(resolved) {
				log.Printf("Could not find %s", execPath)
			} else {
				files, err := p.ProcessModelFile(
					modelDir,
					modelType,
					resolved.Value,
				)
				if err != nil {
					return opt.None[[]string](), err
				}
				if opt.IsNone(files) {
					break
				}

				results = append(results, files.Value...)
			}

		case "load":
			if len(args) < 2 {
				break
			}

			addTexture(args[1])

		case "skin":
			if len(args) < 3 {
				break
			}

			for i := 2; i < 4; i++ {
				if i == len(args) {
					break
				}

				addTexture(args[i])
			}

		case "mdlenvmap":
			if len(args) != 4 {
				break
			}

			for _, texture := range p.FindCubemap(NormalizeTexture(args[3])) {
				addRootFile(texture)
			}

		case "ambient":
		case "animpart":
		case "basemodelcfg": // TODO dynamic code in models?
		case "cullface":
		case "dir":
		case "mdlalphablend":
		case "mdlalphadepth":
		case "mdlalphatest":
		case "mdlambient":
		case "mdlbb":
		case "mdlcollide":
		case "mdlcullface":
		case "mdldepthoffset":
		case "mdlellipsecollide":
		case "mdlextendbb":
		case "mdlfullbright":
		case "mdlglare":
		case "mdlglow":
		case "mdlpitch":
		case "mdlscale":
		case "mdlshader":
		case "mdlshadow":
		case "mdlspec":
		case "mdlspin":
		case "mdltrans":
		case "mdlyaw":
		case "noclip":
		case "pitch":
		case "rdeye":
		case "rdjoint":
		case "rdlimitdist":
		case "rdlimitrot":
		case "rdtri":
		case "rdvert":
		case "scroll":
		case "spec":
		case "tag":
			break

		default:
			log.Printf("Unhandled modelcommand: %s", command)
		}
	}

	return opt.Some[[]string](results), nil
}

func (p *Processor) ProcessModel(path string) (opt.Option[[]string], error) {
	results := make([]string, 0)

	modelDir := filepath.Join(
		"packages/models",
		path,
	)

	// Some references are relative to the model config
	addRelative := func(file string) {
		resolved := p.ResolveRelative(modelDir, file)

		if opt.IsNone(resolved) {
			log.Printf("Failed to find cfg-relative model path %s (%s)", file, path)
			return
		}

		results = append(results, resolved.Value)
	}

	_type := Find[string](func(x string) bool {
		// First look for the cfg
		cfg := fmt.Sprintf(
			"%s/%s.cfg",
			modelDir,
			x,
		)

		resolved := p.SearchFile(cfg)

		if opt.IsSome(resolved) {
			return true
		}

		// Then tris, since that is also there
		tris := fmt.Sprintf(
			"%s/tris.%s",
			modelDir,
			x,
		)

		resolved = p.SearchFile(tris)

		if opt.IsSome(resolved) {
			return true
		}

		return false
	})(MODELTYPES)

	if opt.IsNone(_type) {
		return opt.None[[]string](), errors.New(fmt.Sprintf("Failed to infer type for model %s", path))
	}

	modelType := _type.Value

	defaultFiles := []string{
		fmt.Sprintf("tris.%s", modelType),
		"skin.png",
		"skin.jpg",
		"mask.png",
		"mask.jpg",
	}

	hadDefault := false
	for _, _default := range defaultFiles {
		resolved := p.ResolveRelative(modelDir, _default)

		if opt.IsNone(resolved) {
			continue
		}

		hadDefault = true
		addRelative(_default)
	}

	cfgPath := fmt.Sprintf(
		"%s/%s.cfg",
		modelDir,
		modelType,
	)

	resolved := p.SearchFile(cfgPath)

	if opt.IsNone(resolved) {
		if !hadDefault {
			return opt.None[[]string](), errors.New(fmt.Sprintf("Model %s had neither defaults nor a .cfg", path))
		}

		return opt.Some[[]string](results), nil
	}

	cfgFiles, err := p.ProcessModelFile(modelDir, modelType, resolved.Value)
	if err != nil {
		return opt.None[[]string](), err
	}

	if opt.IsNone(cfgFiles) {
		return opt.None[[]string](), nil
	}

	results = append(results, cfgFiles.Value...)

	return opt.Some[[]string](results), nil
}

func (processor *Processor) ResetModels() {
	processor.Models = make([]Model, 0)
}

func (processor *Processor) AddModel(textures []string) {
	model := Model{}
	model.Paths = textures
	processor.Models = append(processor.Models, model)
}
