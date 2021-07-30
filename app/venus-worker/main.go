package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/constants"
	"github.com/filecoin-project/venus-sealer/lib/vfile"
	"github.com/filecoin-project/venus-sealer/models"
	"github.com/filecoin-project/venus-sealer/service"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/mitchellh/go-homedir"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	paramfetch "github.com/filecoin-project/go-paramfetch"

	sealer "github.com/filecoin-project/venus-sealer"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/lib/rpcenc"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
)

var log = logging.Logger("main")

func main() {
	sealer.SetupLogLevels()

	local := []*cli.Command{
		runCmd,
		infoCmd,
		storageCmd,
		setCmd,
		waitQuietCmd,
		tasksCmd,
	}

	app := &cli.App{
		Name:                 "venus-worker",
		Usage:                "Remote miner worker",
		Version:              constants.UserVersion(),
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"VENUS_WORKER_PATH"},
				Hidden:  true,
				Value:   "~/.venusworker", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				EnvVars: []string{"VENUS_WORKER_CONFIG"},
				Hidden:  true,
				Value:   "~/.venusworker/config.toml", // TODO: Consider XDG_DATA_HOME
			},
			&cli.BoolFlag{
				Name:  "enable-gpu-proving",
				Usage: "enable use of GPU for mining operations",
				Value: true,
			},
		},

		Commands: local,
	}
	app.Setup()

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start venus worker",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "listen",
			Usage: "host address and port the worker api will listen on",
			Value: "0.0.0.0:3456",
		},
		&cli.BoolFlag{
			Name:  "no-local-storage",
			Usage: "don't use storageminer repo for sector storage",
		},
		&cli.BoolFlag{
			Name:  "no-swap",
			Usage: "don't use swap",
			Value: false,
		},
		&cli.BoolFlag{
			Name:  "addpiece",
			Usage: "enable addpiece",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "precommit1",
			Usage: "enable precommit1 (32G sectors: 1 core, 128GiB Memory)",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "unseal",
			Usage: "enable unsealing (32G sectors: 1 core, 128GiB Memory)",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "precommit2",
			Usage: "enable precommit2 (32G sectors: all cores, 96GiB Memory)",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "commit",
			Usage: "enable commit (32G sectors: all cores or GPUs, 128GiB Memory + 64GiB swap)",
			Value: true,
		},
		&cli.IntFlag{
			Name:  "parallel-fetch-limit",
			Usage: "maximum fetch operations to run in parallel",
			Value: 5,
		},
		&cli.StringFlag{
			Name:  "miner-addr",
			Usage: "miner address to connect",
			Value: "127.0.0.1:2345",
		},
		&cli.StringFlag{
			Name:  "miner-token",
			Usage: "miner token to connect",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "timeout",
			Usage: "used when 'listen' is unspecified. must be a valid duration recognized by golang's time.ParseDuration function",
			Value: "30m",
		},
	},
	Action: func(cctx *cli.Context) error {
		log.Info("Starting venus worker")

		if !cctx.Bool("enable-gpu-proving") {
			if err := os.Setenv("BELLMAN_NO_GPU", "true"); err != nil {
				return xerrors.Errorf("could not set no-gpu env: %+v", err)
			}
		}

		// Open repo
		cfgPath := cctx.String("config")
		ok, err := config.ConfigExist(cfgPath)
		if err != nil {
			return err
		}
		var cfg *config.StorageWorker
		if !ok {
			cfg = config.GetDefaultWorkerConfig()
		} else {
			cfg, err = config.WorkerFromFile(cfgPath)
			if err != nil {
				return err
			}
		}

		cfg.ConfigPath = cfgPath
		err = flagData(cfg, cctx)
		if err != nil {
			return err
		}

		log.Infof("config: %v", *cfg)

		if !ok {
			// Create repo, generate config.toml file
			err = config.SaveConfig(cfg.ConfigPath, cfg)
			if err != nil {
				return err
			}
		}

		dataDir, err := homedir.Expand(cfg.DataDir)
		if err != nil {
			return err
		}

		// Check the legitimacy of the repo
		err = vfile.EnsureDir(dataDir)
		if err != nil {
			return err
		}

		// Connect to venus-sealer
		ctx := api.ReqContext(cctx)
		var (
			nodeApi api.StorageMiner
			closer  func()
		)
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			nodeApi, closer, err = api.GetStorageMinerAPI(cctx, api.StorageMinerUseHttp, api.StorageSealerAddr(cfg.Sealer.Url), api.StorageSealerToken(cfg.Sealer.Token))
			if err == nil {
				_, err = nodeApi.Version(ctx)
				if err == nil {
					break
				}
			}

			fmt.Printf("\r\x1b[0KConnecting to miner API ... (%s)", err)
			time.Sleep(time.Second * 2)
		}
		defer closer()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Register all metric views
		if err := view.Register(); err != nil {
			log.Fatalf("Cannot register the view: %v", err)
		}

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}
		if v.APIVersion != constants.MinerAPIVersion0 {
			return xerrors.Errorf("lotus-miner API version doesn't match: expected: %s", api.Version{APIVersion: constants.MinerAPIVersion0})
		}
		log.Infof("Remote version %s", v)

		// Check proof-params
		act, err := nodeApi.ActorAddress(ctx)
		if err != nil {
			return err
		}

		ssize, err := nodeApi.ActorSectorSize(ctx, act)
		if err != nil {
			return err
		}

		if cctx.Bool("commit") {
			ps, err := asset.Asset("fixtures/_assets/proof-params/parameters.json")
			if err != nil {
				return err
			}

			srs, err := asset.Asset("fixtures/_assets/proof-params/srs-inner-product.json")
			if err != nil {
				return err
			}

			if err := paramfetch.GetParams(ctx, ps, srs, uint64(ssize)); err != nil {
				return xerrors.Errorf("get params: %w", err)
			}
		}

		var taskTypes []types.TaskType
		taskTypes = append(taskTypes, types.TTFetch, types.TTCommit1, types.TTFinalize)
		if cctx.Bool("addpiece") {
			taskTypes = append(taskTypes, types.TTAddPiece)
		}
		if cctx.Bool("precommit1") {
			taskTypes = append(taskTypes, types.TTPreCommit1)
		}
		if cctx.Bool("unseal") {
			taskTypes = append(taskTypes, types.TTUnseal)
		}
		if cctx.Bool("precommit2") {
			taskTypes = append(taskTypes, types.TTPreCommit2)
		}
		if cctx.Bool("commit") {
			taskTypes = append(taskTypes, types.TTCommit2)
		}

		if len(taskTypes) == 0 {
			return xerrors.Errorf("no task types specified")
		}

		log.Infof("Acceptable task types: %v", taskTypes)

		localStorage := cfg.LocalStorage()
		_, err = localStorage.GetStorage()
		if !ok || err != nil {
			log.Infof("get storage err: %v", err)

			err = config.SaveConfig(cfgPath, cfg)
			if err != nil {
				return err
			}

			var localPaths []stores.LocalPath

			if !cctx.Bool("no-local-storage") {
				b, err := json.MarshalIndent(&stores.LocalStorageMeta{
					ID:       stores.ID(uuid.New().String()),
					Weight:   10,
					CanSeal:  true,
					CanStore: false,
				}, "", "  ")

				if err != nil {
					return xerrors.Errorf("marshaling storage config: %w", err)
				}

				if err := ioutil.WriteFile(filepath.Join(dataDir, "sectorstore.json"), b, 0644); err != nil {
					return xerrors.Errorf("persisting storage metadata (%s): %w", filepath.Join(cfg.DataDir, "sectorstore.json"), err)
				}

				localPaths = append(localPaths, stores.LocalPath{
					Path: dataDir,
				})
			}

			if err := localStorage.SetStorage(func(sc *stores.StorageConfig) {
				sc.StoragePaths = append(sc.StoragePaths, localPaths...)
			}); err != nil {
				return xerrors.Errorf("set storage config: %w", err)
			}
		}

		dbRepo, err := models.SetDataBase(config.HomeDir(dataDir), &cfg.DB)
		if err != nil {
			return err
		}
		err = dbRepo.AutoMigrate()
		if err != nil {
			return err
		}
		log.Info("Opening local storage; connecting to master")
		const unspecifiedAddress = "0.0.0.0"
		address := cctx.String("listen")
		addressSlice := strings.Split(address, ":")
		if ip := net.ParseIP(addressSlice[0]); ip != nil {
			if ip.String() == unspecifiedAddress {
				timeout, err := time.ParseDuration(cctx.String("timeout"))
				if err != nil {
					return err
				}
				rip, err := extractRoutableIP(cfg.Sealer.Url, timeout)
				if err != nil {
					return err
				}
				address = rip + ":" + addressSlice[1]
			}
		}

		localStore, err := stores.NewLocal(ctx, localStorage, nodeApi, []string{"http://" + address + "/remote"})
		if err != nil {
			return err
		}

		remote := stores.NewRemote(localStore, nodeApi, cfg.Sealer.AuthHeader(), cctx.Int("parallel-fetch-limit"),
			&stores.DefaultPartialFileHandler{})

		fh := &stores.FetchHandler{Local: localStore, PfHandler: &stores.DefaultPartialFileHandler{}}
		remoteHandler := func(w http.ResponseWriter, r *http.Request) {
			if !auth.HasPerm(r.Context(), nil, api.PermAdmin) {
				w.WriteHeader(401)
				_ = json.NewEncoder(w).Encode(struct{ Error string }{"unauthorized: missing admin permission"})
				return
			}

			fh.ServeHTTP(w, r)
		}

		// Create / expose the worker
		wsts := service.NewWorkCallService(dbRepo, "worker")
		//wsts := statestore.New(namespace.Wrap(ds, sealer.WorkerCallsPrefix))

		workerApi := &worker{
			LocalWorker: sectorstorage.NewLocalWorker(sectorstorage.WorkerConfig{
				TaskTypes: taskTypes,
				NoSwap:    cctx.Bool("no-swap"),
			}, remote, localStore, nodeApi, nodeApi, wsts),
			localStore: localStore,
			ls:         localStorage,
		}

		mux := mux.NewRouter()

		log.Info("Setting up control endpoint at " + address)

		readerHandler, readerServerOpt := rpcenc.ReaderParamDecoder()
		rpcServer := jsonrpc.NewServer(readerServerOpt)
		rpcServer.Register("Filecoin", workerApi)

		mux.Handle("/rpc/v0", rpcServer)
		mux.Handle("/rpc/streams/v0/push/{uuid}", readerHandler)
		mux.PathPrefix("/remote").HandlerFunc(remoteHandler)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

		ah := &auth.Handler{
			Verify: nodeApi.AuthVerify,
			Next:   mux.ServeHTTP,
		}

		srv := &http.Server{
			Handler: ah,
			BaseContext: func(listener net.Listener) context.Context {
				apiKey, _ := tag.NewKey("api")
				ctx, _ := tag.New(context.Background(), tag.Upsert(apiKey, "lotus-worker"))
				return ctx
			},
		}

		go func() {
			cs := <-ctx.Done()
			log.Warnf("Shutting down: %v ...", cs)
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()

		nl, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}

		{
			a, err := net.ResolveTCPAddr("tcp", address)
			if err != nil {
				return xerrors.Errorf("parsing address: %w", err)
			}

			ma, err := manet.FromNetAddr(a)
			if err != nil {
				return xerrors.Errorf("creating api multiaddress: %w", err)
			}

			if err := cfg.LocalStorage().SetAPIEndpoint(ma); err != nil {
				return xerrors.Errorf("setting api endpoint: %w", err)
			}

			// TODO: ideally this would be a token with some permissions dropped
			if err := cfg.LocalStorage().SetAPIToken([]byte(cfg.Sealer.Token)); err != nil {
				return xerrors.Errorf("setting api token: %w", err)
			}
		}

		minerSession, err := nodeApi.Session(ctx)
		if err != nil {
			return xerrors.Errorf("getting miner session: %w", err)
		}

		waitQuietCh := func() chan struct{} {
			out := make(chan struct{})
			go func() {
				workerApi.LocalWorker.WaitQuiet()
				close(out)
			}()
			return out
		}

		go func() {
			heartbeats := time.NewTicker(stores.HeartbeatInterval)
			defer heartbeats.Stop()

			var redeclareStorage bool
			var readyCh chan struct{}
			for {
				// If we're reconnecting, redeclare storage first
				if redeclareStorage {
					log.Info("Redeclaring local storage")

					if err := localStore.Redeclare(ctx); err != nil {
						log.Errorf("Redeclaring local storage failed: %+v", err)

						select {
						case <-ctx.Done():
							return // graceful shutdown
						case <-heartbeats.C:
						}
						continue
					}
				}

				// TODO: we could get rid of this, but that requires tracking resources for restarted tasks correctly
				if readyCh == nil {
					log.Info("Making sure no local tasks are running")
					readyCh = waitQuietCh()
				}

				for {
					curSession, err := nodeApi.Session(ctx)
					if err != nil {
						log.Errorf("heartbeat: checking remote session failed: %+v", err)
					} else {
						if curSession != minerSession {
							minerSession = curSession
							break
						}
					}

					select {
					case <-readyCh: // 没有tasks后退出
						if err := nodeApi.WorkerConnect(ctx, "http://"+address+"/rpc/v0"); err != nil {
							log.Errorf("Registering worker failed: %+v", err)
							cancel()
							return
						}

						log.Info("Worker registered successfully, waiting for tasks")

						readyCh = nil
					case <-heartbeats.C:
					case <-ctx.Done():
						return // graceful shutdown
					}
				}

				log.Errorf("VENUS-MINER CONNECTION LOST")

				redeclareStorage = true
			}
		}()

		return srv.Serve(nl)
	},
}

func cleaningUpRepo(path string) {
	log.Infof("Cleaning up %s after attempt...", path)
	if err := os.RemoveAll(path); err != nil {
		log.Errorf("Failed to clean up failed storage repo: %s", err)
	}
}

func extractRoutableIP(url string, timeout time.Duration) (string, error) {
	minerAddr := strings.Split(url, "/")
	conn, err := net.DialTimeout("tcp", minerAddr[2]+":"+minerAddr[4], timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close() //nolint:errcheck

	localAddr := conn.LocalAddr().(*net.TCPAddr)

	return strings.Split(localAddr.IP.String(), ":")[0], nil
}

func flagData(cfg *config.StorageWorker, cctx *cli.Context) error {
	//rewrite sealer connection
	if cctx.IsSet("miner-addr") {
		cfg.Sealer.Url = cctx.String("miner-addr")
	}

	if cctx.IsSet("miner-token") {
		cfg.Sealer.Token = cctx.String("miner-token")
	}

	if cctx.IsSet("repo") {
		cfg.DataDir = cctx.String("repo")
	}
	return nil
}
