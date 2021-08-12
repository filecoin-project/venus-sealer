package main

import (
	"fmt"
	"os"

	"github.com/filecoin-project/go-address"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"

	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/tracing"
)

var log = logging.Logger("main")

func main() {
	sealer.SetupLogLevels()

	local := []*cli.Command{
		initCmd, runCmd, pprofCmd, sectorsCmd, actorCmd, infoCmd, sealingCmd, storageCmd, messagerCmds, provingCmd, stopCmd, versionCmd, tokenCmd,
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
			&cli.StringFlag{
				Name:  "network",
				Usage: "network type: one of mainnet,calibration,2k&nerpa, Default: mainnet",
			},
			&cli.BoolFlag{
				Name: "color",
			},
			&cli.StringFlag{
				Name:    "repo",
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
		},
		Commands: local,
		Before: func(cctx *cli.Context) error {
			network := cctx.String("network")
			switch network {
			case "mainnet":
				constants.SetAddressNetwork(address.Mainnet)
			case "2k":
				constants.InsecurePoStValidation = true
			default:
				if network == "" {
					_ = cctx.Set("network", "mainnet")
					if _, ok := os.LookupEnv("VENUS_ADDRESS_TYPE"); !ok {
						constants.SetAddressNetwork(address.Mainnet)
					}
				}
			}

			return nil
		},
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
