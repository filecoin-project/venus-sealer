package config

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus/pkg/specactors/policy"
	"github.com/filecoin-project/venus/pkg/types"
	"time"
)

func GetDefaultWorkerConfig() *StorageWorker {
	return &StorageWorker{
		DataDir: "~/venusworker",
		Url:     "",
		Token:   "",
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
		Sealing: SealingConfig{
			MaxWaitDealsSectors:       2, // 64G with 32G sectors
			MaxSealingSectors:         0,
			MaxSealingSectorsForDeals: 0,
			WaitDealsDelay:            Duration(time.Hour * 6),
		},
		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:   true,
			AllowPreCommit1: true,
			AllowPreCommit2: true,
			AllowCommit:     true,
			AllowUnseal:     true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},
		Fees: MinerFeeConfig{
			MaxPreCommitGasFee:     types.MustParseFIL("0.025"),
			MaxCommitGasFee:        types.MustParseFIL("0.05"),
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
			UpgradeIgnitionHeight:  94000,
			ForkLengthThreshold:    policy.ChainFinality,
			InsecurePoStValidation: false,
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
	}
	var secret [32]byte
	rand.Read(secret[:])
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
		Sealing: SealingConfig{
			MaxWaitDealsSectors:       2, // 64G with 32G sectors
			MaxSealingSectors:         0,
			MaxSealingSectorsForDeals: 0,
			WaitDealsDelay:            Duration(time.Hour * 6),
		},
		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:   true,
			AllowPreCommit1: true,
			AllowPreCommit2: true,
			AllowCommit:     true,
			AllowUnseal:     true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},
		Fees: MinerFeeConfig{
			MaxPreCommitGasFee:     types.MustParseFIL("0.025"),
			MaxCommitGasFee:        types.MustParseFIL("0.05"),
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
			UpgradeIgnitionHeight:  94000,
			ForkLengthThreshold:    policy.ChainFinality,
			InsecurePoStValidation: false,
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
	}
	var secret [32]byte
	rand.Read(secret[:])
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
		Sealing: SealingConfig{
			MaxWaitDealsSectors:       2, // 64G with 32G sectors
			MaxSealingSectors:         0,
			MaxSealingSectorsForDeals: 0,
			WaitDealsDelay:            Duration(time.Hour * 6),
		},

		Storage: sectorstorage.SealerConfig{
			AllowAddPiece:   true,
			AllowPreCommit1: true,
			AllowPreCommit2: true,
			AllowCommit:     true,
			AllowUnseal:     true,

			// Default to 10 - tcp should still be able to figure this out, and
			// it's the ratio between 10gbit / 1gbit
			ParallelFetchLimit: 10,
		},

		Fees: MinerFeeConfig{
			MaxPreCommitGasFee:     types.MustParseFIL("0.025"),
			MaxCommitGasFee:        types.MustParseFIL("0.05"),
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
			UpgradeIgnitionHeight:  -2,
			ForkLengthThreshold:    policy.ChainFinality,
			InsecurePoStValidation: false,
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
	}
	var secret [32]byte
	rand.Read(secret[:])
	cfg.JWT.Secret = hex.EncodeToString(secret[:])
	cfg.API.ListenAddress = "/ip4/127.0.0.1/tcp/2345/http"
	cfg.API.RemoteListenAddress = "127.0.0.1:2345"
	return cfg
}
