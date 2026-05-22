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
}

func (cmd *BundleCmd) Run() error {
	manifest, err := packager.LoadManifest(cmd.Manifest)
	if err != nil {
		return err
	}

	cache := assets.FSStore(CLI.Cache)
	os.MkdirAll(CLI.Cache, 0755)

	ctx := context.Background()
	return packager.RunManifest(ctx, manifest, cmd.Root, cache)
}
