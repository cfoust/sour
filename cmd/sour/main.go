package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/cfoust/sour/pkg/config"
	"github.com/cfoust/sour/pkg/version"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var CLI struct {
	Version bool   `help:"Print version information and exit." short:"v"`
	Debug   bool   `help:"Whether to enable debug logging."`
	CPU     string `help:"Save a CPU performance report to the given path." name:"perf-file" optional:"" default:""`
	Trace   string `help:"Save a trace report to the given path." name:"trace-file" optional:"" default:""`

	Serve struct {
		Address string   `optional:"" name:"address" help:"IP address the HTTP server will listen on." default:""`
		Port    int      `optional:"" name:"port" help:"TCP port the HTTP server will listen on. This overrides the value set in any configurations." default:"-1"`
		Configs []string `arg:"" optional:"" name:"configs" help:"Configuration files for the server." type:"file"`
	} `cmd:"" default:"withargs" help:"Start the sour server."`

	Config struct {
	} `cmd:"" help:"Write Sour's default configuration to standard output."`
}

func writeError(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	os.Exit(1)
}

func runCPUProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"unable to create %s: %s",
			CLI.CPU,
			err,
		)
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		return fmt.Errorf(
			"could not start CPU profile: %s",
			err,
		)
	}

	return nil
}

func runTraceProfile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf(
			"unable to create %s: %s",
			CLI.Trace,
			err,
		)
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		return fmt.Errorf(
			"could not start trace profile: %s",
			err,
		)
	}

	return nil
}

func main() {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log.Logger = log.Output(consoleWriter)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if len(os.Args) == 1 {
		CLI.Serve.Port = -1
		err := serveCommand([]string{})
		if err != nil {
			writeError(err)
		}
		return
	}

	if len(CLI.CPU) > 0 {
		err := runCPUProfile(CLI.CPU)
		if err != nil {
			writeError(err)
		}

		defer pprof.StopCPUProfile()
	}

	if len(CLI.Trace) > 0 {
		err := runTraceProfile(CLI.Trace)
		if err != nil {
			writeError(err)
		}

		defer trace.Stop()
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
