package venus_sealer

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/api/market"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-storedcounter"

	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/journal"
	"github.com/filecoin-project/venus-sealer/models/repo"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/service"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/storage-sealing/sealiface"
	types2 "github.com/filecoin-project/venus-sealer/types"

	config2 "github.com/filecoin-project/venus-market/config"
	"github.com/filecoin-project/venus-market/piecestorage"

	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin/miner"
	"github.com/filecoin-project/venus/venus-shared/actors/policy"
	"github.com/filecoin-project/venus/venus-shared/types"
)

func OpenFilesystemJournal(homeDir config.HomeDir, lc fx.Lifecycle, disabled journal.DisabledEvents) (journal.Journal, error) {
	jrnl, err := journal.OpenFSJournal(string(homeDir), disabled)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error { return jrnl.Close() },
	})

	return jrnl, err
}

//auth

const (
	JWTSecretName   = "auth-jwt-private" //nolint:gosec
	KTJwtHmacSecret = "jwt-hmac-secret"  //nolint:gosec
)

type JwtPayload struct {
	Allow []auth.Permission
}

func APISecret(cfg *config.StorageMiner) (*types2.APIAlg, error) {
	log.Warn("Generating new API secret")

	sk, err := hex.DecodeString(cfg.JWT.Secret)
	if err != nil {
		return nil, err
	}

	if len(sk) != 32 {
		return nil, xerrors.Errorf("error private key format")
	}

	// TODO: make this configurable
	p := JwtPayload{
		Allow: api.AllPermissions,
	}

	cliToken, err := jwt.Sign(&p, jwt.NewHS256(sk))
	if err != nil {
		return nil, err
	}

	if err := cfg.LocalStorage().SetAPIToken(cliToken); err != nil {
		return nil, err
	}
	return (*types2.APIAlg)(jwt.NewHS256(sk)), nil
}

//storage

func StorageAuth(ctx MetricsCtx, ca api.Common) (sectorstorage.StorageAuth, error) {
	token, err := ca.AuthNew(ctx, []auth.Permission{"admin"})
	if err != nil {
		return nil, xerrors.Errorf("creating storage auth header: %w", err)
	}

	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+string(token))
	return sectorstorage.StorageAuth(headers), nil
}

func MinerID(ma types2.MinerAddress) (types2.MinerID, error) {
	id, err := address.IDFromAddress(address.Address(ma))
	return types2.MinerID(id), err
}

func MinerAddress(metaDataService *service.MetadataService) (types2.MinerAddress, error) {
	ma, err := metaDataService.GetMinerAddress()
	return types2.MinerAddress(ma), err
}

func SealProofType(maddr types2.MinerAddress, fnapi api.FullNode) (abi.RegisteredSealProof, error) {
	mi, err := fnapi.StateMinerInfo(context.TODO(), address.Address(maddr), types.EmptyTSK)
	if err != nil {
		return 0, err
	}
	networkVersion, err := fnapi.StateNetworkVersion(context.TODO(), types.EmptyTSK)
	if err != nil {
		return 0, err
	}

	return miner.PreferredSealProofTypeFromWindowPoStType(networkVersion, mi.WindowPoStProofType)
}

var StorageCounterDSPrefix = "/storage/nextid"

// nolint
type sidsc struct {
	sc *storedcounter.StoredCounter
}

// nolint
func (s *sidsc) Next() (abi.SectorNumber, error) {
	i, err := s.sc.Next()
	return abi.SectorNumber(i), err
}

func SectorIDCounter(metaDataService *service.MetadataService) types2.SectorIDCounter {
	return metaDataService
}

var WorkerCallsPrefix = datastore.NewKey("/worker/calls")
var ManagerWorkPrefix = datastore.NewKey("/stmgr/calls")

func LocalStorage(mctx MetricsCtx, lc fx.Lifecycle, ls stores.LocalStorage, si stores.SectorIndex, urls sectorstorage.URLs) (*stores.Local, error) {
	ctx := LifecycleCtx(mctx, lc)
	return stores.NewLocal(ctx, ls, si, urls)
}

