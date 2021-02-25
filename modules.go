package venus_sealer

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc/auth"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-statestore"
	"github.com/filecoin-project/go-storedcounter"
	types2 "github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/dtypes"
	sectorstorage "github.com/filecoin-project/venus-sealer/extern/sector-storage"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/stores"
	sealing "github.com/filecoin-project/venus-sealer/extern/storage-sealing"
	"github.com/filecoin-project/venus-sealer/extern/storage-sealing/sealiface"
	"github.com/filecoin-project/venus-sealer/journal"
	"github.com/filecoin-project/venus-sealer/lib/backupds"
	"github.com/filecoin-project/venus-sealer/repo"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus/fixtures/asset"
	"github.com/filecoin-project/venus/pkg/specactors/builtin/miner"
	"github.com/filecoin-project/venus/pkg/types"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	"go.uber.org/fx"
	"go.uber.org/multierr"
	"golang.org/x/xerrors"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

func OpenFilesystemJournal(lr repo.LockedRepo, lc fx.Lifecycle, disabled journal.DisabledEvents) (journal.Journal, error) {
	jrnl, err := journal.OpenFSJournal(lr, disabled)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(_ context.Context) error { return jrnl.Close() },
	})

	return jrnl, err
}

func KeyStore(lr repo.LockedRepo) (types2.KeyStore, error) {
	return lr.KeyStore()
}

//auth

const (
	JWTSecretName   = "auth-jwt-private" //nolint:gosec
	KTJwtHmacSecret = "jwt-hmac-secret"  //nolint:gosec
)

type JwtPayload struct {
	Allow []auth.Permission
}

func APISecret(keystore types2.KeyStore, lr repo.LockedRepo) (*dtypes.APIAlg, error) {
	key, err := keystore.Get(JWTSecretName)

	if errors.Is(err, types.ErrKeyInfoNotFound) {
		log.Warn("Generating new API secret")

		sk, err := ioutil.ReadAll(io.LimitReader(rand.Reader, 32))
		if err != nil {
			return nil, err
		}

		key = types2.KeyInfo{
			Type:       KTJwtHmacSecret,
			PrivateKey: sk,
		}

		if err := keystore.Put(JWTSecretName, key); err != nil {
			return nil, xerrors.Errorf("writing API secret: %w", err)
		}

		// TODO: make this configurable
		p := JwtPayload{
			Allow: api.AllPermissions,
		}

		cliToken, err := jwt.Sign(&p, jwt.NewHS256(key.PrivateKey))
		if err != nil {
			return nil, err
		}

		if err := lr.SetAPIToken(cliToken); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, xerrors.Errorf("could not get JWT Token: %w", err)
	}

	return (*dtypes.APIAlg)(jwt.NewHS256(key.PrivateKey)), nil
}

//repo
func LockedRepo(lr repo.LockedRepo) func(lc fx.Lifecycle) repo.LockedRepo {
	return func(lc fx.Lifecycle) repo.LockedRepo {
		lc.Append(fx.Hook{
			OnStop: func(_ context.Context) error {
				return lr.Close()
			},
		})
		return lr
	}
}

func Datastore(r repo.LockedRepo) (dtypes.MetadataDS, error) {
	mds, err := r.Datastore("/metadata")
	if err != nil {
		return nil, err
	}

	return backupds.Wrap(mds), nil
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

func MinerID(ma dtypes.MinerAddress) (dtypes.MinerID, error) {
	id, err := address.IDFromAddress(address.Address(ma))
	return dtypes.MinerID(id), err
}

func MinerAddress(ds dtypes.MetadataDS) (dtypes.MinerAddress, error) {
	ma, err := minerAddrFromDS(ds)
	return dtypes.MinerAddress(ma), err
}

func minerAddrFromDS(ds dtypes.MetadataDS) (address.Address, error) {
	maddrb, err := ds.Get(datastore.NewKey("miner-address"))
	if err != nil {
		return address.Undef, err
	}

	return address.NewFromBytes(maddrb)
}

func SealProofType(maddr dtypes.MinerAddress, fnapi api.FullNode) (abi.RegisteredSealProof, error) {
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

func SectorIDCounter(ds dtypes.MetadataDS) sealing.SectorIDCounter {
	sc := storedcounter.New(ds, datastore.NewKey(StorageCounterDSPrefix))
	return &sidsc{sc}
}

var WorkerCallsPrefix = datastore.NewKey("/worker/calls")
var ManagerWorkPrefix = datastore.NewKey("/stmgr/calls")

func SectorStorage(mctx MetricsCtx, lc fx.Lifecycle, ls stores.LocalStorage, si stores.SectorIndex, sc sectorstorage.SealerConfig, urls sectorstorage.URLs, sa sectorstorage.StorageAuth, ds dtypes.MetadataDS) (*sectorstorage.Manager, error) {
	ctx := LifecycleCtx(mctx, lc)

	wsts := statestore.New(namespace.Wrap(ds, WorkerCallsPrefix))
	smsts := statestore.New(namespace.Wrap(ds, ManagerWorkPrefix))

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

func StorageNetworkName(ctx MetricsCtx, a api.FullNode) (dtypes.NetworkName, error) {
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
	MetadataDS         dtypes.MetadataDS
	Sealer             sectorstorage.SectorManager
	SectorIDCounter    sealing.SectorIDCounter
	Verifier           ffiwrapper.Verifier
	GetSealingConfigFn dtypes.GetSealingConfigFunc
	Journal            journal.Journal
	AddrSel            *storage.AddressSelector
	NetworkParams      *config.NetParamsConfig
}

func StorageMiner(fc config.MinerFeeConfig) func(params StorageMinerParams) (*storage.Miner, error) {
	return func(params StorageMinerParams) (*storage.Miner, error) {
		var (
			ds     = params.MetadataDS
			mctx   = params.MetricsCtx
			lc     = params.Lifecycle
			api    = params.API
			sealer = params.Sealer
			sc     = params.SectorIDCounter
			verif  = params.Verifier
			gsd    = params.GetSealingConfigFn
			j      = params.Journal
			as     = params.AddrSel
			np     = params.NetworkParams
		)

		maddr, err := minerAddrFromDS(ds)
		if err != nil {
			return nil, err
		}

		ctx := LifecycleCtx(mctx, lc)

		fps, err := storage.NewWindowedPoStScheduler(api, fc, as, sealer, verif, sealer, j, maddr, np)
		if err != nil {
			return nil, err
		}

		sm, err := storage.NewMiner(api, maddr, ds, sealer, sc, verif, gsd, fc, j, as, np)
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

func NewSetSealConfigFunc(r repo.LockedRepo) (dtypes.SetSealingConfigFunc, error) {
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

func NewGetSealConfigFunc(r repo.LockedRepo) (dtypes.GetSealingConfigFunc, error) {
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

func readCfg(r repo.LockedRepo, accessor func(*config.StorageMiner)) error {
	raw, err := r.Config()
	if err != nil {
		return err
	}

	cfg, ok := raw.(*config.StorageMiner)
	if !ok {
		return xerrors.New("expected address of config.StorageMiner")
	}

	accessor(cfg)

	return nil
}

func mutateCfg(r repo.LockedRepo, mutator func(*config.StorageMiner)) error {
	var typeErr error

	setConfigErr := r.SetConfig(func(raw interface{}) {
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
