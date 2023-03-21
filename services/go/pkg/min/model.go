package min

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"context"

	"github.com/repeale/fp-go/option"

	"github.com/rs/zerolog/log"
)

// This is slightly different from the other Normalize because models
// specifically use relative paths for some stuff
func (p *Processor) NormalizeModelPath(modelDir string, path string) string {
	return filepath.Clean(filepath.Join(modelDir, path))
}

func (p *Processor) ResolveRelative(ctx context.Context, modelDir string, file string) *Reference {
	path := p.NormalizeModelPath(modelDir, file)
	resolved := p.SearchFile(ctx, path)

	if resolved != nil {
		return resolved
	}

	// Also check the parent dir (Cube does this, too)
	parent := filepath.Join(
		filepath.Dir(path),
		"..",
		filepath.Base(path),
	)
	return p.SearchFile(ctx, parent)
}

func (p *Processor) ProcessModelFile(ctx context.Context, modelDir string, modelType string, ref *Reference) ([]*Reference, error) {
	results := make([]*Reference, 0)

	addFile := func(ref *Reference) {
		results = append(results, ref)
	}

	addFile(ref)

	src, err := ref.ReadFile(ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read %s", ref.Path))
	}

	previous := p.current
	p.current = ref
	p.cfgVM.Run(string(src))
	p.current = previous

	return results, nil
}

func (p *Processor) ProcessModel(ctx context.Context, path string) error {
	p.processingModel = true
	p.ModelFiles = make([]*Reference, 0)

	defer func() {
		p.processingModel = false
	}()

	modelDir := filepath.Join(
		"packages/models",
		path,
	)

	// Some references are relative to the model config
	addRelative := func(file string) {
		resolved := p.ResolveRelative(ctx, modelDir, file)

		if resolved == nil {
			log.Printf("Failed to find cfg-relative model path %s (%s)", file, path)
			return
		}

		p.ModelFiles = append(p.ModelFiles, resolved)
	}

	p.modelName = path
	if strings.HasPrefix(p.modelName, "/") {
		p.modelName = p.modelName[1:]
	}
	p.modelDir = p.modelName

	_type := Find(func(x string) bool {
		// First look for the cfg
		cfg := fmt.Sprintf(
			"%s.cfg",
			x,
		)

		resolved := p.SearchFile(ctx, cfg)

		if resolved != nil {
			return true
		}

		// Then tris, since that is also there
		tris := fmt.Sprintf(
			"tris.%s",
			x,
		)

		resolved = p.SearchFile(ctx, tris)

		if resolved != nil {
			return true
		}

		return false
	})(MODELTYPES)

	if opt.IsNone(_type) {
		return errors.New(fmt.Sprintf("Failed to infer type for model '%s'", path))
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
		resolved := p.ResolveRelative(ctx, modelDir, _default)

		if resolved == nil {
			continue
		}

		hadDefault = true
		addRelative(_default)
	}

	cfgPath := fmt.Sprintf(
		"%s.cfg",
		modelType,
	)

	resolved := p.SearchFile(ctx, cfgPath)

	if resolved == nil {
		if !hadDefault {
			return errors.New(fmt.Sprintf("Model %s had neither defaults nor a .cfg", path))
		}

		return nil
	}

	cfgFiles, err := p.ProcessModelFile(ctx, modelDir, modelType, resolved)
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

func (processor *Processor) AddModel(name string) {
	processor.Models = append(processor.Models, Model{
		Name: name,
	})
}
