package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	Base  string
}

type MapFetcher struct {
	Sources []*AssetSource
}

func NewMapFetcher() *MapFetcher {
	return &MapFetcher{
		Sources: make([]*AssetSource, 0),
	}
}

func DownloadFile(url string, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
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

	return indexURL[:lastSlash+1]
}

func GetURLBase(url string) string {
	lastSlash := strings.LastIndex(url, "/")
	if lastSlash == -1 {
		return ""
	}

	return url[lastSlash+1:]
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
			Base:  CleanSourcePath(url),
		})
	}

	m.Sources = sources

	return nil
}

type FoundMap struct {
	Map    *GameMap
	Source *AssetSource
}

func (f *FoundMap) GetBaseURL() string {
	return fmt.Sprintf("%s%s", f.Source.Base, f.Map.Bundle)
}

func (f *FoundMap) GetOGZURL() string {
	return f.GetBaseURL() + ".ogz"
}

func (f *FoundMap) GetDesktopURL() string {
	return f.GetBaseURL() + ".desktop"
}

func (m *MapFetcher) FindMap(mapName string) opt.Option[FoundMap] {
	for _, source := range m.Sources {
		for _, gameMap := range source.Index.Maps {
			if gameMap.Name != mapName {
				continue
			}

			return opt.Some[FoundMap](FoundMap{
				Map:    &gameMap,
				Source: source,
			})
		}
	}

	return opt.None[FoundMap]()
}
