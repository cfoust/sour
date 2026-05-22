package packager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest describes what assets to build.
type Manifest struct {
	Roots    []string       `yaml:"roots"`
	Skip     string         `yaml:"skip"`
	Output   string         `yaml:"output"`
	Prefix   string         `yaml:"prefix"`
	Download bool           `yaml:"download"`
	Compress bool           `yaml:"compress"`
	Web      bool           `yaml:"web"`
	Desktop  bool           `yaml:"desktop"`
	Mods     []ManifestMod  `yaml:"mods"`
	Models   *PatternFilter `yaml:"models"`
	Textures *PatternFilter `yaml:"textures"`
	Maps     *PatternFilter `yaml:"maps"`
}

// ManifestMod defines a named mod bundle.
type ManifestMod struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Image       string         `yaml:"image"`
	Files       *PatternFilter `yaml:"files"`
	Models      *PatternFilter `yaml:"models"`
	Maps        *PatternFilter `yaml:"maps"`
	Textures    *PatternFilter `yaml:"textures"`
}

// PatternFilter selects items by include/exclude glob patterns.
type PatternFilter struct {
	Include PatternSet `yaml:"include"`
	Exclude PatternSet `yaml:"exclude"`
}

// PatternSet is a list of glob patterns. In YAML it can be a single
// string or a list of strings:
//
//	include: "*"
//	include: [dust2, complex]
//	include:
//	  - dust2
//	  - complex
type PatternSet []string

func (p *PatternSet) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		*p = PatternSet{value.Value}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		*p = PatternSet(list)
		return nil
	default:
		return fmt.Errorf("pattern must be a string or list of strings")
	}
}

// LoadManifest reads and parses a YAML manifest file.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &m, nil
}

// MergeRoots prepends CLI roots to the manifest's roots, skipping duplicates.
func (m *Manifest) MergeRoots(cliRoots []string) {
	if len(cliRoots) == 0 {
		return
	}
	seen := make(map[string]bool)
	for _, r := range m.Roots {
		seen[r] = true
	}
	for _, r := range cliRoots {
		if !seen[r] {
			m.Roots = append([]string{r}, m.Roots...)
			seen[r] = true
		}
	}
}

// MatchPattern checks whether name matches any of the given glob patterns.
func MatchPattern(patterns []string, name string) bool {
	for _, p := range patterns {
		if p == "*" {
			return true
		}
		if matched, _ := filepath.Match(p, name); matched {
			return true
		}
	}
	return false
}

// FilterStrings applies a PatternFilter to a list of names, returning
// those that match include and don't match exclude.
func FilterStrings(filter PatternFilter, items []string) []string {
	if len(filter.Include) == 0 {
		return nil
	}

	var result []string
	for _, item := range items {
		if !MatchPattern(filter.Include, item) {
			continue
		}
		if len(filter.Exclude) > 0 && MatchPattern(filter.Exclude, item) {
			continue
		}
		result = append(result, item)
	}
	return result
}

// EnumerateMapNames extracts map names from root file paths.
// e.g., "packages/base/complex.ogz" → "complex"
func EnumerateMapNames(files []string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, f := range files {
		if !strings.HasSuffix(f, ".ogz") {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(f), ".ogz")
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// EnumerateModelNames extracts model names from root file paths.
// A model directory is identified by either:
// 1. Having a cfg file (md5.cfg, md3.cfg, etc.) — a root model
// 2. Being a subdirectory of a model that contains textures — a skin variant
//    (e.g., armor/green has skin.jpg but no cfg, inherits armor/md3.cfg)
func EnumerateModelNames(files []string) []string {
	cfgNames := map[string]bool{
		"md2.cfg": true, "md3.cfg": true, "md5.cfg": true,
		"obj.cfg": true, "smd.cfg": true, "iqm.cfg": true,
	}

	// First pass: find all directories with model cfgs
	modelRoots := make(map[string]bool)
	// Track all directories that contain any file under packages/models/
	dirsWithFiles := make(map[string]bool)

	for _, f := range files {
		if !strings.HasPrefix(f, "packages/models/") {
			continue
		}

		rel := f[len("packages/models/"):]
		dir := filepath.Dir(rel)
		dirsWithFiles[dir] = true

		if cfgNames[filepath.Base(f)] {
			modelRoots[dir] = true
		}
	}

	seen := make(map[string]bool)
	var names []string

	// Add all model roots
	for name := range modelRoots {
		seen[name] = true
		names = append(names, name)
	}

	// Second pass: find variant directories (subdirs of model roots
	// that have files but no cfg of their own)
	for dir := range dirsWithFiles {
		if seen[dir] {
			continue
		}
		// Walk up to see if a parent is a model root
		parent := dir
		for parent != "." && parent != "" {
			parent = filepath.Dir(parent)
			if modelRoots[parent] {
				seen[dir] = true
				names = append(names, dir)
				break
			}
		}
	}

	return names
}

