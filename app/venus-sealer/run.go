package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path"
	"regexp"

	"github.com/mitchellh/go-homedir"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"github.com/zbiljic/go-filelock"
	"go.opencensus.io/stats/view"
	"golang.org/x/xerrors"

	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/ulimit"
	"github.com/filecoin-project/venus-sealer/types"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a venus sealer process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "miner-api",
			Usage: "2345",
		},
		&cli.BoolFlag{
			Name:  "enable-gpu-proving",
			Usage: "enable use of GPU for mining operations",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.BoolFlag{
			Name:  "manage-fdlimit",
			Usage: "manage open file limit",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("enable-gpu-proving") {
			err := os.Setenv("BELLMAN_NO_GPU", "true")
			if err != nil {
				return err
			}
		}

		if cctx.Bool("manage-fdlimit") {
			if _, _, err := ulimit.ManageFdLimit(); err != nil {
				log.Errorf("setting file descriptor limit: %s", err)
			}
		}

		//read config
		cfgPath := cctx.String("config")
		cfg, err := config.MinerFromFile(cfgPath)
		if err != nil {
			return err
		}

		cfg.ConfigPath = cfgPath
		if cctx.IsSet("repo") {
			cfg.DataDir = cctx.String("repo")
		}

		//lock repo
		dataDir, err := homedir.Expand(cfg.DataDir)
		if err != nil {
			return err
		}
		fl, err := filelock.New(path.Join(dataDir, "repo.lock"))
		if err != nil {
			return err
		}
		err = fl.Lock()
		if err != nil {
			return err
		}
		defer fl.Unlock()

		nodeApi, ncloser, err := api.GetFullNodeAPIV2(cctx)
		if err != nil {
			return xerrors.Errorf("getting full node api: %w", err)
		}
		defer ncloser()
		ctx := api.DaemonContext(cctx)

		// Register all metric views
		if err := view.Register(); err != nil {
			log.Fatalf("Cannot register the view: %v", err)
		}

		if err := checkV1ApiSupport(nodeApi); err != nil {
			return err
		}

		log.Info("Checking full node sync status")
		if !cctx.Bool("nosync") {
			if err := api.SyncWait(ctx, nodeApi, cfg.NetParams.BlockDelaySecs, false); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		shutdownChan := make(chan struct{})

		var minerapi api.StorageMiner
		stop, err := sealer.New(ctx,
			sealer.ConfigStorageAPIImpl(&minerapi),
			sealer.Override(new(types.ShutdownChan), shutdownChan),
			sealer.Repo(cfg),
			sealer.Online(cfg),
			sealer.ApplyIf(func(s *sealer.Settings) bool { return cctx.IsSet("miner-api") },
				sealer.Override(new(types.APIEndpoint), func() (types.APIEndpoint, error) {
					regex, _ := regexp.Compile(`tcp/\d*`)
					newAddr := regex.ReplaceAll([]byte(cfg.API.ListenAddress), []byte("tcp/"+cctx.String("miner-api")))
					cfg.API.ListenAddress = string(newAddr)
					return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
				})),
			sealer.Override(new(api.FullNode), nodeApi),
		)
		if err != nil {
			return xerrors.Errorf("creating node: %w", err)
		}

		endpoint, err := cfg.API.APIEndpoint()
		if err != nil {
			return xerrors.Errorf("getting API endpoint: %w", err)
		}

		// Instantiate the miner node handler.
		handler, err := sealer.MinerHandler(minerapi, true)
		if err != nil {
			return xerrors.Errorf("failed to instantiate rpc handler: %w", err)
		}

		// Serve the RPC.
		rpcStopper, err := sealer.ServeRPC(handler, "venus-miner", endpoint)
		if err != nil {
			return fmt.Errorf("failed to start json-rpc endpoint: %s", err)
		}

		// Monitor for shutdown.
		finishCh := MonitorShutdown(shutdownChan,
			ShutdownHandler{Component: "rpc server", StopFunc: rpcStopper},
			ShutdownHandler{Component: "miner", StopFunc: stop},
		)

		<-finishCh
		return nil
	},
}

func checkV1ApiSupport(nodeApi api.FullNode) error {
	v, err := nodeApi.Version(context.Background())

	if err != nil {
		return err
	}

	if !v.APIVersion.EqMajorMinor(constants.FullAPIVersion1) {
		return xerrors.Errorf("Remote API version didn't match (expected %s, remote %s)", constants.FullAPIVersion0, v.APIVersion)
	}

	log.Infof("Remote version %s", v)
	return nil
}
