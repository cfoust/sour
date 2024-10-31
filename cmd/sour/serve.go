package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/pkg/config"
	"github.com/cfoust/sour/pkg/server/ingress"
	"github.com/cfoust/sour/pkg/server/servers"
	"github.com/cfoust/sour/pkg/server/service"
	"github.com/cfoust/sour/pkg/server/static"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func serve(configs []string) error {
	config, err := config.Process(CLI.Serve.Configs)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sour configuration, please specify one with the SOUR_CONFIG environment variable")
	}

	serverConfig := config.Server

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = log.Output(consoleWriter)

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
	numCfgMaps := 0
	for _, map_ := range maps {
		if map_.HasCFG {
			numCfgMaps++
		}
	}

	if len(maps) == 0 {
		return fmt.Errorf("no maps found")
	}

	log.Info().Msgf("loaded %d maps (%d no .cfg)", len(maps), len(maps)-numCfgMaps)

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
	go wsIngress.StartWatcher(ctx)

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
		mux.Handle("/assets/", http.StripPrefix(
			"/assets/",
			http.FileServer(
				http.Dir("../client/dist/assets"),
			),
		))

		errc <- http.ListenAndServe(
			fmt.Sprintf("0.0.0.0:%d", serverConfig.Ingress.Web.Port),
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
