package main

import (
	"context"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/mitchellh/go-homedir"
	"github.com/zbiljic/go-filelock"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"regexp"
	"syscall"

	mux "github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/api/impl"
	"github.com/filecoin-project/venus-sealer/lib/ulimit"
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
		if cctx.IsSet("data") {
			cfg.DataDir = cctx.String("data")
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
			sealer.Override(new(types.ShutdownChan), shutdownChan),
		)
		if err != nil {
			return xerrors.Errorf("creating node: %w", err)
		}

		endpoint, err := cfg.API.APIEndpoint()
		if err != nil {
			return xerrors.Errorf("getting API endpoint: %w", err)
		}

		lst, err := manet.Listen(endpoint)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		mux := mux.NewRouter()

		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", minerapi)

		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/remote").HandlerFunc(minerapi.(*impl.StorageMinerAPI).ServeRemote)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

		ah := &auth.Handler{
			Verify: minerapi.AuthVerify,
			Next:   mux.ServeHTTP,
		}

		srv := &http.Server{
			Handler: ah,
			BaseContext: func(listener net.Listener) context.Context {
				key, _ := tag.NewKey("api")
				ctx, _ := tag.New(context.Background(), tag.Upsert(key, "venus-sealer"))
				return ctx
			},
		}

		sigChan := make(chan os.Signal, 2)
		go func() {
			select {
			case sig := <-sigChan:
				log.Warnw("received shutdown", "signal", sig)
			case <-shutdownChan:
				log.Warn("received shutdown")
			}

			log.Warn("Shutting down...")
			if err := stop(context.TODO()); err != nil {
				log.Errorf("graceful shutting down failed: %s", err)
			}
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		return srv.Serve(manet.NetListener(lst))
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
