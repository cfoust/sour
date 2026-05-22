package main

import (
	"context"
	"os"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"
)

// BundleCmd builds assets from a YAML manifest.
type BundleCmd struct {
	Manifest string   `arg:"" help:"Path to manifest YAML file."`
	Root     []string `help:"Additional asset roots (prepended to manifest roots)." name:"extra-root"`
	Maps     []string `help:"Override map filter (e.g. dust2, 'complex', 'd*')." name:"maps"`
	Models   []string `help:"Override model filter." name:"models"`
	Textures []string `help:"Override texture filter." name:"textures"`
}

func (cmd *BundleCmd) Run() error {
	manifest, err := packager.LoadManifest(cmd.Manifest)
	if err != nil {
		return err
	}

	if len(cmd.Maps) > 0 {
		manifest.Maps = &packager.PatternFilter{
			Include: packager.PatternSet(cmd.Maps),
		}
	}
	if len(cmd.Models) > 0 {
		manifest.Models = &packager.PatternFilter{
			Include: packager.PatternSet(cmd.Models),
		}
	}
	if len(cmd.Textures) > 0 {
		manifest.Textures = &packager.PatternFilter{
			Include: packager.PatternSet(cmd.Textures),
		}
	}

	cache := assets.FSStore(CLI.Cache)
	os.MkdirAll(CLI.Cache, 0755)

	ctx := context.Background()
	return packager.RunManifest(ctx, manifest, cmd.Root, cache)
}
