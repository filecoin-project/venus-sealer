package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/big"

	miner5 "github.com/filecoin-project/specs-actors/v5/actors/builtin/miner"

	"github.com/filecoin-project/venus/venus-shared/actors/policy"
	"github.com/filecoin-project/venus/venus-shared/types"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
)

func GetDefaultWorkerConfig() *StorageWorker {
	return &StorageWorker{
		DataDir: "~/.venusworker",
		Sealer: NodeConfig{
			Url:   "",
			Token: "",
		},
		DB: DbConfig{
			Type:  "sqlite",
			MySql: MySqlConfig{},
			Sqlite: SqliteConfig{
				Path: "worker.db",
			},
		},
	}
}
func GetDefaultStorageConfig(network string) (*StorageMiner, error) {
	switch network {
	case "mainnet":
		return DefaultMainnetStorageMiner(), nil
	case "calibration":
		return DefaultCalibrationStorageMiner(), nil
	case "2k":
		return Default2kStorageMiner(), nil
	case "force":
		return DefaultForceNetStorageMiner(), nil
	case "butterfly":
		return DefaultButterflyStorageMiner(), nil
	default:
		return nil, errors.New("unsupport network type")
	}
}

func DefaultMainnetStorageMiner() *StorageMiner {
	cfg := &StorageMiner{
		DataDir: "~/.venussealer",
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/38491/http",
			Timeout:       Duration(30 * time.Second),
		},
		Sealing: defSealing,
		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:            true,
			AllowPreCommit1:          true,
			AllowPreCommit2:          true,
			AllowCommit:              true,
			AllowUnseal:              true,
			AllowReplicaUpdate:       true,
			AllowProveReplicaUpdate2: true,
			AllowRegenSectorKey:      true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},
		Fees: MinerFeeConfig{
			MaxPreCommitGasFee: types.MustParseFIL("0.025"),
			MaxCommitGasFee:    types.MustParseFIL("0.05"),

			MaxPreCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.025"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.025"), // TODO: update before v1.10.0
			},
			MaxCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.05"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.05"), // TODO: update before v1.10.0
			},

			MaxTerminateGasFee:     types.MustParseFIL("0.5"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},
		Addresses: MinerAddressConfig{
			PreCommitControl: []string{},
			CommitControl:    []string{},
		},
		NetParams: NetParamsConfig{
			UpgradeIgnitionHeight: 94000,
			UpgradeOhSnapHeight:   1594680,
			UpgradeSkyrHeight:     99999999999999,
			ForkLengthThreshold:   policy.ChainFinality,
			BlockDelaySecs:        30,
		},
		DB: DbConfig{
			Type: "sqlite",
			MySql: MySqlConfig{
				Addr:            "",
				User:            "",
				Pass:            "",
				Name:            "",
				MaxOpenConn:     0,
				MaxIdleConn:     0,
				ConnMaxLifeTime: 0,
			},
			Sqlite: SqliteConfig{
				Path: "sealer.db",
			},
		},
		Node: NodeConfig{
			Url:   "/ip4/127.0.0.1/tcp/3453",
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJhbGwiXX0.50-NxTSm90nOzY5bu9XUc49Rk7k2iW7PlHb9BvErDpM",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		RegisterMarket: defMarket,
	}
	var secret [32]byte
	_, _ = rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}

