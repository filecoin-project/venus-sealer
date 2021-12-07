package venus_sealer

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	storage2 "github.com/filecoin-project/specs-storage/storage"
	api2 "github.com/filecoin-project/venus-market/api"
	config2 "github.com/filecoin-project/venus-market/config"
	"github.com/filecoin-project/venus-market/piecestorage"
	"github.com/filecoin-project/venus-sealer/api"
	"github.com/filecoin-project/venus-sealer/api/impl"
	"github.com/filecoin-project/venus-sealer/config"
	"github.com/filecoin-project/venus-sealer/journal"
	"github.com/filecoin-project/venus-sealer/market_client"
	"github.com/filecoin-project/venus-sealer/models"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/proof_client"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/ffiwrapper"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/service"
	"github.com/filecoin-project/venus-sealer/storage"
	"github.com/filecoin-project/venus-sealer/storage/sectorblocks"
	"github.com/filecoin-project/venus-sealer/types"
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
	AutoMigrateKey
	SetNetParamsKey
	GetParamsKey
	ExtractApiKey
	// daemon
	SetApiEndpointKey

	//proof
	StartProofEventKey
	StartWalletEventKey
	WarmupKey
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

func Online(cfg *config.StorageMiner) Option {
	return Options(
		Override(new(MetricsCtx), func() context.Context {
			return metricsi.CtxScope(context.Background(), "venus-sealer,")
		}),
		Override(new(journal.DisabledEvents), journal.EnvDisabledEvents),
		Override(new(journal.Journal), OpenFilesystemJournal),

		Override(new(types.SetSealingConfigFunc), NewSetSealConfigFunc),
		Override(new(types.GetSealingConfigFunc), NewGetSealConfigFunc),

		Override(new(api.Common), From(new(impl.CommonAPI))),
		Override(new(sectorstorage.StorageAuth), StorageAuth),

		Override(new(*stores.Index), stores.NewIndex),
		Override(new(stores.SectorIndex), From(new(*stores.Index))),
		Override(new(types.MinerID), MinerID),
		Override(new(types.MinerAddress), MinerAddress),
		Override(new(abi.RegisteredSealProof), SealProofType),
		Override(new(stores.LocalStorage), cfg.LocalStorage()), //todo
		Override(new(types.SectorIDCounter), SectorIDCounter),
		Override(new(*stores.Local), LocalStorage),
		Override(new(*stores.Remote), RemoteStorage),
		Override(new(*sectorstorage.Manager), SectorStorage),
		Override(new(ffiwrapper.Verifier), ffiwrapper.ProofVerifier),
		Override(new(ffiwrapper.Prover), ffiwrapper.ProofProver),
		Override(new(storage.WinningPoStProver), storage.NewWinningPoStProver),
		Override(new(sectorstorage.SectorManager), From(new(*sectorstorage.Manager))),
		Override(new(storage2.Prover), From(new(sectorstorage.SectorManager))),
		Override(new(storiface.WorkerReturn), From(new(sectorstorage.SectorManager))),

		Override(new(types.GetSealingConfigFunc), NewGetSealConfigFunc),
		Override(new(*sectorblocks.SectorBlocks), sectorblocks.NewSectorBlocks),
		Override(new(*storage.Miner), StorageMiner(config.DefaultMainnetStorageMiner().Fees)),
		// Override(new(*storage.AddressSelector), AddressSelector(nil)), // venus-sealer run: Call Repo before, Online after,will overwrite the original injection(MinerAddressConfig)
		Override(new(types.NetworkName), StorageNetworkName),
		Override(GetParamsKey, GetParams),
		Override(AutoMigrateKey, models.AutoMigrate),
		Override(SetNetParamsKey, SetupNetParams),
		Override(StartProofEventKey, proof_client.StartProofEvent),
		Override(StartWalletEventKey, market_client.StartMarketEvent),
		Override(WarmupKey, DoPoStWarmup),
	)
}

func Repo(cfg *config.StorageMiner) Option {
	return func(settings *Settings) error {
		return Options(
			Override(new(*types.APIAlg), APISecret),

			Override(new(config.HomeDir), HomeDir(cfg.DataDir)),
			Override(new(*config.NetParamsConfig), &cfg.NetParams),
			Override(new(sectorstorage.SealerConfig), cfg.Storage),
			Override(new(*storage.AddressSelector), AddressSelector(&cfg.Addresses)),
			Override(new(*config.DbConfig), &cfg.DB),
			Override(new(*config2.PieceStorage), &cfg.PieceStorage),
			Override(new(*config.StorageMiner), cfg),
			Override(new(*config.MessagerConfig), &cfg.Messager),
			Override(new(*config.MarketConfig), &cfg.Market),
			Override(new(*config.RegisterMarketConfig), &cfg.RegisterMarket),
			Override(new(*config.RegisterProofConfig), &cfg.RegisterProof),
			ConfigAPI(cfg),

			Override(new(api.IMessager), api.NewMessageRPC),
			Override(new(api2.MarketFullNode), api.NewMarketRPC),
			Override(new(piecestorage.IPieceStorage), NewPieceStorage),
			Override(new(*market_client.MarketEventClient), market_client.NewMarketEventClient),
			Override(new(*proof_client.ProofEventClient), proof_client.NewProofEventClient),
			Override(new(repo.Repo), models.SetDataBase),
			Providers(
				service.NewDealRefServiceService,
				service.NewLogService,
				service.NewMetadataService,
				service.NewSectorInfoService,
			//	service.NewWorkCallService,
			//	service.NewWorkStateService,
			),
		)(settings)
	}
}

// Config sets up constructors based on the provided Config
func ConfigAPI(cfg *config.StorageMiner) Option {
	return Options(
		Override(new(types.APIEndpoint), func() (types.APIEndpoint, error) {
			return multiaddr.NewMultiaddr(cfg.API.ListenAddress)
		}),
		Override(SetApiEndpointKey, func(e types.APIEndpoint) error {
			return cfg.LocalStorage().SetAPIEndpoint(e)
		}),
		Override(new(types.APIToken), func() ([]byte, error) {
			return cfg.LocalStorage().APIToken()
		}),
		Override(new(sectorstorage.URLs), func(e types.APIEndpoint) (sectorstorage.URLs, error) {
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
