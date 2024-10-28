package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/cfoust/sour/pkg/assets"
	"github.com/cfoust/sour/svc/server/config"
	"github.com/cfoust/sour/svc/server/ingress"
	"github.com/cfoust/sour/svc/server/servers"
	"github.com/cfoust/sour/svc/server/service"
	"github.com/cfoust/sour/svc/server/static"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	debug := flag.Bool("debug", false, "Whether to enable debug logging.")
	flag.Parse()

	configJson, ok := os.LookupEnv("SOUR_CONFIG")
	if !ok {
		log.Fatal().Msg("SOUR_CONFIG not defined")
	}

	sourConfig, err := config.GetSourConfig([]byte(configJson))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sour configuration, please specify one with the SOUR_CONFIG environment variable")
	}

	clusterConfig := sourConfig.Cluster

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = log.Output(consoleWriter)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Warn().Msg("debug logging enabled")
	}

	clusterConfig.LogSessions = false

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cache assets.Store
	cacheDir := clusterConfig.CacheDirectory
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
		clusterConfig.Assets,
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
		log.Fatal().Msg("no maps found")
	}

	log.Info().Msgf("loaded %d maps (%d no .cfg)", len(maps), len(maps)-numCfgMaps)

	go assetFetcher.PollDownloads(ctx)

	serverManager := servers.NewServerManager(assetFetcher, clusterConfig.ServerDescription, clusterConfig.Presets)
	cluster := service.NewCluster(
		ctx,
		serverManager,
		assetFetcher,
		clusterConfig,
	)

	err = serverManager.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server manager")
	}

	newConnections := make(chan ingress.Connection)

	wsIngress := ingress.NewWSIngress(newConnections)

	enet := make([]*ingress.ENetIngress, 0)
	infoServices := make([]*servers.ServerInfoService, 0)

	cluster.StartServers(ctx)

	for _, enetConfig := range clusterConfig.Ingress.Desktop {
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

				if enetConfig.ServerInfo.Cluster {
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

	staticSite, err := static.Site(configJson)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load site data")
	}

	errc := make(chan error, 1)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", staticSite)
		mux.Handle("/ws/", wsIngress)
		mux.Handle("/api/", cluster)

		errc <- http.ListenAndServe(
			fmt.Sprintf("0.0.0.0:%d", clusterConfig.Ingress.Web.Port),
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
}
