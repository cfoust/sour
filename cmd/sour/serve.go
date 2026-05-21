package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/assets/packager"
	"github.com/cfoust/sour/pkg/catalog"
	"github.com/cfoust/sour/pkg/config"
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/server/servers"
	"github.com/cfoust/sour/pkg/server/service"
	"github.com/cfoust/sour/pkg/server/static"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func serveCommand(configs []string) error {
	config, err := config.Process(CLI.Serve.Configs)
	if err != nil {
		return err
	}

	serverConfig := config.Server

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if CLI.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Warn().Msg("debug logging enabled")
	}

	serverConfig.LogSessions = false

	ctx := context.Background()

	var cache assets.Store
	cacheDir := serverConfig.CacheDirectory
	if cacheDir != "" {
		err = os.MkdirAll(cacheDir, 0755)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to make cache dir: %s", cacheDir)
		}
		cache = assets.FSStore(cacheDir)
	}
	if cache == nil {
		log.Fatal().Msg("no cache directory specified")
	}

	// Check for assets installed via homebrew
	if homebrew, ok := os.LookupEnv("HOMEBREW_PREFIX"); ok {
		source := filepath.Join(
			homebrew,
			"share/sour/assets/.index.source",
		)

		if _, err := os.Stat(source); err == nil {
			serverConfig.Assets = append(
				serverConfig.Assets,
				"fs:"+source,
			)
		}
	}

	// Also check the current directory
	if current, err := os.Getwd(); err == nil {
		source := filepath.Join(
			current,
			"assets",
			".index.source",
		)

		if _, err := os.Stat(source); err == nil {
			serverConfig.Assets = append(
				serverConfig.Assets,
				"fs:"+source,
			)
		}
	}

	// Collect all asset roots to serve to the client.
	// Each root gets an /assets/N/ HTTP endpoint.
	type servedRoot struct {
		packaged  *assets.PackagedRoot
		fsDir     string // non-empty for filesystem-backed roots
		indexName string // filename of the index (e.g. ".index.source")
	}
	var servedRoots []servedRoot
	{
		roots, err := assets.LoadRoots(
			ctx,
			cache,
			serverConfig.Assets,
			false,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to load roots: %w",
				err,
			)
		}

		for _, root := range roots {
			packaged, ok := root.(*assets.PackagedRoot)
			if !ok {
				continue
			}

			if !packaged.IsFS() {
				continue
			}

			source := packaged.Source()
			idx := len(servedRoots)
			servedRoots = append(servedRoots, servedRoot{
				packaged:  packaged,
				fsDir:     filepath.Dir(source),
				indexName: filepath.Base(source),
			})

			config.Client.Assets = append(
				config.Client.Assets,
				fmt.Sprintf(
					"#origin/assets/%d/%s",
					idx,
					filepath.Base(source),
				),
			)
		}
	}

	// Load and merge catalog sources
	var catalogFSDirs []string
	var mergedCatalog *catalog.ResolvedCatalog
	if len(serverConfig.Catalogs) > 0 {
		var sources []catalog.Source
		for _, src := range serverConfig.Catalogs {
			if strings.HasPrefix(src, "fs:") {
				catPath := src[3:]
				absPath, err := filepath.Abs(catPath)
				if err != nil {
					log.Warn().Err(err).Msgf("failed to resolve catalog path: %s", catPath)
					continue
				}
				cat, err := catalog.LoadCatalog(absPath)
				if err != nil {
					log.Warn().Err(err).Msgf("failed to load catalog: %s", absPath)
					continue
				}
				dir := filepath.Dir(absPath)
				catalogFSDirs = append(catalogFSDirs, dir)
				sources = append(sources, catalog.Source{
					Catalog: cat,
					BaseURL: fmt.Sprintf("/catalog/%d", len(catalogFSDirs)-1),
				})
			}
			// HTTP catalog sources would be fetched here
		}

		if len(sources) > 0 {
			mergedCatalog = catalog.MergeCatalogs(sources)
			config.Client.Catalog = "#origin/catalog.json"
			log.Info().Msgf("merged %d catalog source(s)", len(sources))
		}
	}

	// Build maps from raw directories (in memory)
	var mapDirBuilder *packager.MemPackager
	if len(serverConfig.MapDirs) > 0 {
		var err error
		mapDirBuilder, err = packager.BuildMapDirs(ctx, serverConfig.MapDirs, serverConfig.Assets...)
		if err != nil {
			log.Warn().Err(err).Msg("failed to build map dirs")
		} else if len(mapDirBuilder.Maps) > 0 {
			// Add built maps to catalog
			if mergedCatalog == nil {
				mergedCatalog = &catalog.ResolvedCatalog{
					Maps: make(map[string]catalog.ResolvedMapEntry),
				}
				config.Client.Catalog = "#origin/catalog.json"
			}
			for _, m := range mapDirBuilder.Maps {
				mergedCatalog.Maps[m.Name] = catalog.ResolvedMapEntry{
					Description: m.Description,
				}
			}
			log.Info().Msgf("built %d maps from directories in memory", len(mapDirBuilder.Maps))
		}
	}

	assetFetcher, err := assets.NewAssetFetcher(
		ctx,
		cache,
		serverConfig.Assets,
		true,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("asset fetcher failed to initialize")
	}

	// Add in-memory built maps to the asset fetcher
	if mapDirBuilder != nil && len(mapDirBuilder.Maps) > 0 {
		memReader, readerErr := packager.NewMemReader(mapDirBuilder)
		if readerErr != nil {
			log.Warn().Err(readerErr).Msg("failed to create in-memory reader")
		} else {
			memRoot, err := assets.NewPackagedRoot(ctx, memReader, "", false)
			if err != nil {
				log.Warn().Err(err).Msg("failed to create in-memory asset root")
			} else {
				assetFetcher.AddRoot(memRoot)

				idx := len(servedRoots)
				servedRoots = append(servedRoots, servedRoot{
					packaged:  memRoot,
					indexName: ".index.source",
				})
				config.Client.Assets = append(
					config.Client.Assets,
					fmt.Sprintf("#origin/assets/%d/.index.source", idx),
				)
			}
		}
	}

	maps := assetFetcher.GetMaps("")
	uniqueMaps := make(map[string]struct{})
	for _, map_ := range maps {
		if len(map_.Name) == 0 {
			continue
		}

		if _, ok := uniqueMaps[map_.Name]; ok {
			continue
		}

		uniqueMaps[map_.Name] = struct{}{}
	}

	if len(maps) == 0 {
		return fmt.Errorf("no maps found")
	}

	log.Info().Msgf("loaded %d maps", len(uniqueMaps))

	// Now that we know which maps exist, finalize the catalog
	var mergedCatalogJSON []byte
	if mergedCatalog != nil {
		mergedCatalog.FilterAvailable(uniqueMaps)
		mergedCatalogJSON, err = json.Marshal(mergedCatalog)
		if err != nil {
			log.Warn().Err(err).Msg("failed to marshal merged catalog")
			mergedCatalogJSON = nil
		}
	}

	go assetFetcher.PollDownloads(ctx)

	serverManager := servers.NewServerManager(
		assetFetcher,
		serverConfig.ServerDescription,
		serverConfig.Presets,
	)
	cluster := service.NewCluster(
		ctx,
		serverManager,
		assetFetcher,
		serverConfig,
	)

	err = serverManager.Start()
	if err != nil {
		return err
	}

	newConnections := make(chan ingress.Connection)
	wsIngress := ingress.NewWSIngress(newConnections)
	wsIngress.SetClusterLister(func() []ingress.ClusterServerInfo {
		var result []ingress.ClusterServerInfo
		for _, s := range serverManager.ListServers() {
			result = append(result, ingress.ClusterServerInfo{
				Name:           s.Name,
				Map:            s.Map,
				Mode:           s.Mode,
				CurrentPlayers: s.CurrentPlayers,
				MaxPlayers:     s.MaxPlayers,
			})
		}
		return result
	})
	enet := make([]*ingress.ENetIngress, 0)
	infoServices := make([]*servers.ServerInfoService, 0)
	cluster.StartServers(ctx)
	for _, enetConfig := range serverConfig.Ingress.Desktop {
		enetIngress := ingress.NewENetIngress(newConnections)
		enetIngress.Serve(enetConfig.Port)
		enetIngress.InitialCommand = fmt.Sprintf("join %s", enetConfig.Target)
		go enetIngress.Poll(ctx)

		log.Info().
			Str("type", "desktop").
			Msgf(
				"listening on udp:0.0.0.0:%d",
				enetConfig.Port,
			)

		if enetConfig.ServerInfo.Enabled {
			serverManager.Mutex.Lock()
			for _, server := range serverManager.Servers {
				if server.Reference() != enetConfig.Target {
					continue
				}

				serverInfo := servers.NewServerInfoService(server)

				if enetConfig.ServerInfo.Server {
					serverInfo = servers.NewServerInfoService(cluster)
				}

				err := serverInfo.Serve(ctx, enetConfig.Port+1, enetConfig.ServerInfo.Master)
				if err != nil {
					log.Fatal().Err(err).Msg("failed to start server info service")
				}
				infoServices = append(infoServices, serverInfo)
			}
			serverManager.Mutex.Unlock()
		}

		enet = append(enet, enetIngress)
	}
	go cluster.PollUsers(ctx, newConnections)
	go cluster.PollDuels(ctx)

	// Encode the client config as json
	clientConfig, err := json.Marshal(config.Client)
	if err != nil {
		return err
	}

	var siteHandler http.Handler
	if CLI.Serve.Dev {
		// Serve from the on-disk site directory so Parcel watch changes
		// are picked up without rebuilding the Go binary.
		siteDir := filepath.Join("pkg", "server", "static", "site")
		siteHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Intercept index.js to prepend config injection
			if r.URL.Path == "/index.js" {
				body, err := os.ReadFile(filepath.Join(siteDir, "index.js"))
				if err != nil {
					http.Error(w, "index.js not found — run 'yarn serve' in client/", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/javascript")
				fmt.Fprintf(w, "const INJECTED_SOUR_CONFIG = %s;\n", string(clientConfig))
				w.Write(body)
				return
			}
			http.FileServer(http.Dir(siteDir)).ServeHTTP(w, r)
		})
		log.Info().Msgf("dev mode: serving client from %s", siteDir)
	} else {
		var err error
		siteHandler, err = static.Site(string(clientConfig))
		if err != nil {
			log.Fatal().Err(err).Msg("failed to load site data")
		}
	}

	errc := make(chan error, 1)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", siteHandler)
		mux.Handle("/ws/", wsIngress)
		mux.Handle("/api/", cluster)

		// Serve the merged catalog JSON
		if mergedCatalogJSON != nil {
			catalogJSON := mergedCatalogJSON
			mux.HandleFunc("/catalog.json", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write(catalogJSON)
			})
		}

		// Serve catalog image directories
		for i, dir := range catalogFSDirs {
			log.Info().Msgf("serving catalog: %s -> /catalog/%d", dir, i)
			prefix := fmt.Sprintf("/catalog/%d/", i)
			handler := http.FileServer(http.Dir(dir))
			handler = http.StripPrefix(prefix, handler)
			mux.Handle(prefix, handler)
		}

		for i, sr := range servedRoots {
			prefix := fmt.Sprintf("/assets/%d/", i)

			if sr.fsDir != "" {
				// Filesystem-backed root: serve files directly
				log.Info().Msgf("serving: %s -> %s", sr.fsDir, prefix)
				handler := http.FileServer(http.Dir(sr.fsDir))
				handler = http.StripPrefix(prefix, handler)
				handler = SkipIndex(handler)
				mux.Handle(prefix, handler)
			} else {
				// In-memory root: serve via PackageReader
				log.Info().Msgf("serving in-memory assets -> %s", prefix)
				root := sr.packaged
				reader := root.Reader()
				mux.HandleFunc(prefix, func(w http.ResponseWriter, r *http.Request) {
					key := strings.TrimPrefix(r.URL.Path, prefix)
					var data []byte
					var err error
					if key == sr.indexName {
						data, err = reader.Index(r.Context())
					} else {
						data, err = reader.Read(r.Context(), key)
					}
					if err != nil {
						http.NotFound(w, r)
						return
					}
					w.Header().Set("Content-Type", "application/octet-stream")
					w.Write(data)
				})
			}
		}

		address := serverConfig.Ingress.Web.Address
		if CLI.Serve.Address != "" {
			address = CLI.Serve.Address
		}

		port := serverConfig.Ingress.Web.Port
		if CLI.Serve.Port != -1 {
			port = CLI.Serve.Port
		}

		host := fmt.Sprintf(
			"%s:%d",
			address,
			port,
		)

		log.Info().
			Str("type", "web").
			Msgf("listening on tcp:%s", host)

		errc <- http.ListenAndServe(
			host,
			mux,
		)
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	signal.Notify(sigs, os.Kill)

	select {
	case err := <-errc:
		log.Printf("failed to serve: %v", err)
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	for _, enetIngress := range enet {
		enetIngress.Shutdown()
	}
	for _, infoService := range infoServices {
		infoService.Shutdown()
	}
	cluster.Shutdown()

	return nil
}
