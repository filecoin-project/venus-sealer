package main

import (
	"fmt"
	"os"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/lib/blockstore"
	"github.com/filecoin-project/venus-sealer/types"
	builtin_actors "github.com/filecoin-project/venus/builtin-actors"
	"github.com/filecoin-project/venus/venus-shared/actors"

	"github.com/filecoin-project/go-address"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"

	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/tracing"

	panicreporter "github.com/filecoin-project/venus-sealer/app/panic-reporter"
)

var log = logging.Logger("main")

func main() {
	sealer.SetupLogLevels()

	local := []*cli.Command{
		logCmd, initCmd, runCmd, pprofCmd, sectorsCmd, dealsCmd, actorCmd, infoCmd, sealingCmd, storageCmd, messagerCmds, provingCmd, stopCmd, versionCmd, tokenCmd, fetchParamCmd,
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

			networkName := types.NetworkName(cctx.String("network"))
			if cctx.Command.Name != "init" {
				nodeApi, ncloser, err := api.GetFullNodeAPIV2(cctx)
				if err != nil {
					return xerrors.Errorf("getting full node api: %w", err)
				}
				defer ncloser()

				networkName, err = nodeApi.StateNetworkName(cctx.Context)
				if err != nil {
					return err
				}
			}

			if err := builtin_actors.LoadActorsFromCar(convertNetworkNameToNetworkType(string(networkName))); err != nil {
				return err
			}

			// preload manifest so that we have the correct code CID inventory for cli since that doesn't
			// go through CI
			if len(builtin_actors.BuiltinActorsV8Bundle()) > 0 {
				bs := blockstore.NewMemory()

				if err := actors.LoadManifestFromBundle(cctx.Context, bs, actors.Version8, builtin_actors.BuiltinActorsV8Bundle()); err != nil {
					panic(fmt.Errorf("error loading actor manifest: %w", err))
				}
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
				Usage:   "specify other actor to query / manipulate",
				Aliases: []string{"a"},
			},
			&cli.StringFlag{
				Name:  "network",
				Usage: "network type: one of mainnet,butterfly,calibration,2k,force, Default: mainnet",
			},
			&cli.BoolFlag{
				// examined in the Before above
				Name:        "color",
				Usage:       "use color in display output",
				DefaultText: "depends on output being a TTY",
			},
			&cli.StringFlag{
				Name:    "panic-reports",
				EnvVars: []string{"VENUS_PANIC_REPORT_PATH"},
				Hidden:  true,
				Value:   "~/.venussealer", // should follow --repo default
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"VENUS_SEALER_PATH"},
				Hidden:  false,
				Value:   "~/.venussealer", // TODO: Consider XDG_DATA_HOME
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
		After: func(c *cli.Context) error {
			if r := recover(); r != nil {
				// Generate report in LOTUS_PATH and re-raise panic
				panicreporter.GeneratePanicReport(c.String("panic-reports"), "repo", c.App.Name)
				panic(r)
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