func DefaultForceNetStorageMiner() *StorageMiner {
	cfg := &StorageMiner{
		DataDir: "~/.venussealer",
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/38491/http",
			Timeout:       Duration(30 * time.Second),
		},
		Sealing: defSealing,
		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:            true,
			AllowPreCommit1:          true,
			AllowPreCommit2:          true,
			AllowCommit:              true,
			AllowUnseal:              true,
			AllowReplicaUpdate:       true,
			AllowProveReplicaUpdate2: true,
			AllowRegenSectorKey:      true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},
		Fees: MinerFeeConfig{
			MaxPreCommitGasFee: types.MustParseFIL("0.025"),
			MaxCommitGasFee:    types.MustParseFIL("0.05"),

			MaxPreCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.025"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.025"), // TODO: update before v1.10.0
			},
			MaxCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.05"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.05"), // TODO: update before v1.10.0
			},

			MaxTerminateGasFee:     types.MustParseFIL("0.5"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},
		Addresses: MinerAddressConfig{
			PreCommitControl: []string{},
			CommitControl:    []string{},
		},
		NetParams: NetParamsConfig{
			UpgradeIgnitionHeight:   94000,
			UpgradeOhSnapHeight:     999999999999,
			UpgradeSkyrHeight:       999999999999,
			ForkLengthThreshold:     policy.ChainFinality,
			PreCommitChallengeDelay: 10,
			BlockDelaySecs:          30,
		},
		DB: DbConfig{
			Type: "sqlite",
			MySql: MySqlConfig{
				Addr:            "",
				User:            "",
				Pass:            "",
				Name:            "",
				MaxOpenConn:     0,
				MaxIdleConn:     0,
				ConnMaxLifeTime: 0,
			},
			Sqlite: SqliteConfig{
				Path: "sealer.db",
			},
		},
		Node: NodeConfig{
			Url:   "/ip4/127.0.0.1/tcp/3453",
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJhbGwiXX0.50-NxTSm90nOzY5bu9XUc49Rk7k2iW7PlHb9BvErDpM",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		RegisterMarket: defMarket,
	}
	var secret [32]byte
	_, _ = rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}

func DefaultCalibrationStorageMiner() *StorageMiner {
	cfg := &StorageMiner{
		DataDir: "~/.venussealer",
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/38491/http",
			Timeout:       Duration(30 * time.Second),
		},
		Sealing: defSealing,
		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:            true,
			AllowPreCommit1:          true,
			AllowPreCommit2:          true,
			AllowCommit:              true,
			AllowUnseal:              true,
			AllowReplicaUpdate:       true,
			AllowProveReplicaUpdate2: true,
			AllowRegenSectorKey:      true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},
		Fees: MinerFeeConfig{
			MaxPreCommitGasFee: types.MustParseFIL("0.025"),
			MaxCommitGasFee:    types.MustParseFIL("0.05"),

			MaxPreCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.025"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.025"), // TODO: update before v1.10.0
			},
			MaxCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.05"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.05"), // TODO: update before v1.10.0
			},

			MaxTerminateGasFee:     types.MustParseFIL("0.5"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},
		Addresses: MinerAddressConfig{
			PreCommitControl: []string{},
			CommitControl:    []string{},
		},
		NetParams: NetParamsConfig{
			UpgradeIgnitionHeight: -3,
			// 2022-02-10T19:23:00Z
			UpgradeOhSnapHeight: 682006,
			// 2022-06-16T17:30:00Z
			UpgradeSkyrHeight:   1044660,
			ForkLengthThreshold: policy.ChainFinality,
			BlockDelaySecs:      30,
		},
		DB: DbConfig{
			Type: "sqlite",
			MySql: MySqlConfig{
				Addr:            "",
				User:            "",
				Pass:            "",
				Name:            "",
				MaxOpenConn:     0,
				MaxIdleConn:     0,
				ConnMaxLifeTime: 0,
			},
			Sqlite: SqliteConfig{
				Path: "sealer.db",
			},
		},
		Node: NodeConfig{
			Url:   "/ip4/127.0.0.1/tcp/3453",
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJhbGwiXX0.50-NxTSm90nOzY5bu9XUc49Rk7k2iW7PlHb9BvErDpM",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		RegisterMarket: defMarket,

		Dealmaking: DealmakingConfig{
			ConsiderOnlineStorageDeals:     true,
			ConsiderOfflineStorageDeals:    true,
			ConsiderOnlineRetrievalDeals:   true,
			ConsiderOfflineRetrievalDeals:  true,
			ConsiderVerifiedStorageDeals:   true,
			ConsiderUnverifiedStorageDeals: true,
			PieceCidBlocklist:              []cid.Cid{},
			// TODO: It'd be nice to set this based on sector size
			MaxDealStartDelay:               Duration(time.Hour * 24 * 14),
			ExpectedSealDuration:            Duration(time.Hour * 24),
			PublishMsgPeriod:                Duration(time.Hour),
			MaxDealsPerPublishMsg:           8,
			MaxProviderCollateralMultiplier: 2,

			SimultaneousTransfersForStorage:   DefaultSimultaneousTransfers,
			SimultaneousTransfersForRetrieval: DefaultSimultaneousTransfers,

			StartEpochSealingBuffer: 480, // 480 epochs buffer == 4 hours from adding deal to sector to sector being sealed

			RetrievalPricing: &RetrievalPricing{
				Strategy: RetrievalPricingDefaultMode,
				Default: &RetrievalPricingDefault{
					VerifiedDealsFreeTransfer: true,
				},
				External: &RetrievalPricingExternal{
					Path: "",
				},
			},
		},
	}
	var secret [32]byte
	_, _ = rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}

