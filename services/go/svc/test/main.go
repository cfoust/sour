package main

import (
	"context"
	"log"
	"github.com/cfoust/sour/pkg/marshal"
	"os"
	"os/signal"
)

func main() {
	marshaller := marshal.NewMarshaller(
		"../server/qserv",
		50000,
		51000,
	)

	for i := 0; i < 3; i++ {
		_, err := marshaller.NewServer(context.Background())
		if err != nil {
			log.Fatal(err)
		}

	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	select {
	case sig := <-sigs:
		log.Printf("terminating: %v", sig)
	}

	marshaller.Shutdown()
}