func RemoteStorage(lstor *stores.Local, si stores.SectorIndex, sa sectorstorage.StorageAuth, sc sectorstorage.SealerConfig) *stores.Remote {
	return stores.NewRemote(lstor, si, http.Header(sa), sc.ParallelFetchLimit, &stores.DefaultPartialFileHandler{})
}

func SectorStorage(mctx MetricsCtx, lc fx.Lifecycle, lstor *stores.Local, stor *stores.Remote, ls stores.LocalStorage, si stores.SectorIndex, sc sectorstorage.SealerConfig, repo repo.Repo) (*sectorstorage.Manager, error) {
	ctx := LifecycleCtx(mctx, lc)

	wsts := service.NewWorkCallService(repo, "sealer")
	smsts := service.NewWorkStateService(repo)

	sst, err := sectorstorage.New(ctx, lstor, stor, ls, si, sc, wsts, smsts)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: sst.Close,
	})

	return sst, nil
}

func GetParams(mctx MetricsCtx, spt abi.RegisteredSealProof) error {
	ssize, err := spt.SectorSize()
	if err != nil {
		return err
	}

	ps, err := asset.Asset("fixtures/_assets/proof-params/parameters.json")
	if err != nil {
		return err
	}

	srs, err := asset.Asset("fixtures/_assets/proof-params/srs-inner-product.json")
	if err != nil {
		return err
	}
	if err := paramfetch.GetParams(mctx, ps, srs, uint64(ssize)); err != nil {
		return xerrors.Errorf("get params: %w", err)
	}
	return nil
}

func StorageNetworkName(ctx MetricsCtx, a api.FullNode) (types2.NetworkName, error) {
	/*	if !build.Devnet {
		return "testnetnet", nil
	}*/
	return a.StateNetworkName(ctx)
}

func AddressSelector(addrConf *config.MinerAddressConfig) func() (*storage.AddressSelector, error) {
	return func() (*storage.AddressSelector, error) {
		as := &storage.AddressSelector{}
		if addrConf == nil {
			return as, nil
		}

		log.Infof("miner address config: %v", *addrConf)
		for _, s := range addrConf.PreCommitControl {
			addr, err := address.NewFromString(s)
			if err != nil {
				return nil, xerrors.Errorf("parsing precommit control address: %w", err)
			}

			as.PreCommitControl = append(as.PreCommitControl, addr)
		}

		for _, s := range addrConf.CommitControl {
			addr, err := address.NewFromString(s)
			if err != nil {
				return nil, xerrors.Errorf("parsing commit control address: %w", err)
			}

			as.CommitControl = append(as.CommitControl, addr)
		}

		as.DisableOwnerFallback = addrConf.DisableOwnerFallback
		as.DisableWorkerFallback = addrConf.DisableWorkerFallback

		return as, nil
	}
}

func NewPieceStorage(cfg *config2.PieceStorage, preSignOp piecestorage.IPreSignOp) (piecestorage.IPieceStorage, error) {
	if cfg.PreSignS3.Enable {
		piecestorage.RegisterPieceStorageCtor(piecestorage.PreSignS3,
			func(cfg interface{}) (piecestorage.IPieceStorage, error) {
				return piecestorage.NewPresignS3Storage(preSignOp), nil
			})
	}
	if !cfg.S3.Enable && !cfg.Fs.Enable && !cfg.PreSignS3.Enable {
		return nil, nil
	}
	return piecestorage.NewPieceStorage(cfg)
}

func NewPreSignS3Op(cfg *config2.PieceStorage, marketAPI market.IMarket) piecestorage.IPreSignOp {
	return marketAPI
}

