package assets

import (
	"encoding/json"
	"fmt"
	"os"

	pkgassets "github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/dump"

	"github.com/fxamacker/cbor/v2"
)

// InfoCmd groups inspection/query subcommands.
type InfoCmd struct {
	List  InfoListCmd  `cmd:"" help:"List asset dependencies for a map, model, or cfg."`
	Fs    InfoFsCmd    `cmd:"" help:"List all files in roots."`
	Query InfoQueryCmd `cmd:"" help:"Resolve file paths through roots."`
	Hash  InfoHashCmd  `cmd:"" help:"Hash resolved assets."`
	Index InfoIndexCmd `cmd:"" help:"Dump .index.source to JSON."`
	Modes InfoModesCmd `cmd:"" help:"Derive game modes from a map."`
}

// sour assets info list
type InfoListCmd struct {
	Type   string `help:"Asset type: map, model, cfg." default:"map" name:"type"`
	Index  string `help:"Save texture index to file." optional:""`
	Target string `arg:"" help:"Target file or reference."`
}

func (cmd *InfoListCmd) Run(parent *Cmd) error {
	ctx, roots, _, err := parent.LoadRoots()
	if err != nil {
		return err
	}
	return dump.Dump(ctx, roots, cmd.Type, cmd.Index, cmd.Target)
}

// sour assets info fs
type InfoFsCmd struct{}

func (cmd *InfoFsCmd) Run(parent *Cmd) error {
	_, roots, _, err := parent.LoadRoots()
	if err != nil {
		return err
	}
	for _, file := range dump.List(roots) {
		fmt.Println(file)
	}
	return nil
}

// sour assets info query
type InfoQueryCmd struct {
	Targets []string `arg:"" help:"Paths to query."`
}

func (cmd *InfoQueryCmd) Run(parent *Cmd) error {
	ctx, roots, _, err := parent.LoadRoots()
	if err != nil {
		return err
	}
	results, err := dump.Query(ctx, roots, cmd.Targets)
	if err != nil {
		return err
	}
	for _, r := range results {
		fmt.Printf("%s -> %s\n", r.Target, r.Resolved)
	}
	return nil
}

// sour assets info hash
type InfoHashCmd struct {
	Targets []string `arg:"" help:"Paths to hash."`
}

func (cmd *InfoHashCmd) Run(parent *Cmd) error {
	ctx, roots, _, err := parent.LoadRoots()
	if err != nil {
		return err
	}
	hash, err := dump.Hash(ctx, roots, cmd.Targets)
	if err != nil {
		return err
	}
	fmt.Print(hash)
	return nil
}

// sour assets info index
type InfoIndexCmd struct {
	File string `arg:"" help:"Path to .index.source file."`
}

func (cmd *InfoIndexCmd) Run() error {
	data, err := os.ReadFile(cmd.File)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cmd.File, err)
	}
	var index pkgassets.Index
	if err := cbor.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("decoding CBOR: %w", err)
	}
	jsonData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// sour assets info modes
type InfoModesCmd struct {
	File string `arg:"" help:"Path to .ogz file."`
}

func (cmd *InfoModesCmd) Run() error {
	modes, err := dump.DeriveGameModes(cmd.File)
	if err != nil {
		return err
	}
	out, _ := json.Marshal(modes)
	fmt.Println(string(out))
	return nil
}
