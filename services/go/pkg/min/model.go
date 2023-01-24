package min

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/repeale/fp-go/option"

	"github.com/rs/zerolog/log"
)

// This is slightly different from the other Normalize because models
// specifically use relative paths for some stuff
func (p *Processor) NormalizeModelPath(modelDir string, path string) string {
	return filepath.Clean(filepath.Join(modelDir, path))
}

func (p *Processor) ResolveRelative(modelDir string, file string) *Reference {
	path := p.NormalizeModelPath(modelDir, file)
	resolved := p.SearchFile(path)

	if resolved != nil {
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

func (p *Processor) ProcessModelFile(modelDir string, modelType string, ref *Reference) ([]*Reference, error) {
	results := make([]*Reference, 0)

	addFile := func(ref *Reference) {
		results = append(results, ref)
	}

	addFile(ref)

	src, err := ref.ReadFile()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read %s", ref.Path))
	}

	previous := p.current
	p.current = ref
	p.cfgVM.Run(string(src))
	p.current = previous

	return results, nil
}

func (p *Processor) ProcessModel(path string) error {
	p.ModelFiles = make([]*Reference, 0)

	modelDir := filepath.Join(
		"packages/models",
		path,
	)

	// Some references are relative to the model config
	addRelative := func(file string) {
		resolved := p.ResolveRelative(modelDir, file)

		if resolved == nil {
			log.Printf("Failed to find cfg-relative model path %s (%s)", file, path)
			return
		}

		p.ModelFiles = append(p.ModelFiles, resolved)
	}

	_type := Find(func(x string) bool {
		// First look for the cfg
		cfg := fmt.Sprintf(
			"%s/%s.cfg",
			modelDir,
			x,
		)

		resolved := p.SearchFile(cfg)

		if resolved != nil {
			return true
		}

		// Then tris, since that is also there
		tris := fmt.Sprintf(
			"%s/tris.%s",
			modelDir,
			x,
		)

		resolved = p.SearchFile(tris)

		if resolved != nil {
			return true
		}

		return false
	})(MODELTYPES)

	if opt.IsNone(_type) {
		return errors.New(fmt.Sprintf("Failed to infer type for model %s", path))
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

		if resolved == nil {
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

	if resolved == nil {
		if !hadDefault {
			return errors.New(fmt.Sprintf("Model %s had neither defaults nor a .cfg", path))
		}

		return nil
	}

	p.cfgVM.Run(fmt.Sprintf(`set mdlname "%s"`, path))
	cfgFiles, err := p.ProcessModelFile(modelDir, modelType, resolved)
	if err != nil {
		return nil
	}

	if cfgFiles == nil {
		return fmt.Errorf("no cfg files")
	}

	p.ModelFiles = append(p.ModelFiles, cfgFiles...)

	return nil
}

func (processor *Processor) ResetModels(limit int) {
	if limit > len(processor.Models) {
		limit = len(processor.Models)
	}
	if limit < 0 {
		limit = 0
	}
	processor.Models = processor.Models[:limit]
}

func (processor *Processor) AddModel(textures []*Reference) {
	model := Model{}
	model.Paths = textures
	processor.Models = append(processor.Models, model)
}