type StorageMinerParams struct {
	fx.In

	Lifecycle          fx.Lifecycle
	MetricsCtx         MetricsCtx
	API                api.FullNode
	Messager           api.IMessager
	MarketClient       market.IMarket
	MetadataService    *service.MetadataService
	LogService         *service.LogService
	SectorInfoService  *service.SectorInfoService
	Sealer             sectorstorage.SectorManager
	SectorIDCounter    types2.SectorIDCounter
	Verifier           ffiwrapper.Verifier
	Prover             ffiwrapper.Prover
	GetSealingConfigFn types2.GetSealingConfigFunc
	Journal            journal.Journal
	AddrSel            *storage.AddressSelector
	NetworkParams      *config.NetParamsConfig
	PieceStorage       piecestorage.IPieceStorage `optional:"true"`
	Maddr              types2.MinerAddress
}

func DoPoStWarmup(ctx MetricsCtx, api api.FullNode, metadataService *service.MetadataService, prover storage.WinningPoStProver) error {
	maddr, err := metadataService.GetMinerAddress()
	if err != nil {
		return err
	}
	deadlines, err := api.StateMinerDeadlines(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting deadlines: %w", err)
	}

	var sector abi.SectorNumber = math.MaxUint64

out:
	for dlIdx := range deadlines {
		partitions, err := api.StateMinerPartitions(ctx, maddr, uint64(dlIdx), types.EmptyTSK)
		if err != nil {
			return xerrors.Errorf("getting partitions for deadline %d: %w", dlIdx, err)
		}

		for _, partition := range partitions {
			b, err := partition.ActiveSectors.First()
			if err == bitfield.ErrNoBitsSet {
				continue
			}
			if err != nil {
				return err
			}
			sector = abi.SectorNumber(b)
			break out
		}
	}

	if sector == math.MaxUint64 {
		log.Info("skipping winning PoSt warmup, no sectors")
		return nil
	}

	log.Infow("starting winning PoSt warmup", "sector", sector)
	start := time.Now()

	var r abi.PoStRandomness = make([]byte, abi.RandomnessLength)
	_, _ = rand.Read(r)

	si, err := api.StateSectorGetInfo(ctx, maddr, sector, types.EmptyTSK)
	if err != nil {
		return xerrors.Errorf("getting sector info: %w", err)
	}

	ts, err := api.ChainHead(ctx)
	if err != nil {
		return xerrors.Errorf("PostWarmup failed: call ChainHead failed:%w", err)
	}

	version, err := api.StateNetworkVersion(ctx, ts.Key())
	if err != nil {
		return xerrors.Errorf("PostWarmup failed: get network version failed:%w", err)
	}
	_, err = prover.ComputeProof(ctx, []builtin.ExtendedSectorInfo{
		{
			SealProof:    si.SealProof,
			SectorNumber: sector,
			SectorKey:    si.SectorKeyCID,
			SealedCID:    si.SealedCID,
		}}, r, ts.Height(), version)
	if err != nil {
		log.Errorf("failed to compute proof: %s, please check your storage and restart sealer after fixed", err.Error())
		return nil
	}

	log.Infow("winning PoSt warmup successful", "took", time.Since(start))
	return nil
}

func StorageMiner(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.Miner, error) {
	return func(params StorageMinerParams) (*storage.Miner, error) {
		var (
			metadataService   = params.MetadataService
			sectorinfoService = params.SectorInfoService
			logService        = params.LogService
			mctx              = params.MetricsCtx
			lc                = params.Lifecycle
			api               = params.API
			messager          = params.Messager
			marketClient      = params.MarketClient
			sealer            = params.Sealer
			sc                = params.SectorIDCounter
			verif             = params.Verifier
			prover            = params.Prover
			gsd               = params.GetSealingConfigFn
			j                 = params.Journal
			as                = params.AddrSel
			np                = params.NetworkParams
			ps                = params.PieceStorage
			maddr             = address.Address(params.Maddr)
		)

		ctx := LifecycleCtx(mctx, lc)

		fps, err := storage.NewWindowedPoStScheduler(api, messager, fc, as, sealer, verif, sealer, j, maddr, np)
		if err != nil {
			return nil, err
		}

		sm, err := storage.NewMiner(api, ps, messager, marketClient, maddr, metadataService, sectorinfoService, logService, sealer, sc, verif, prover, gsd, fc, j, as, np)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				go fps.Run(ctx)
				return sm.Run(ctx)
			},
			OnStop: sm.Stop,
		})

		return sm, nil
	}
}

