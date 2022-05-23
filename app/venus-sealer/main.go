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
	"github.com/filecoin-project/venus-sealer/api"
	panicreporter "github.com/filecoin-project/venus-sealer/app/panic-reporter"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/blockstore"
	"github.com/filecoin-project/venus-sealer/lib/tracing"
	"github.com/filecoin-project/venus-sealer/types"
	builtinactors "github.com/filecoin-project/venus/venus-shared/builtin-actors"
	vtypes "github.com/filecoin-project/venus/venus-shared/types"
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
				if err := originBefore(cctx); err != nil {
					return err
				}
			}
			return loadActorsWithCmdBefore(cctx)
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

var loadActorsWithCmdBefore = func(cctx *cli.Context) error {
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
	nt, err := networkNameToNetworkType(networkName)
	if err != nil {
		return err
	}
	builtinactors.SetNetworkBundle(nt)
	if err := os.Setenv(builtinactors.RepoPath, cctx.String("repo")); err != nil {
		return xerrors.Errorf("failed to set env %s", builtinactors.RepoPath)
	}

	bs := blockstore.NewMemory()
	if err := builtinactors.FetchAndLoadBundles(cctx.Context, bs, builtinactors.BuiltinActorReleases); err != nil {
		panic(fmt.Errorf("error loading actor manifest: %w", err))
	}

	return nil
}

func networkNameToNetworkType(networkName types.NetworkName) (vtypes.NetworkType, error) {
	switch networkName {
	case "":
		return vtypes.NetworkDefault, xerrors.Errorf("network name is empty")
	case "mainnet":
		return vtypes.NetworkMainnet, nil
	case "calibrationnet", "calibration":
		return vtypes.NetworkCalibnet, nil
	case "butterflynet", "butterfly":
		return vtypes.NetworkButterfly, nil
	case "interopnet", "interop":
		return vtypes.NetworkInterop, nil
	default:
		// include 2k force
		return vtypes.Network2k, nil
	}
}
