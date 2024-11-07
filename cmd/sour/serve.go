package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/cfoust/sour/pkg/assets"
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

	// Find all of the directories we need to map into the client
	var fsRoots []string
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

			fsRoots = append(
				fsRoots,
				filepath.Dir(source),
			)

			config.Client.Assets = append(
				config.Client.Assets,
				fmt.Sprintf(
					"#origin/assets/%d/%s",
					len(fsRoots)-1,
					filepath.Base(source),
				),
			)
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

	staticSite, err := static.Site(string(clientConfig))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load site data")
	}

	errc := make(chan error, 1)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", staticSite)
		mux.Handle("/ws/", wsIngress)
		mux.Handle("/api/", cluster)

		for i, dir := range fsRoots {
			log.Info().Msgf("serving: %s -> /assets/%d", dir, i)
			prefix := fmt.Sprintf("/assets/%d/", i)

			handler := http.FileServer(http.Dir(dir))
			handler = http.StripPrefix(
				prefix,
				handler,
			)
			handler = SkipIndex(handler)
			mux.Handle(prefix, handler)
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
