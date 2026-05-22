package assets

import (
	"context"
	"os"

	pkgassets "github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"
)

// Cmd is the top-level subcommand for asset operations.
type Cmd struct {
	Root  []string `help:"Asset source roots." name:"root" short:"r"`
	Cache string   `help:"Cache directory." default:"cache/"`

	Bundle BundleCmd `cmd:"" help:"Build assets from a YAML manifest."`
	Index  IndexCmd  `cmd:"" help:"Index a directory into a raw .index.source."`
	Info   InfoCmd   `cmd:"" help:"Inspect assets, roots, and indexes."`
	Quad   QuadCmd   `cmd:"" help:"Build Quadropolis assets."`
}

func (cmd *Cmd) LoadRoots() (context.Context, []pkgassets.Root, pkgassets.Store, error) {
	ctx := context.Background()
	cache := pkgassets.FSStore(cmd.Cache)
	os.MkdirAll(cmd.Cache, 0755)
	roots, err := pkgassets.LoadRoots(ctx, cache, cmd.Root, false)
	if err != nil {
		return nil, nil, nil, err
	}
	return ctx, roots, cache, nil
}

// BundleCmd builds assets from a YAML manifest.
type BundleCmd struct {
	Manifest string   `arg:"" help:"Path to manifest YAML file."`
	Root     []string `help:"Additional asset roots (prepended to manifest roots)." name:"extra-root"`
	Maps     []string `help:"Override map filter." name:"maps"`
	Models   []string `help:"Override model filter." name:"models"`
	Textures []string `help:"Override texture filter." name:"textures"`
}

func (cmd *BundleCmd) Run(parent *Cmd) error {
	manifest, err := packager.LoadManifest(cmd.Manifest)
	if err != nil {
		return err
	}
	if len(cmd.Maps) > 0 {
		manifest.Maps = &packager.PatternFilter{Include: packager.PatternSet(cmd.Maps)}
	}
	if len(cmd.Models) > 0 {
		manifest.Models = &packager.PatternFilter{Include: packager.PatternSet(cmd.Models)}
	}
	if len(cmd.Textures) > 0 {
		manifest.Textures = &packager.PatternFilter{Include: packager.PatternSet(cmd.Textures)}
	}
	cache := pkgassets.FSStore(parent.Cache)
	os.MkdirAll(parent.Cache, 0755)
	return packager.RunManifest(context.Background(), manifest, cmd.Root, cache)
}