func Default2kStorageMiner() *StorageMiner {
	cfg := &StorageMiner{
		DataDir: "~/.venussealer",
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/38491/http",
			Timeout:       Duration(30 * time.Second),
		},
		Sealing: defSealing,

		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:            true,
			AllowPreCommit1:          true,
			AllowPreCommit2:          true,
			AllowCommit:              true,
			AllowUnseal:              true,
			AllowReplicaUpdate:       true,
			AllowProveReplicaUpdate2: true,
			AllowRegenSectorKey:      true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},

		Fees: MinerFeeConfig{
			MaxPreCommitGasFee: types.MustParseFIL("0.025"),
			MaxCommitGasFee:    types.MustParseFIL("0.05"),

			MaxPreCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.025"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.025"), // TODO: update before v1.10.0
			},
			MaxCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.05"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.05"), // TODO: update before v1.10.0
			},

			MaxTerminateGasFee:     types.MustParseFIL("0.5"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},

		Addresses: MinerAddressConfig{
			PreCommitControl: []string{},
			CommitControl:    []string{},
		},
		NetParams: NetParamsConfig{
			UpgradeIgnitionHeight:   -2,
			UpgradeOhSnapHeight:     -18,
			UpgradeSkyrHeight:       -19,
			ForkLengthThreshold:     policy.ChainFinality,
			BlockDelaySecs:          4,
			PreCommitChallengeDelay: 10,
		},
		DB: DbConfig{
			Type: "sqlite",
			MySql: MySqlConfig{
				Addr:            "",
				User:            "",
				Pass:            "",
				Name:            "",
				MaxOpenConn:     0,
				MaxIdleConn:     0,
				ConnMaxLifeTime: 0,
			},
			Sqlite: SqliteConfig{
				Path: "sealer.db",
			},
		},
		Node: NodeConfig{
			Url:   "/ip4/127.0.0.1/tcp/3453",
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJhbGwiXX0.50-NxTSm90nOzY5bu9XUc49Rk7k2iW7PlHb9BvErDpM",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		RegisterMarket: defMarket,
	}
	var secret [32]byte
	_, _ = rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}

