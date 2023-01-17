package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"runtime"
	"time"

	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/auth"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/service"
	"github.com/cfoust/sour/svc/cluster/state"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	debug := flag.Bool("debug", false, "Whether to enable debug logging.")
	cpuProfile := flag.String("cpu", "", "Write cpu profile to `file`.")
	memProfile := flag.String("memory", "", "Write memory profile to `file`.")
	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create CPU profile")
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not start CPU profile")
		}
		defer pprof.StopCPUProfile()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Warn().Msg("debug logging enabled")
	}

	sourConfig, err := config.GetSourConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sour configuration, please specify one with the SOUR_CONFIG environment variable")
	}

	clusterConfig := sourConfig.Cluster

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := state.NewStateService(sourConfig.Redis)

	maps := assets.NewAssetFetcher(state.Client)
	go maps.PollDownloads(ctx)

	sender := service.NewMapSender(maps)
	err = maps.FetchIndices(clusterConfig.Assets)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load assets")
	}

	err = sender.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start map sender")
	}

	var discord *auth.DiscordService = nil
	discordSettings := sourConfig.Discord
	if discordSettings.Enabled {
		log.Info().Msg("Discord authentication enabled")
		discord = auth.NewDiscordService(
			discordSettings,
			state,
		)
	}

	serverManager := servers.NewServerManager(maps, clusterConfig.ServerDescription, clusterConfig.Presets)
	cluster := service.NewCluster(
		ctx,
		serverManager,
		maps,
		sender,
		clusterConfig,
		sourConfig.Discord.Domain,
		discord,
		state.Client,
	)

	err = serverManager.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server manager")
	}

	newConnections := make(chan ingress.Connection)

	wsIngress := ingress.NewWSIngress(newConnections, discord)

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

	errc := make(chan error, 1)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/", wsIngress)
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

	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create memory profile")
		}
		defer f.Close() // error handling omitted for example
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal().Err(err).Msg("could not write memory profile")
		}
	}

	for _, enetIngress := range enet {
		enetIngress.Shutdown()
	}
	for _, infoService := range infoServices {
		infoService.Shutdown()
	}
	cluster.Shutdown()
}
