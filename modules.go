package venus_sealer

import (
	"context"
	"encoding/hex"
	"errors"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	paramfetch "github.com/filecoin-project/go-paramfetch"
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
	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/types"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/ipfs-force-community/venus-messager/api/client"
	"github.com/ipfs/go-datastore"
	"github.com/mitchellh/go-homedir"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"net/http"
	"time"
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

type sidsc struct {
	sc *storedcounter.StoredCounter
}

func (s *sidsc) Next() (abi.SectorNumber, error) {
	i, err := s.sc.Next()
	return abi.SectorNumber(i), err
}

func SectorIDCounter(metaDataService *service.MetadataService) types2.SectorIDCounter {
	return metaDataService
}

var WorkerCallsPrefix = datastore.NewKey("/worker/calls")
var ManagerWorkPrefix = datastore.NewKey("/stmgr/calls")

func SectorStorage(mctx MetricsCtx, lc fx.Lifecycle, ls stores.LocalStorage, si stores.SectorIndex, sc sectorstorage.SealerConfig, urls sectorstorage.URLs, sa sectorstorage.StorageAuth, repo repo.Repo) (*sectorstorage.Manager, error) {
	ctx := LifecycleCtx(mctx, lc)

	wsts := service.NewWorkCallService(repo, "sealer")
	smsts := service.NewWorkStateService(repo)

	sst, err := sectorstorage.New(ctx, ls, si, sc, urls, sa, wsts, smsts)
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
	if err := paramfetch.GetParams(mctx, ps, uint64(ssize)); err != nil {
		return xerrors.Errorf("fetching proof parameters: %w", err)
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

		return as, nil
	}
}

type StorageMinerParams struct {
	fx.In

	Lifecycle          fx.Lifecycle
	MetricsCtx         MetricsCtx
	API                api.FullNode
	Messager           client.IMessager
	MetadataService    *service.MetadataService
	LogService         *service.LogService
	SectorInfoService  *service.SectorInfoService
	Sealer             sectorstorage.SectorManager
	SectorIDCounter    types2.SectorIDCounter
	Verifier           ffiwrapper.Verifier
	GetSealingConfigFn types2.GetSealingConfigFunc
	Journal            journal.Journal
	AddrSel            *storage.AddressSelector
	NetworkParams      *config.NetParamsConfig
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
			sealer            = params.Sealer
			sc                = params.SectorIDCounter
			verif             = params.Verifier
			gsd               = params.GetSealingConfigFn
			j                 = params.Journal
			as                = params.AddrSel
			np                = params.NetworkParams
		)

		maddr, err := metadataService.GetMinerAddress()
		if err != nil {
			return nil, err
		}

		ctx := LifecycleCtx(mctx, lc)

		fps, err := storage.NewWindowedPoStScheduler(api, messager, fc, as, sealer, verif, sealer, j, maddr, np)
		if err != nil {
			return nil, err
		}

		sm, err := storage.NewMiner(api, messager, maddr, metadataService, sectorinfoService, logService, sealer, sc, verif, gsd, fc, j, as, np)
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

func NewSetSealConfigFunc(r *config.StorageMiner) (types2.SetSealingConfigFunc, error) {
	return func(cfg sealiface.Config) (err error) {
		err = mutateCfg(r, func(c *config.StorageMiner) {
			c.Sealing = config.SealingConfig{
				MaxWaitDealsSectors:       cfg.MaxWaitDealsSectors,
				MaxSealingSectors:         cfg.MaxSealingSectors,
				MaxSealingSectorsForDeals: cfg.MaxSealingSectorsForDeals,
				WaitDealsDelay:            config.Duration(cfg.WaitDealsDelay),
			}
		})
		return
	}, nil
}

func NewGetSealConfigFunc(r *config.StorageMiner) (types2.GetSealingConfigFunc, error) {
	return func() (out sealiface.Config, err error) {
		err = readCfg(r, func(cfg *config.StorageMiner) {
			out = sealiface.Config{
				MaxWaitDealsSectors:       cfg.Sealing.MaxWaitDealsSectors,
				MaxSealingSectors:         cfg.Sealing.MaxSealingSectors,
				MaxSealingSectorsForDeals: cfg.Sealing.MaxSealingSectorsForDeals,
				WaitDealsDelay:            time.Duration(cfg.Sealing.WaitDealsDelay),
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
