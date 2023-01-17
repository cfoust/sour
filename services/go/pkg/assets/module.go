package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/repeale/fp-go/option"
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

type FetchResult struct {
	Data []byte
	Err  error
}

type FetchJob struct {
	Asset  string
	Result chan FetchResult
}

type AssetFetcher struct {
	jobs    chan FetchJob
	sources []*AssetSource
	redis   *redis.Client
}

func NewAssetFetcher(redis *redis.Client) *AssetFetcher {
	return &AssetFetcher{
		sources: make([]*AssetSource, 0),
		jobs:    make(chan FetchJob),
		redis:   redis,
	}
}

func WriteBytes(data []byte, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	out, err = os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(data)
	if err != nil {
		return err
	}

	return nil
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

func (m *AssetFetcher) FetchIndices(assetSources []string) error {
	sources := make([]*AssetSource, 0)

	for _, url := range assetSources {
		index, err := FetchIndex(url)
		if err != nil {
			return err
		}

		sources = append(sources, &AssetSource{
			Index: index,
			Base:  CleanSourcePath(url),
		})
	}

	m.sources = sources

	return nil
}

func (m *AssetFetcher) GetAssetURL(id string) opt.Option[string] {
	for _, source := range m.sources {
		for _, asset := range source.Index.Assets {
			if asset == id {
				return opt.Some(fmt.Sprintf("%s%s", source.Base, asset))
			}
		}
	}

	return opt.None[string]()
}

var AssetMissing = fmt.Errorf("asset not found")

func DownloadBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

const (
	ASSET_KEY    = "assets-%s"
	ASSET_EXPIRY = time.Duration(1 * time.Hour)
)

func (m *AssetFetcher) getAsset(ctx context.Context, id string) ([]byte, error) {
	key := fmt.Sprintf(ASSET_KEY, id)
	data, err := m.redis.Get(ctx, key).Bytes()
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if err == nil {
		return data, nil
	}

	url := m.GetAssetURL(id)
	if opt.IsNone(url) {
		return nil, AssetMissing
	}

	data, err = DownloadBytes(url.Value)
	if err != nil {
		return nil, err
	}

	err = m.redis.Set(ctx, key, data, ASSET_EXPIRY).Err()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (m *AssetFetcher) PollDownloads(ctx context.Context) {
	for {
		select {
		case job := <-m.jobs:
			data, err := m.getAsset(ctx, job.Asset)
			job.Result <- FetchResult{
				Data: data,
				Err:  err,
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *AssetFetcher) FetchAsset(ctx context.Context, source *AssetSource, offset int) ([]byte, error) {
	resolved := source.ResolveAsset(offset)
	if opt.IsNone(resolved) {
		return nil, AssetMissing
	}

	out := make(chan FetchResult)
	m.jobs <- FetchJob{
		Asset:  resolved.Value,
		Result: out,
	}

	result := <-out
	return result.Data, result.Err
}

func (m *AssetFetcher) FetchMapBytes(ctx context.Context, needle string) ([]byte, error) {
	map_ := m.FindMap(needle)

	if opt.IsNone(map_) {
		return nil, AssetMissing
	}

	foundMap := map_.Value
	return m.FetchAsset(ctx, foundMap.Source, foundMap.Map.Ogz)
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

func (m *AssetFetcher) FindMap(needle string) opt.Option[FoundMap] {
	otherTarget := needle + ".ogz"
	for _, source := range m.sources {
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
