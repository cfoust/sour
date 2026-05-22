package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/dump"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var CLI struct {
	Root  []string `help:"Asset source roots." name:"root" short:"r"`
	Cache string   `help:"Cache directory." default:"cache/"`
	CPU   string   `help:"Write CPU profile to file." name:"cpu" optional:""`

	Dump     DumpCmd     `cmd:"" help:"Dump asset dependencies."`
	Download DownloadCmd `cmd:"" help:"Download assets from remote sources."`
	List     ListCmd     `cmd:"" help:"List root files."`
	Query    QueryCmd    `cmd:"" help:"Query file resolution."`
	Hash     HashCmd     `cmd:"" help:"Hash assets."`
	Modes    ModesCmd    `cmd:"" help:"Derive game modes from map."`
	Quad     QuadCmd     `cmd:"" help:"Build Quadropolis assets."`
	Catalog  CatalogCmd  `cmd:"" help:"Generate catalog files."`
	DumpIndex DumpIndexCmd `cmd:"" help:"Dump .index.source to JSON." name:"dump-index"`
	Index     IndexRootCmd `cmd:"" help:"Index a directory into a raw .index.source."`
	Bundle    BundleCmd    `cmd:"" help:"Build assets from a YAML manifest."`
}

// Globals holds shared state from top-level flags.
type Globals struct {
	Roots []assets.Root
	Cache assets.Store
	Ctx   context.Context
}

func loadGlobals() (*Globals, error) {
	ctx := context.Background()
	cache := assets.FSStore(CLI.Cache)
	os.MkdirAll(CLI.Cache, 0755)

	roots, err := assets.LoadRoots(ctx, cache, CLI.Root, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load roots: %w", err)
	}

	return &Globals{
		Roots: roots,
		Cache: cache,
		Ctx:   ctx,
	}, nil
}

// DumpCmd dumps asset dependencies for a map, model, or cfg.
type DumpCmd struct {
	Type  string `help:"Asset type: map, model, cfg." default:"map" name:"type"`
	Index string `help:"Save texture index to file." optional:""`
	Target string `arg:"" help:"Target file or reference."`
}

func (cmd *DumpCmd) Run() error {
	g, err := loadGlobals()
	if err != nil {
		return err
	}
	return dump.Dump(g.Ctx, g.Roots, cmd.Type, cmd.Index, cmd.Target)
}

// DownloadCmd downloads assets from remote sources.
type DownloadCmd struct {
	Outdir  string   `help:"Output directory." default:"output/" name:"outdir"`
	Targets []string `arg:"" help:"Asset IDs to download."`
}

func (cmd *DownloadCmd) Run() error {
	g, err := loadGlobals()
	if err != nil {
		return err
	}
	return dump.Download(g.Ctx, g.Roots, cmd.Outdir, cmd.Targets)
}

// ListCmd lists all files from roots.
type ListCmd struct{}

func (cmd *ListCmd) Run() error {
	g, err := loadGlobals()
	if err != nil {
		return err
	}
	for _, file := range dump.List(g.Roots) {
		fmt.Println(file)
	}
	return nil
}

// QueryCmd queries file resolution.
type QueryCmd struct {
	Targets []string `arg:"" help:"Paths to query."`
}

func (cmd *QueryCmd) Run() error {
	g, err := loadGlobals()
	if err != nil {
		return err
	}
	results, err := dump.Query(g.Ctx, g.Roots, cmd.Targets)
	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Printf("%s->%s\n", r.Target, r.Resolved)
	}
	return nil
}

// HashCmd computes a hash of assets.
type HashCmd struct {
	Targets []string `arg:"" help:"Paths to hash."`
}

func (cmd *HashCmd) Run() error {
	g, err := loadGlobals()
	if err != nil {
		return err
	}
	hash, err := dump.Hash(g.Ctx, g.Roots, cmd.Targets)
	if err != nil {
		return err
	}
	fmt.Print(hash)
	return nil
}

// ModesCmd derives game modes from a map.
type ModesCmd struct {
	File string `arg:"" help:"Path to .ogz file."`
}

func (cmd *ModesCmd) Run() error {
	modes, err := dump.DeriveGameModes(cmd.File)
	if err != nil {
		return err
	}
	out, _ := json.Marshal(modes)
	fmt.Println(string(out))
	return nil
}

// DumpIndexCmd dumps a CBOR .index.source to JSON.
type DumpIndexCmd struct {
	File string `arg:"" help:"Path to .index.source file."`
}

func (cmd *DumpIndexCmd) Run() error {
	return dumpIndexToJSON(cmd.File)
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := kong.Parse(&CLI,
		kong.Name("sourdump"),
		kong.Description("Asset pipeline tool for Sour"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
	)

	if CLI.CPU != "" {
		f, err := os.Create(CLI.CPU)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
