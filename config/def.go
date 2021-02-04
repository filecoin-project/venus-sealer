package config

import (
	"encoding"
	"github.com/filecoin-project/go-state-types/abi"
	"time"

	"github.com/filecoin-project/lotus/chain/types"
	sectorstorage "github.com/filecoin-project/venus-sealer/extern/sector-storage"
)


// StorageMiner is a miner config
type StorageMiner struct {
	API    API
	Sealing    SealingConfig
	Storage    sectorstorage.SealerConfig
	Fees       MinerFeeConfig
	Addresses  MinerAddressConfig
	NetParams  NetParamsConfig
}

type SealingConfig struct {
	// 0 = no limit
	MaxWaitDealsSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectorsForDeals uint64

	WaitDealsDelay Duration
}

type MinerFeeConfig struct {
	MaxPreCommitGasFee     types.FIL
	MaxCommitGasFee        types.FIL
	MaxTerminateGasFee     types.FIL
	MaxWindowPoStGasFee    types.FIL
	MaxPublishDealsFee     types.FIL
	MaxMarketBalanceAddFee types.FIL
}

type MinerAddressConfig struct {
	PreCommitControl []string
	CommitControl    []string
}

// API contains configs for API endpoint
type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration
}

type FeeConfig struct {
	DefaultMaxFee types.FIL
}

type NetParamsConfig struct {
	UpgradeIgnitionHeight abi.ChainEpoch
	ForkLengthThreshold abi.ChainEpoch
	InsecurePoStValidation bool
	BlockDelaySecs uint64
}

var DefaultDefaultMaxFee = types.MustParseFIL("0.007")
var DefaultSimultaneousTransfers = uint64(20)


var _ encoding.TextMarshaler = (*Duration)(nil)
var _ encoding.TextUnmarshaler = (*Duration)(nil)

// Duration is a wrapper type for time.Duration
// for decoding and encoding from/to TOML
type Duration time.Duration

// UnmarshalText implements interface for TOML decoding
func (dur *Duration) UnmarshalText(text []byte) error {
	d, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*dur = Duration(d)
	return err
}

func (dur Duration) MarshalText() ([]byte, error) {
	d := time.Duration(dur)
	return []byte(d.String()), nil
}
