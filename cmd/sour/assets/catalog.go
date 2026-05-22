package assets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// CatalogCmd generates catalog files.
type CatalogCmd struct {
	Quad QuadCatalogCmd `cmd:"" help:"Generate Quadropolis catalog from nodes.json."`
}

// QuadCatalogCmd generates a catalog.json from Quadropolis metadata.
type QuadCatalogCmd struct {
	Nodes   string `help:"Path to nodes.json." required:""`
	DB      string `help:"Path to quadropolis db/ directory." required:""`
	NodeMap string `help:"Path to node_map.json." required:"" name:"node-map"`
	Outdir  string `help:"Output directory." default:"catalogs/quadropolis"`
}

func (cmd *QuadCatalogCmd) Run() error {
	nodesData, err := os.ReadFile(cmd.Nodes)
	if err != nil {
		return fmt.Errorf("reading nodes.json: %w", err)
	}

	var nodes []quadNode
	if err := json.Unmarshal(nodesData, &nodes); err != nil {
		return fmt.Errorf("parsing nodes.json: %w", err)
	}

	nodeMapData, err := os.ReadFile(cmd.NodeMap)
	if err != nil {
		return fmt.Errorf("reading node_map.json: %w", err)
	}

	// node_map.json maps string node IDs to list of map names
	var nodeMapRaw map[string][]string
	if err := json.Unmarshal(nodeMapData, &nodeMapRaw); err != nil {
		return fmt.Errorf("parsing node_map.json: %w", err)
	}

	// Build node lookup by ID
	nodesByID := make(map[int]quadNode)
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	os.MkdirAll(cmd.Outdir, 0755)

	catalogMaps := make(map[string]interface{})

	for nodeIDStr, mapNames := range nodeMapRaw {
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			continue
		}

		node, ok := nodesByID[nodeID]
		if !ok {
			continue
		}

		author, date := parseAuthorDate(node.Author)
		description := parseDescription(node.Content)

		// Find and copy screenshot
		var image string
		for _, file := range node.Files {
			name := file.Name
			if name == "" {
				continue
			}
			if !strings.HasSuffix(strings.ToLower(name), ".jpg") && !strings.HasSuffix(strings.ToLower(name), ".png") {
				continue
			}

			src := filepath.Join(cmd.DB, file.Hash)
			if _, err := os.Stat(src); err != nil {
				continue
			}

			ext := filepath.Ext(name)
			destName := file.Hash + ext
			dest := filepath.Join(cmd.Outdir, destName)
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				data, err := os.ReadFile(src)
				if err != nil {
					continue
				}
				os.WriteFile(dest, data, 0644)
			}
			image = destName
			break
		}

		for _, mapName := range mapNames {
			entry := make(map[string]interface{})
			if author != "" {
				entry["author"] = author
			}
			if date != "" {
				entry["date"] = date
			}
			if description != "" {
				entry["description"] = description
			}
			if image != "" {
				entry["image"] = image
			}
			catalogMaps[mapName] = entry
		}
	}

	catalog := map[string]interface{}{"maps": catalogMaps}
	catalogJSON, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}

	outPath := filepath.Join(cmd.Outdir, "catalog.json")
	if err := os.WriteFile(outPath, catalogJSON, 0644); err != nil {
		return err
	}

	log.Info().Msgf("wrote %d map entries to %s", len(catalogMaps), outPath)
	return nil
}

// parseAuthorDate splits "Author | YYYY-MM-DD HH:MM" into (author, date).
func parseAuthorDate(field string) (string, string) {
	parts := strings.SplitN(field, " | ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(field), ""
}

// parseDescription extracts description from node content, skipping pipe-delimited metadata.
func parseDescription(content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) == 0 {
		return ""
	}

	// If first line looks like pipe-delimited metadata, skip it
	if strings.Contains(lines[0], "|") {
		lines = lines[1:]
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}
