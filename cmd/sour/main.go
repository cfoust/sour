package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cfoust/sour/pkg/config"
	"github.com/cfoust/sour/pkg/version"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var CLI struct {
	Version bool `help:"Print version information and exit." short:"v"`
	Debug   bool `help:"Whether to enable debug logging."`

	Serve struct {
		Configs []string `arg:"" optional:"" name:"configs" help:"Configuration files for the server." type:"file"`
	} `cmd:"" help:"Start the sour server."`

	Config struct {
	} `cmd:"" help:"Write Sour's default configuration to standard output."`
}

func writeError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func main() {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = log.Output(consoleWriter)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if len(os.Args) == 1 {
		err := serveCommand([]string{})
		if err != nil {
			writeError(err)
		}
		return
	}

	ctx := kong.Parse(&CLI,
		kong.Name("sour"),
		kong.Description("a modern Sauerbraten server"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}))

	if CLI.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Warn().Msg("debug logging enabled")
	}

	if CLI.Version {
		fmt.Printf(
			"sour %s (commit %s)\n",
			version.Version,
			version.GitCommit,
		)
		fmt.Printf(
			"built %s\n",
			version.BuildTime,
		)
		os.Exit(0)
	}

	switch ctx.Command() {
	case "serve":
		fallthrough
	case "serve <configs>":
		err := serveCommand(CLI.Serve.Configs)
		if err != nil {
			writeError(err)
		}
	case "config":
		os.Stdout.Write(config.DEFAULT)
	}
}