func WindowPostScheduler(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.WindowPoStScheduler, error) {
	return func(params StorageMinerParams) (*storage.WindowPoStScheduler, error) {
		var (
			mctx     = params.MetricsCtx
			lc       = params.Lifecycle
			api      = params.API
			messager = params.Messager
			sealer   = params.Sealer
			verif    = params.Verifier
			j        = params.Journal
			as       = params.AddrSel
			np       = params.NetworkParams
			maddr    = address.Address(params.Maddr)
		)

		ctx := LifecycleCtx(mctx, lc)

		fps, err := storage.NewWindowedPoStScheduler(api, messager, fc, as, sealer, verif, sealer, j, maddr, np)
		if err != nil {
			return nil, err
		}

		lc.Append(fx.Hook{
			OnStart: func(context.Context) error {
				go fps.Run(ctx)
				return nil
			},
		})

		return fps, nil
	}
}

func NewSetSealConfigFunc(r *config.StorageMiner) (types2.SetSealingConfigFunc, error) {
	return func(cfg sealiface.Config) (err error) {
		err = mutateCfg(r, func(c *config.StorageMiner) {
			c.Sealing = config.SealingConfig{
				MaxWaitDealsSectors:       cfg.MaxWaitDealsSectors,
				MaxSealingSectors:         cfg.MaxSealingSectors,
				MaxSealingSectorsForDeals: cfg.MaxSealingSectorsForDeals,
				PreferNewSectorsForDeals:  cfg.PreferNewSectorsForDeals,
				MaxUpgradingSectors:       cfg.MaxUpgradingSectors,
				WaitDealsDelay:            config.Duration(cfg.WaitDealsDelay),
				MakeNewSectorForDeals:     cfg.MakeNewSectorForDeals,
				AlwaysKeepUnsealedCopy:    cfg.AlwaysKeepUnsealedCopy,
				FinalizeEarly:             cfg.FinalizeEarly,

				BatchPreCommits:     cfg.BatchPreCommits,
				MaxPreCommitBatch:   cfg.MaxPreCommitBatch,
				PreCommitBatchWait:  config.Duration(cfg.PreCommitBatchWait),
				PreCommitBatchSlack: config.Duration(cfg.PreCommitBatchSlack),

				AggregateCommits:           cfg.AggregateCommits,
				MinCommitBatch:             cfg.MinCommitBatch,
				MaxCommitBatch:             cfg.MaxCommitBatch,
				CommitBatchWait:            config.Duration(cfg.CommitBatchWait),
				CommitBatchSlack:           config.Duration(cfg.CommitBatchSlack),
				AggregateAboveBaseFee:      types.FIL(cfg.AggregateAboveBaseFee),
				BatchPreCommitAboveBaseFee: types.FIL(cfg.BatchPreCommitAboveBaseFee),

				CollateralFromMinerBalance: cfg.CollateralFromMinerBalance,
				AvailableBalanceBuffer:     types.FIL(cfg.AvailableBalanceBuffer),
				DisableCollateralFallback:  cfg.DisableCollateralFallback,

				TerminateBatchMax:  cfg.TerminateBatchMax,
				TerminateBatchMin:  cfg.TerminateBatchMin,
				TerminateBatchWait: config.Duration(cfg.TerminateBatchWait),
			}
		})
		return
	}, nil
}

