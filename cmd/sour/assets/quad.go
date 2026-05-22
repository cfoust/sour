package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	pkgassets "github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// QuadCmd builds Quadropolis assets.
type QuadCmd struct {
	Dry    bool     `help:"Dry run, just print what would be built."`
	Prefix string   `help:"Index file prefix." default:""`
	Outdir string   `help:"Output directory." default:"output/quad" env:"ASSET_OUTPUT_DIR"`
	Nodes  []string `arg:"" optional:"" help:"Specific node IDs to build."`
}

type quadFile struct {
	URL      string   `json:"url"`
	Hash     string   `json:"hash"`
	Name     string   `json:"name"`
	Contents []string `json:"contents"`
}

type quadNode struct {
	ID      int        `json:"id"`
	Title   string     `json:"title"`
	Author  string     `json:"author"`
	Image   *quadFile  `json:"image"`
	Content string     `json:"content"`
	Files   []quadFile `json:"files"`
}

type mapJob struct {
	roots   []string
	mapPath string // empty means the file itself is a map
}

type modJob struct {
	root string
}

var bannedSuffixes = []string{
	"native_server",
	"sauerbraten_unix",
	".exe",
	".bat",
	".py",
	".cgz",
}

func isValidMod(files []string) bool {
	for _, file := range files {
		for _, suffix := range bannedSuffixes {
			if strings.HasSuffix(file, suffix) {
				return false
			}
		}
	}
	return true
}

func getJobs(file quadFile) ([]mapJob, []modJob) {
	var maps []mapJob
	var mods []modJob

	contents := file.Contents

	if len(contents) == 0 && file.Name != "" {
		if strings.HasSuffix(file.Name, ".ogz") {
			maps = append(maps, mapJob{})
		}
		if strings.HasSuffix(file.Name, ".cfg") {
			mods = append(mods, modJob{})
		}
		return maps, mods
	}

	// Filter __MACOSX
	var filtered []string
	for _, c := range contents {
		if !strings.HasPrefix(c, "__MACOSX") {
			filtered = append(filtered, c)
		}
	}
	contents = filtered

	if len(contents) == 0 || file.Name == "" {
		return maps, mods
	}

	// Find data roots
	rootSet := make(map[string]bool)
	for _, entry := range contents {
		parts := strings.Split(entry, string(filepath.Separator))
		for i, part := range parts {
			if part != "data" && part != "packages" {
				continue
			}
			root := ""
			if i > 0 {
				root = filepath.Join(parts[:i]...)
			}
			rootSet[root] = true
		}
	}

	var roots []string
	for r := range rootSet {
		roots = append(roots, r)
	}

	// Find map files
	for _, c := range contents {
		if !strings.HasSuffix(c, ".ogz") {
			continue
		}
		r := roots
		if len(r) == 0 {
			r = []string{""}
		}
		maps = append(maps, mapJob{
			roots:   r,
			mapPath: c,
		})
	}

	for _, root := range roots {
		var rootFiles []string
		for _, c := range contents {
			if strings.HasPrefix(c, root) {
				rootFiles = append(rootFiles, c)
			}
		}
		if !isValidMod(rootFiles) {
			continue
		}
		mods = append(mods, modJob{root: root})
	}

	// Fallback
	if len(maps) == 0 && len(mods) == 0 {
		if !isValidMod(contents) {
			return maps, mods
		}
		mods = append(mods, modJob{})
	}

	return maps, mods
}

type quadBuildResult struct {
	nodeID     int
	mapNames   []string
	packager   *packager.Packager
	failedMaps []string
}

