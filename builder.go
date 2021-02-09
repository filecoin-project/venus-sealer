package venus_sealer

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	types2 "github.com/filecoin-project/lotus/chain/types"
	storage2 "github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/api/impl"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/dtypes"
	sectorstorage "github.com/filecoin-project/venus-sealer/extern/sector-storage"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/extern/sector-storage/storiface"
	sealing "github.com/filecoin-project/venus-sealer/extern/storage-sealing"
	"github.com/filecoin-project/venus-sealer/journal"
	"github.com/filecoin-project/venus-sealer/repo"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	logging "github.com/ipfs/go-log/v2"
	metricsi "github.com/ipfs/go-metrics-interface"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/fx"
	"golang.org/x/xerrors"
)

var log = logging.Logger("modules")

type invoke int

// Invokes are called in the order they are defined.
//nolint:golint
const (
	// InitJournal at position 0 initializes the journal global var as soon as
	// the system starts, so that it's available for all other components.
	InitJournalKey = invoke(iota)

	// miner
	GetParamsKey
	ExtractApiKey
	// daemon
	SetApiEndpointKey

	_nInvokes // keep this last
)

type Settings struct {
	// modules is a map of constructors for DI
	//
	// In most cases the index will be a reflect. Type of element returned by
	// the constructor, but for some 'constructors' it's hard to specify what's
	// the return type should be (or the constructor returns fx group)
	modules map[interface{}]fx.Option

	// invokes are separate from modules as they can't be referenced by return
	// type, and must be applied in correct order
	invokes []fx.Option
}

type StopFunc func(context.Context) error

// New builds and starts new Filecoin node
func New(ctx context.Context, opts ...Option) (StopFunc, error) {
	settings := Settings{
		modules: map[interface{}]fx.Option{},
		invokes: make([]fx.Option, _nInvokes),
	}

	// apply module options in the right order
	if err := Options(opts...)(&settings); err != nil {
		return nil, xerrors.Errorf("applying node options failed: %w", err)
	}

	// gather constructors for fx.Options
	ctors := make([]fx.Option, 0, len(settings.modules))
	for _, opt := range settings.modules {
		ctors = append(ctors, opt)
	}

	// fill holes in invokes for use in fx.Options
	for i, opt := range settings.invokes {
		if opt == nil {
			settings.invokes[i] = fx.Options()
		}
	}

	app := fx.New(
		fx.Options(ctors...),
		fx.Options(settings.invokes...),

		fx.NopLogger,
	)

	// TODO: we probably should have a 'firewall' for Closing signal
	//  on this context, and implement closing logic through lifecycles
	//  correctly
	if err := app.Start(ctx); err != nil {
		// comment fx.NopLogger few lines above for easier debugging
		return nil, xerrors.Errorf("starting node: %w", err)
	}

	return app.Stop, nil
}

// Online sets up basic libp2p node
func Online() Option {
	return Options(
		Override(new(MetricsCtx), func() context.Context {
			return metricsi.CtxScope(context.Background(), "venus-sealer,")
		}),
		Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		Override(new(journal.Journal), OpenFilesystemJournal),

		Override(new(dtypes.SetSealingConfigFunc), NewSetSealConfigFunc),
		Override(new(dtypes.GetSealingConfigFunc), NewGetSealConfigFunc),

		Override(new(api.Common), From(new(impl.CommonAPI))),
		Override(new(sectorstorage.StorageAuth), StorageAuth),

		Override(new(*stores.Index), stores.NewIndex),
		Override(new(stores.SectorIndex), From(new(*stores.Index))),
		Override(new(dtypes.MinerID), MinerID),
		Override(new(dtypes.MinerAddress), MinerAddress),
		Override(new(abi.RegisteredSealProof), SealProofType),
		Override(new(stores.LocalStorage), From(new(repo.LockedRepo))),
		Override(new(sealing.SectorIDCounter), SectorIDCounter),
		Override(new(*sectorstorage.Manager), SectorStorage),
		Override(new(ffiwrapper.Verifier), ffiwrapper.ProofVerifier),

		Override(new(sectorstorage.SectorManager), From(new(*sectorstorage.Manager))),
		Override(new(storage2.Prover), From(new(sectorstorage.SectorManager))),
		Override(new(storiface.WorkerReturn), From(new(sectorstorage.SectorManager))),

		Override(new(*sectorblocks.SectorBlocks), sectorblocks.NewSectorBlocks),
		Override(new(*storage.Miner), StorageMiner(config.DefaultMainnetStorageMiner().Fees)),
		Override(new(*storage.AddressSelector), AddressSelector(nil)),
		Override(new(dtypes.NetworkName), StorageNetworkName),
		Override(GetParamsKey, GetParams),
	)
}

func Repo(lr repo.LockedRepo) Option {
	return func(settings *Settings) error {
		c, err := lr.Config()
		if err != nil {
			return err
		}

		cfg, ok := c.(*config.StorageMiner)
		if !ok {
			return xerrors.Errorf("invalid config from repo, got: %T", c)
		}

		return Options(
			Override(new(repo.LockedRepo), LockedRepo(lr)), // module handles closing
			Override(new(dtypes.MetadataDS), Datastore),
			Override(new(types2.KeyStore), KeyStore),
			Override(new(*dtypes.APIAlg), APISecret),

			Override(new(*config.NetParamsConfig), &cfg.NetParams),
			Override(new(sectorstorage.SealerConfig), cfg.Storage),
			Override(new(*storage.AddressSelector), AddressSelector(&cfg.Addresses)),

			ConfigAPI(cfg),
		)(settings)
	}
}

// Config sets up constructors based on the provided Config
func ConfigAPI(cfg *config.StorageMiner) Option {
	return Options(
		Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
			return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
		}),
		Override(SetApiEndpointKey, func(lr repo.LockedRepo, e dtypes.APIEndpoint) error {
			return lr.SetAPIEndpoint(e)
		}),
		Override(new(sectorstorage.URLs), func(e dtypes.APIEndpoint) (sectorstorage.URLs, error) {
			ip := cfg.API.RemoteListenAddress

			var urls sectorstorage.URLs
			urls = append(urls, "http://"+ip+"/remote") // TODO: This makes no assumptions, and probably could...
			return urls, nil
		}),
	)
}

func ConfigStorageAPIImpl(out *api.StorageMiner) Option {
	return Options(
		func(s *Settings) error {
			resAPI := &impl.StorageMinerAPI{}
			s.invokes[ExtractApiKey] = fx.Populate(resAPI)
			*out = resAPI
			return nil
		},
	)
}