func DefaultButterflyStorageMiner() *StorageMiner {
	cfg := &StorageMiner{
		DataDir: "~/.venussealer",
		API: API{
			ListenAddress: "/ip4/127.0.0.1/tcp/38491/http",
			Timeout:       Duration(30 * time.Second),
		},
		Sealing: defSealing,

		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:            true,
			AllowPreCommit1:          true,
			AllowPreCommit2:          true,
			AllowCommit:              true,
			AllowUnseal:              true,
			AllowReplicaUpdate:       true,
			AllowProveReplicaUpdate2: true,
			AllowRegenSectorKey:      true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},

		Fees: MinerFeeConfig{
			MaxPreCommitGasFee: types.MustParseFIL("0.025"),
			MaxCommitGasFee:    types.MustParseFIL("0.05"),

			MaxPreCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.025"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.025"), // TODO: update before v1.10.0
			},
			MaxCommitBatchGasFee: BatchFeeConfig{
				Base:      types.MustParseFIL("0.05"), // TODO: update before v1.10.0
				PerSector: types.MustParseFIL("0.05"), // TODO: update before v1.10.0
			},

			MaxTerminateGasFee:     types.MustParseFIL("0.5"),
			MaxWindowPoStGasFee:    types.MustParseFIL("5"),
			MaxPublishDealsFee:     types.MustParseFIL("0.05"),
			MaxMarketBalanceAddFee: types.MustParseFIL("0.007"),
		},

		Addresses: MinerAddressConfig{
			PreCommitControl: []string{},
			CommitControl:    []string{},
		},
		NetParams: NetParamsConfig{
			UpgradeIgnitionHeight: -3,
			UpgradeOhSnapHeight:   -18,
			UpgradeSkyrHeight:     50,
			ForkLengthThreshold:   policy.ChainFinality,
			BlockDelaySecs:        30,
		},
		DB: DbConfig{
			Type: "sqlite",
			MySql: MySqlConfig{
				Addr:            "",
				User:            "",
				Pass:            "",
				Name:            "",
				MaxOpenConn:     0,
				MaxIdleConn:     0,
				ConnMaxLifeTime: 0,
			},
			Sqlite: SqliteConfig{
				Path: "sealer.db",
			},
		},
		Node: NodeConfig{
			Url:   "/ip4/127.0.0.1/tcp/3453",
			Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJhbGwiXX0.50-NxTSm90nOzY5bu9XUc49Rk7k2iW7PlHb9BvErDpM",
		},
		JWT: JWTConfig{
			Secret: "",
		},
		RegisterMarket: defMarket,
	}
	var secret [32]byte
	_, _ = rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}

var defMarket = RegisterMarketConfig{
	Urls:  []string{},
	Token: "",
}

var defSealing = SealingConfig{
	MaxWaitDealsSectors:       2, // 64G with 32G sectors
	MaxSealingSectors:         0,
	MaxSealingSectorsForDeals: 0,
	WaitDealsDelay:            Duration(time.Hour * 6),
	AlwaysKeepUnsealedCopy:    false, // todo
	FinalizeEarly:             false,
	MakeNewSectorForDeals:     true,

	BatchPreCommits:    false,                              // todo
	MaxPreCommitBatch:  miner5.PreCommitSectorBatchMaxSize, // up to 256 sectors
	PreCommitBatchWait: Duration(24 * time.Hour),           // this should be less than 31.5 hours, which is the expiration of a precommit ticket
	// XXX snap deals wait deals slack if first
	PreCommitBatchSlack: Duration(3 * time.Hour), // time buffer for forceful batch submission before sectors/deals in batch would start expiring, higher value will lower the chances for message fail due to expiration

	AggregateCommits: false,                       // todo
	MinCommitBatch:   miner5.MinAggregatedSectors, // per FIP13, we must have at least four proofs to aggregate, where 4 is the cross over point where aggregation wins out on single provecommit gas costs
	MaxCommitBatch:   miner5.MaxAggregatedSectors, // maximum 819 sectors, this is the maximum aggregation per FIP13
	CommitBatchWait:  Duration(24 * time.Hour),    // this can be up to 30 days
	CommitBatchSlack: Duration(1 * time.Hour),     // time buffer for forceful batch submission before sectors/deals in batch would start expiring, higher value will lower the chances for message fail due to expiration

	BatchPreCommitAboveBaseFee: types.FIL(types.BigMul(types.PicoFil, types.NewInt(320))), // 0.32 nFIL
	AggregateAboveBaseFee:      types.FIL(types.BigMul(types.PicoFil, types.NewInt(320))), // 0.32 nFIL

	TerminateBatchMin:               1,
	TerminateBatchMax:               100,
	TerminateBatchWait:              Duration(5 * time.Minute),
	CommittedCapacitySectorLifetime: Duration(time.Duration(policy.GetMaxSectorExpirationExtension()*30) * time.Second),

	CollateralFromMinerBalance: false,
	AvailableBalanceBuffer:     types.FIL(big.Zero()),
	DisableCollateralFallback:  false,
}