func buildQuadNode(
	ctx context.Context,
	params packager.BuildParams,
	outdir string,
	node quadNode,
	quadRoot string,
	roots []string,
) (*quadBuildResult, error) {
	p := packager.New(outdir)
	nodePrefix := strconv.Itoa(node.ID)

	// Find image
	var image string
	for i, file := range node.Files {
		name := file.Name
		if name == "" {
			continue
		}
		if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".png") {
			continue
		}
		imagePath := path.Join(nodePrefix, strconv.Itoa(i), name)
		img, err := p.BuildImage(ctx, params, imagePath)
		if err == nil && img != "" {
			image = img
		}
	}

	description := node.Content
	var failedMaps []string

	for i, file := range node.Files {
		fileDir := path.Join(nodePrefix, strconv.Itoa(i))
		fileRoot := fmt.Sprintf("%s@%s", quadRoot, fileDir)

		fileContents, err := packager.GetRootFiles(ctx, params.Roots)
		if err != nil {
			continue
		}
		_ = fileContents

		fileRootStrings := append(roots, fileRoot)
		cache := pkgassets.FSStore("cache/")
		fileAssetRoots, err := pkgassets.LoadRoots(ctx, cache, fileRootStrings, false)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to load roots for node %d file %d", node.ID, i)
			continue
		}

		fileParams := params
		fileParams.Roots = fileAssetRoots
		fileParams.RootStrings = fileRootStrings

		fileParamsNoSkip := fileParams
		fileParamsNoSkip.SkipRoot = ""

		maps, mods := getJobs(file)

		// Node 4405 had both map and mod, normally skip mods if maps found
		if len(maps) > 0 && len(mods) > 0 && node.ID != 4405 {
			mods = nil
		}

		// Handle non-archive files
		if file.Contents == nil {
			if len(mods) > 0 {
				mod := mods[0]
				_ = mod
				modFile := filepath.Base(file.Name)
				resolved, err := packager.QueryFiles(ctx, fileAssetRoots, []string{modFile})
				if err != nil || len(resolved) == 0 || resolved[0].From == "nil" {
					continue
				}
				p.BuildMod(ctx, fileParamsNoSkip, resolved, fmt.Sprintf("quad-%d", node.ID), description, image)
				continue
			}

			if len(maps) == 0 {
				continue
			}

			mapName := strings.TrimSuffix(filepath.Base(file.Name), filepath.Ext(file.Name))
			_, err := p.BuildMap(ctx, fileParams, filepath.Base(file.Name), mapName, description, image)
			if err != nil {
				failedMaps = append(failedMaps, fileDir+mapName)
				log.Warn().Err(err).Msgf("failed to build map id=%d map=%s", node.ID, file.Name)
			}
			continue
		}

		// Handle mods from archives
		for j, mod := range mods {
			contents := file.Contents
			var modFiles []string
			for _, c := range contents {
				if strings.HasPrefix(c, mod.root) {
					modFiles = append(modFiles, c)
				}
			}
			if len(modFiles) == 0 {
				continue
			}

			resolved, err := packager.QueryFiles(ctx, fileAssetRoots, modFiles)
			if err != nil {
				continue
			}

			name := fmt.Sprintf("quad-%d", node.ID)
			if len(mods) > 1 {
				name = fmt.Sprintf("quad-%d-%d", node.ID, j)
			}

			p.BuildMod(ctx, fileParamsNoSkip, resolved, name, description, image)
		}

		// Handle maps from archives
		for _, job := range maps {
			if job.mapPath == "" {
				continue
			}

			mapName := strings.TrimSuffix(filepath.Base(job.mapPath), filepath.Ext(job.mapPath))
			_, err := p.BuildMap(ctx, fileParams, job.mapPath, mapName, description, image)
			if err != nil {
				failedMaps = append(failedMaps, fileDir+job.mapPath)
				log.Warn().Err(err).Msgf("failed to build map id=%d map=%s", node.ID, job.mapPath)
			}
		}
	}

	var mapNames []string
	for _, m := range p.Maps {
		mapNames = append(mapNames, m.Name)
	}

	return &quadBuildResult{
		nodeID:     node.ID,
		mapNames:   mapNames,
		packager:   p,
		failedMaps: failedMaps,
	}, nil
}

func (cmd *QuadCmd) Run(parent *Cmd) error {
	ctx := context.Background()

	os.MkdirAll(cmd.Outdir, 0755)

	quadRoot := "https://static.sourga.me/quadropolis/4412/.index.source"
	roots := []string{
		"sour",
		"https://static.sourga.me/blobs/6481/.index.source",
		quadRoot,
	}

	cache := pkgassets.FSStore(parent.Cache)
	os.MkdirAll(parent.Cache, 0755)

	assetRoots, err := pkgassets.LoadRoots(ctx, cache, roots, false)
	if err != nil {
		return fmt.Errorf("failed to load roots: %w", err)
	}

	params := packager.BuildParams{
		Roots:          assetRoots,
		RootStrings:    roots,
		SkipRoot:       roots[1],
		CompressImages: false,
		DownloadAssets: false,
		BuildWeb:       false,
		BuildDesktop:   false,
	}

	// Load nodes.json
	nodesData, err := os.ReadFile("nodes.json")
	if err != nil {
		return fmt.Errorf("reading nodes.json: %w", err)
	}

	var allNodes []quadNode
	if err := json.Unmarshal(nodesData, &allNodes); err != nil {
		return fmt.Errorf("parsing nodes.json: %w", err)
	}

	// Filter by target node IDs if specified
	var targetIDs map[int]bool
	if len(cmd.Nodes) > 0 {
		targetIDs = make(map[int]bool)
		for _, n := range cmd.Nodes {
			id, err := strconv.Atoi(n)
			if err != nil {
				return fmt.Errorf("invalid node ID: %s", n)
			}
			targetIDs[id] = true
		}
	}

	var nodes []quadNode
	for i := len(allNodes) - 1; i >= 0; i-- {
		node := allNodes[i]
		if targetIDs != nil && !targetIDs[node.ID] {
			continue
		}
		nodes = append(nodes, node)
	}

	if cmd.Dry {
		for _, node := range nodes {
			fmt.Printf("node %d: %s\n", node.ID, node.Title)
			for _, file := range node.Files {
				maps, mods := getJobs(file)
				fmt.Printf("  file %s: %d maps, %d mods\n", file.Name, len(maps), len(mods))
			}
		}
		return nil
	}

	p := packager.New(cmd.Outdir)

	failures, err := os.Create("failures.txt")
	if err != nil {
		return err
	}
	defer failures.Close()

	nodeMap := make(map[int][]string)
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for _, node := range nodes {
		node := node
		g.Go(func() error {
			result, err := buildQuadNode(gctx, params, cmd.Outdir, node, quadRoot, roots)
			if err != nil {
				log.Warn().Err(err).Msgf("failed to build node %d", node.ID)
				return nil
			}

			mu.Lock()
			p.Merge(result.packager)
			if len(result.mapNames) > 0 {
				nodeMap[result.nodeID] = result.mapNames
			}
			for _, f := range result.failedMaps {
				failures.WriteString(f + "\n")
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	log.Info().Msgf("built %d mods and %d maps", len(p.Mods), len(p.Maps))

	// Write node_map.json
	nodeMapJSON, err := json.Marshal(nodeMap)
	if err != nil {
		return err
	}
	nodeMapPath := filepath.Join(cmd.Outdir, "node_map.json")
	if err := os.WriteFile(nodeMapPath, nodeMapJSON, 0644); err != nil {
		return err
	}
	log.Info().Msgf("wrote node_map.json with %d nodes", len(nodeMap))

	return p.DumpIndex(cmd.Prefix)
}