func NewGetSealConfigFunc(r *config.StorageMiner) (types2.GetSealingConfigFunc, error) {
	return func() (out sealiface.Config, err error) {
		err = readCfg(r, func(cfg *config.StorageMiner) {
			// log.Infof("max sealing sectors: %v", cfg.Sealing.MaxSealingSectors)
			out = sealiface.Config{
				MaxWaitDealsSectors:             cfg.Sealing.MaxWaitDealsSectors,
				MaxSealingSectors:               cfg.Sealing.MaxSealingSectors,
				MaxSealingSectorsForDeals:       cfg.Sealing.MaxSealingSectorsForDeals,
				PreferNewSectorsForDeals:        cfg.Sealing.PreferNewSectorsForDeals,
				MaxUpgradingSectors:             cfg.Sealing.MaxUpgradingSectors,
				WaitDealsDelay:                  time.Duration(cfg.Sealing.WaitDealsDelay),
				MakeNewSectorForDeals:           cfg.Sealing.MakeNewSectorForDeals,
				CommittedCapacitySectorLifetime: time.Duration(cfg.Sealing.CommittedCapacitySectorLifetime),
				AlwaysKeepUnsealedCopy:          cfg.Sealing.AlwaysKeepUnsealedCopy,
				FinalizeEarly:                   cfg.Sealing.FinalizeEarly,

				BatchPreCommits:     cfg.Sealing.BatchPreCommits,
				MaxPreCommitBatch:   cfg.Sealing.MaxPreCommitBatch,
				PreCommitBatchWait:  time.Duration(cfg.Sealing.PreCommitBatchWait),
				PreCommitBatchSlack: time.Duration(cfg.Sealing.PreCommitBatchSlack),

				AggregateCommits:      cfg.Sealing.AggregateCommits,
				MinCommitBatch:        cfg.Sealing.MinCommitBatch,
				MaxCommitBatch:        cfg.Sealing.MaxCommitBatch,
				CommitBatchWait:       time.Duration(cfg.Sealing.CommitBatchWait),
				CommitBatchSlack:      time.Duration(cfg.Sealing.CommitBatchSlack),
				AggregateAboveBaseFee: types.BigInt(cfg.Sealing.AggregateAboveBaseFee),

				TerminateBatchMax:  cfg.Sealing.TerminateBatchMax,
				TerminateBatchMin:  cfg.Sealing.TerminateBatchMin,
				TerminateBatchWait: time.Duration(cfg.Sealing.TerminateBatchWait),

				BatchPreCommitAboveBaseFee: types.BigInt(cfg.Sealing.BatchPreCommitAboveBaseFee),
				CollateralFromMinerBalance: cfg.Sealing.CollateralFromMinerBalance,
				AvailableBalanceBuffer:     types.BigInt(cfg.Sealing.AvailableBalanceBuffer),
				DisableCollateralFallback:  cfg.Sealing.DisableCollateralFallback,

				StartEpochSealingBuffer: abi.ChainEpoch(cfg.Dealmaking.StartEpochSealingBuffer),
			}
		})
		return
	}, nil
}

func readCfg(cfg *config.StorageMiner, accessor func(*config.StorageMiner)) error {
	accessor(cfg)
	return nil
}

func mutateCfg(cfg *config.StorageMiner, mutator func(*config.StorageMiner)) error {
	var typeErr error
	setConfigErr := cfg.LocalStorage().SetConfig(func(raw interface{}) {
		cfg, ok := raw.(*config.StorageMiner)
		if !ok {
			typeErr = errors.New("expected miner config")
			return
		}

		mutator(cfg)
	})

	return multierr.Combine(typeErr, setConfigErr)
}

// MetricsCtx is a context wrapper with metrics
type MetricsCtx context.Context

// LifecycleCtx creates a context which will be cancelled when lifecycle stops
//
// This is a hack which we need because most of our services use contexts in a
// wrong way
func LifecycleCtx(mctx MetricsCtx, lc fx.Lifecycle) context.Context {
	ctx, cancel := context.WithCancel(mctx)
	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})
	return ctx
}

func HomeDir(path string) func() (config.HomeDir, error) {
	return func() (config.HomeDir, error) {
		path, err := homedir.Expand(path)
		if err != nil {
			return "", err
		}
		return config.HomeDir(path), nil
	}
}

func SetupNetParams(netParams *config.NetParamsConfig) {
	if netParams.PreCommitChallengeDelay > 0 {
		policy.SetPreCommitChallengeDelay(netParams.PreCommitChallengeDelay)
	}
}
