package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/repeale/fp-go/option"
	"github.com/rs/zerolog/log"
)

type Mod struct {
	Name   string
	Bundle string
}

type GameMap struct {
	Name        string
	Bundle      string
	Image       string
	Description string
	Aliases     []string
}

type Index struct {
	Maps []GameMap
	Mods []Mod
}

type AssetSource struct {
	Index *Index
	Base string
}

type MapFetcher struct {
	Sources []*AssetSource
}

func NewMapFetcher() *MapFetcher {
	return &MapFetcher{
		Sources: make([]*AssetSource, 0),
	}
}

func FetchIndex(url string) (*Index, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	buffer, err := io.ReadAll(resp.Body)

	var index Index
	err = json.Unmarshal(buffer, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func CleanSourcePath(indexURL string) string {
	lastSlash := strings.LastIndex(indexURL, "/")
	if lastSlash == -1 {
		return ""
	}

	return indexURL[:lastSlash + 1]
}

func (m *MapFetcher) FetchIndices(assetSources []string) error {
	sources := make([]*AssetSource, 0)

	for _, url := range assetSources {
		index, err := FetchIndex(url)
		if err != nil {
			return err
		}

		log.Info().Str("source", url).Msg("fetched asset index")
		sources = append(sources, &AssetSource{
			Index: index,
			Base: CleanSourcePath(url),
		})
	}

	m.Sources = sources

	return nil
}

// Attempt to resolve a map name
func (m *MapFetcher) FindMapURL(mapName string) opt.Option[string] {
	for _, source := range m.Sources {
		for _, gameMap := range source.Index.Maps {
			if gameMap.Name != mapName {
				continue
			}

			url := fmt.Sprintf("%s%s.ogz", source.Base, gameMap.Bundle)

			return opt.Some[string](url)
		}
	}

	return opt.None[string]()
}
