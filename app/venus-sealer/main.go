package main

import (
	"fmt"
	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/tracing"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
	"os"
)

var log = logging.Logger("main")

const FlagMinerRepo = "miner-repo"

func main() {
	sealer.SetupLogLevels()

	local := []*cli.Command{
		initCmd, runCmd, pprofCmd, sectorsCmd, actorCmd, infoCmd, sealingCmd, storageCmd, messagerCmds, provingCmd, stopCmd, versionCmd,
	}
	jaeger := tracing.SetupJaegerTracing("venus-sealer")
	defer func() {
		if jaeger != nil {
			jaeger.Flush()
		}
	}()

	for _, cmd := range local {
		cmd := cmd
		originBefore := cmd.Before
		cmd.Before = func(cctx *cli.Context) error {
			trace.UnregisterExporter(jaeger)
			jaeger = tracing.SetupJaegerTracing("venus-sealer/" + cmd.Name)

			if originBefore != nil {
				return originBefore(cctx)
			}
			return nil
		}
	}

	app := &cli.App{
		Name:                 "venus-sealer",
		Usage:                "Filecoin decentralized storage network miner",
		Version:              constants.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "actor",
				Value:   "",
				Usage:   "specify other actor to check state for (read only)",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name: "color",
			},
			&cli.StringFlag{
				Name:    "data",
				EnvVars: []string{"VENUS_SEALER_PATH"},
				Hidden:  true,
				Value:   "~/.venussealer", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				EnvVars: []string{"VENUS_SEALER_CONFIG"},
				Hidden:  true,
				Value:   "~/.venussealer/config.toml", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"VENUS_PATH"},
				Hidden:  true,
				Value:   "~/.venus", // TODO: Consider XDG_DATA_HOME
			},
		},
		Commands: local,
	}
	app.Setup()

	RunApp(app)
}

func RunApp(app *cli.App) {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", err) // nolint:errcheck
		var phe *PrintHelpErr
		if xerrors.As(err, &phe) {
			_ = cli.ShowCommandHelp(phe.Ctx, phe.Ctx.Command.Name)
		}
		os.Exit(1)
	}
}
