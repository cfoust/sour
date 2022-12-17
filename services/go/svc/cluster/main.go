package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/cfoust/sour/svc/cluster/assets"
	"github.com/cfoust/sour/svc/cluster/config"
	"github.com/cfoust/sour/svc/cluster/ingress"
	"github.com/cfoust/sour/svc/cluster/servers"
	"github.com/cfoust/sour/svc/cluster/service"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	DEBUG = false
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if DEBUG {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	sourConfig, err := config.GetSourConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load sour configuration, please specify one with the SOUR_CONFIG environment variable")
	}

	clusterConfig := sourConfig.Cluster

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maps := assets.NewMapFetcher()
	err = maps.FetchIndices(clusterConfig.Assets)

	if err != nil {
		log.Fatal().Err(err).Msg("failed to load assets")
	}

	serverManager := servers.NewServerManager(maps, clusterConfig.ServerDescription, clusterConfig.Presets)
	cluster := service.NewCluster(ctx, serverManager, clusterConfig)

	err = serverManager.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to start server manager")
	}

	wsIngress := ingress.NewWSIngress(cluster.Clients)

	enet := make([]*ingress.ENetIngress, 0)
	for _, enetConfig := range clusterConfig.Ingress.Desktop {
		enetIngress := ingress.NewENetIngress(cluster.Clients)
		enetIngress.Serve(enetConfig.Port)
		enetIngress.InitialCommand = enetConfig.Command
		go enetIngress.Poll(ctx)
		enet = append(enet, enetIngress)
	}

	go cluster.StartServers(ctx)
	go cluster.PollClients(ctx)

	errc := make(chan error, 1)
	go func() {
		errc <- wsIngress.Serve(ctx, clusterConfig.Ingress.Web.Port)
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

	wsIngress.Shutdown(ctx)
	for _, enetIngress := range enet {
		enetIngress.Shutdown()
	}
	cluster.Shutdown()
}
