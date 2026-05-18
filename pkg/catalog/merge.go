package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Source represents a loaded catalog with its base URL for resolving image hashes.
type Source struct {
	Catalog *Catalog
	BaseURL string // e.g. "/catalog/0/" or "https://cdn.example.com/quad/"
}

// LoadCatalog reads and parses a catalog JSON file from disk.
func LoadCatalog(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading catalog %s: %w", path, err)
	}

	var cat Catalog
	if err := json.Unmarshal(data, &cat); err != nil {
		return nil, fmt.Errorf("parsing catalog %s: %w", path, err)
	}

	return &cat, nil
}

// MergeCatalogs merges multiple catalog sources into a single ResolvedCatalog.
// Sources are applied in order; later sources win per field.
// Image/gif hashes are resolved to full URLs using each source's BaseURL.
func MergeCatalogs(sources []Source) *ResolvedCatalog {
	result := &ResolvedCatalog{
		Maps: make(map[string]ResolvedMapEntry),
		Mods: make(map[string]ResolvedModEntry),
	}

	for _, src := range sources {
		if src.Catalog == nil {
			continue
		}

		for name, entry := range src.Catalog.Maps {
			existing := result.Maps[name]

			if entry.Author != "" {
				existing.Author = entry.Author
			}
			if entry.Date != "" {
				existing.Date = entry.Date
			}
			if entry.Description != "" {
				existing.Description = entry.Description
			}
			if entry.Image != "" {
				existing.ImageURL = resolveURL(src.BaseURL, entry.Image)
			}
			if entry.Gif != "" {
				existing.GifURL = resolveURL(src.BaseURL, entry.Gif)
			}

			result.Maps[name] = existing
		}

		for name, entry := range src.Catalog.Mods {
			existing := result.Mods[name]

			if entry.Author != "" {
				existing.Author = entry.Author
			}
			if entry.Description != "" {
				existing.Description = entry.Description
			}
			if entry.Image != "" {
				existing.ImageURL = resolveURL(src.BaseURL, entry.Image)
			}

			result.Mods[name] = existing
		}
	}

	return result
}

// FilterAvailable removes catalog entries for maps that don't exist in the asset index.
func (c *ResolvedCatalog) FilterAvailable(mapNames map[string]struct{}) {
	for name := range c.Maps {
		if _, ok := mapNames[name]; !ok {
			delete(c.Maps, name)
		}
	}
}

func resolveURL(base, hash string) string {
	if hash == "" {
		return ""
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + hash
}
