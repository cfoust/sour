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

type IndexAsset struct {
	Id   int
	Path string
}

// https://eagain.net/articles/go-json-array-to-struct/
func (i *IndexAsset) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&i.Id, &i.Path}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number of fields in IndexAsset: %d != %d", g, e)
	}
	return nil
}

type Mod struct {
	Id          string
	Name        string
	Image       string
	Description string
}

type GameMap struct {
	Id          string
	Name        string
	Ogz         int
	Bundle      string
	Assets      []IndexAsset
	Image       string
	Description string
}

type Bundle struct {
	Id      string
	Desktop bool
	Web     bool
	Assets  []IndexAsset
}

type Model struct {
	Id   string
	Name string
}

type Index struct {
	Assets   []string
	Textures []IndexAsset
	Bundles  []Bundle
	Maps     []GameMap
	Models   []Model
	Mods     []Mod
}

type AssetSource struct {
	Index *Index
	Base  string
}

func (a *AssetSource) ResolveAsset(id int) opt.Option[string] {
	if a.Index == nil {
		return opt.None[string]()
	}

	assets := a.Index.Assets
	if id < 0 || id >= len(assets) {
		return opt.None[string]()
	}

	return opt.Some(assets[id])
}

func (a *AssetSource) ResolveBundle(id string) opt.Option[Bundle] {
	if a.Index == nil {
		return opt.None[Bundle]()
	}

	for _, bundle := range a.Index.Bundles {
		if bundle.Id != id {
			continue
		}

		return opt.Some(bundle)
	}

	return opt.None[Bundle]()
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

func (f *FoundMap) GetOGZURL() opt.Option[string] {
	if f.Source == nil || f.Map == nil {
		return opt.None[string]()
	}

	asset := f.Source.ResolveAsset(f.Map.Ogz)
	if opt.IsNone(asset) {
		return opt.None[string]()
	}

	return opt.Some(f.GetBaseURL() + asset.Value)
}

func (f *FoundMap) GetDesktopURL() opt.Option[string] {
	if f.Source == nil || f.Map == nil {
		return opt.None[string]()
	}

	bundle := f.Source.ResolveBundle(f.Map.Bundle)
	if opt.IsNone(bundle) || !bundle.Value.Desktop {
		return opt.None[string]()
	}

	return opt.Some(
		fmt.Sprintf(
			"%s%s.desktop",
			f.GetBaseURL(),
			bundle.Value.Id,
		),
	)
}

func (m *MapFetcher) FindMap(needle string) opt.Option[FoundMap] {
	otherTarget := needle + ".ogz"
	for _, source := range m.Sources {
		for _, gameMap := range source.Index.Maps {
			if gameMap.Name != needle && gameMap.Name != otherTarget && !strings.HasPrefix(gameMap.Id, needle) {
				continue
			}

			return opt.Some(FoundMap{
				Map:    &gameMap,
				Source: source,
			})
		}
	}

	return opt.None[FoundMap]()
}
