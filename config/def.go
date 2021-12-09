package config

import (
	"encoding"
	"github.com/filecoin-project/venus-market/config"
	"net/http"
	"strings"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/venus/pkg/types"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
)

const (
	// RetrievalPricingDefault configures the node to use the default retrieval pricing policy.
	RetrievalPricingDefaultMode = "default"
	// RetrievalPricingExternal configures the node to use the external retrieval pricing script
	// configured by the user.
	RetrievalPricingExternalMode = "external"
)

type HomeDir string

type StorageWorker struct {
	ConfigPath string `toml:"-"`
	DataDir    string
	Sealer     NodeConfig
	DB         DbConfig
}

func (cfg StorageWorker) LocalStorage() *LocalStorage {
	return NewLocalStorage(cfg.DataDir, cfg.ConfigPath)
}

// StorageMiner is a miner config
type StorageMiner struct {
	DataDir    string
	API        API
	Dealmaking DealmakingConfig
	Sealing    SealingConfig
	Storage    sectorstorage.SealerConfig
	Fees       MinerFeeConfig
	Addresses  MinerAddressConfig
	NetParams  NetParamsConfig
	DB         DbConfig
	Node       NodeConfig
	JWT        JWTConfig
	Messager   MessagerConfig

	MarketNode     MarketNodeConfig
	PieceStorage   config.PieceStorage
	RegisterProof  RegisterProofConfig
	RegisterMarket RegisterMarketConfig

	ConfigPath string `toml:"-"`
}

func (cfg StorageMiner) LocalStorage() *LocalStorage {
	return NewLocalStorage(cfg.DataDir, cfg.DataDir)
}

type JWTConfig struct {
	Secret string
}

type NodeConfig struct {
	Url   string
	Token string
}

type MessagerConfig struct {
	Url   string
	Token string
}

type MarketNodeConfig struct {
	Mode  string
	Url   string
	Token string
}

type RegisterProofConfig struct {
	Urls  []string
	Token string
}

type RegisterMarketConfig struct {
	Urls  []string
	Token string
}

func (node *NodeConfig) APIEndpoint() (multiaddr.Multiaddr, error) {
	strma := string(node.Url)
	strma = strings.TrimSpace(strma)

	apima, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, err
	}
	return apima, nil
}

func (node *NodeConfig) ListenAddress() (string, error) {
	maAddr, err := multiaddr.NewMultiaddr(node.Url)
	if err != nil {
		return "", err
	}
	ip, err := maAddr.ValueForProtocol(multiaddr.P_IP4)
	if err != nil {
		return "", err
	}
	port, err := maAddr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil {
		return "", err
	}
	return ip + ":" + port, nil
}

func (node *NodeConfig) AuthHeader() http.Header {
	if len(node.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(node.Token))
		return headers
	}
	return nil
}

type DbConfig struct {
	Type   string
	MySql  MySqlConfig
	Sqlite SqliteConfig
}

type SqliteConfig struct {
	Path string
}

type MySqlConfig struct {
	Addr            string        `toml:"addr"`
	User            string        `toml:"user"`
	Pass            string        `toml:"pass"`
	Name            string        `toml:"name"`
	MaxOpenConn     int           `toml:"maxOpenConn"`
	MaxIdleConn     int           `toml:"maxIdleConn"`
	ConnMaxLifeTime time.Duration `toml:"connMaxLifeTime"`
}

type DealmakingConfig struct {
	// When enabled, the miner can accept online deals
	ConsiderOnlineStorageDeals bool
	// When enabled, the miner can accept offline deals
	ConsiderOfflineStorageDeals bool
	// When enabled, the miner can accept retrieval deals
	ConsiderOnlineRetrievalDeals bool
	// When enabled, the miner can accept offline retrieval deals
	ConsiderOfflineRetrievalDeals bool
	// When enabled, the miner can accept verified deals
	ConsiderVerifiedStorageDeals bool
	// When enabled, the miner can accept unverified deals
	ConsiderUnverifiedStorageDeals bool
	// A list of Data CIDs to reject when making deals
	PieceCidBlocklist []cid.Cid
	// Maximum expected amount of time getting the deal into a sealed sector will take
	// This includes the time the deal will need to get transferred and published
	// before being assigned to a sector
	ExpectedSealDuration Duration
	// Maximum amount of time proposed deal StartEpoch can be in future
	MaxDealStartDelay Duration
	// When a deal is ready to publish, the amount of time to wait for more
	// deals to be ready to publish before publishing them all as a batch
	PublishMsgPeriod Duration
	// The maximum number of deals to include in a single PublishStorageDeals
	// message
	MaxDealsPerPublishMsg uint64
	// The maximum collateral that the provider will put up against a deal,
	// as a multiplier of the minimum collateral bound
	MaxProviderCollateralMultiplier uint64
	// The maximum allowed disk usage size in bytes of staging deals not yet
	// passed to the sealing node by the markets service. 0 is unlimited.
	MaxStagingDealsBytes int64
	// The maximum number of parallel online data transfers for storage deals
	SimultaneousTransfersForStorage uint64
	// The maximum number of parallel online data transfers for retrieval deals
	SimultaneousTransfersForRetrieval uint64
	// Minimum start epoch buffer to give time for sealing of sector with deal.
	StartEpochSealingBuffer uint64

	// A command used for fine-grained evaluation of storage deals
	// see https://docs.filecoin.io/mine/lotus/miner-configuration/#using-filters-for-fine-grained-storage-and-retrieval-deal-acceptance for more details
	Filter string
	// A command used for fine-grained evaluation of retrieval deals
	// see https://docs.filecoin.io/mine/lotus/miner-configuration/#using-filters-for-fine-grained-storage-and-retrieval-deal-acceptance for more details
	RetrievalFilter string

	RetrievalPricing *RetrievalPricing
}

type RetrievalPricing struct {
	Strategy string // possible values: "default", "external"

	Default  *RetrievalPricingDefault
	External *RetrievalPricingExternal
}

type RetrievalPricingExternal struct {
	// Path of the external script that will be run to price a retrieval deal.
	// This parameter is ONLY applicable if the retrieval pricing policy strategy has been configured to "external".
	Path string
}

type RetrievalPricingDefault struct {
	// VerifiedDealsFreeTransfer configures zero fees for data transfer for a retrieval deal
	// of a payloadCid that belongs to a verified storage deal.
	// This parameter is ONLY applicable if the retrieval pricing policy strategy has been configured to "default".
	// default value is true
	VerifiedDealsFreeTransfer bool
}

type SealingConfig struct {
	// 0 = no limit
	MaxWaitDealsSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectorsForDeals uint64

	WaitDealsDelay Duration

	// CommittedCapacitySectorLifetime is the default duration a Committed Capacity (CC)
	// sector will live before it must be extended or converted into sector containing deals
	// before it is terminated.
	// Value must be between 180-540 days inclusive.
	CommittedCapacitySectorLifetime Duration

	AlwaysKeepUnsealedCopy bool

	// Run sector finalization before submitting sector proof to the chain
	FinalizeEarly bool

	// enable / disable precommit batching (takes effect after nv13)
	BatchPreCommits bool
	// maximum precommit batch size - batches will be sent immediately above this size
	MaxPreCommitBatch int
	// how long to wait before submitting a batch after crossing the minimum batch size
	PreCommitBatchWait Duration
	// time buffer for forceful batch submission before sectors/deal in batch would start expiring
	PreCommitBatchSlack Duration

	// enable / disable commit aggregation (takes effect after nv13)
	AggregateCommits bool
	// maximum batched commit size - batches will be sent immediately above this size
	MinCommitBatch int
	MaxCommitBatch int
	// how long to wait before submitting a batch after crossing the minimum batch size
	CommitBatchWait Duration
	// time buffer for forceful batch submission before sectors/deals in batch would start expiring
	CommitBatchSlack Duration

	AggregateAboveBaseFee      types.FIL
	BatchPreCommitAboveBaseFee types.FIL

	TerminateBatchMax  uint64
	TerminateBatchMin  uint64
	TerminateBatchWait Duration

	// Whether to use available miner balance for sector collateral instead of sending it with each message
	CollateralFromMinerBalance bool
	// Minimum available balance to keep in the miner actor before sending it with messages
	AvailableBalanceBuffer types.FIL
	// Don't send collateral with messages even if there is no available balance in the miner actor
	DisableCollateralFallback bool
	// Keep this many sectors in sealing pipeline, start CC if needed
	// todo TargetSealingSectors uint64

	// todo TargetSectors - stop auto-pleding new sectors after this many sectors are sealed, default CC upgrade for deals sectors if above
}

type BatchFeeConfig struct {
	Base      types.FIL
	PerSector types.FIL
}

func (b *BatchFeeConfig) FeeForSectors(nSectors int) abi.TokenAmount {
	return big.Add(big.Int(b.Base), big.Mul(big.NewInt(int64(nSectors)), big.Int(b.PerSector)))
}

type MinerFeeConfig struct {
	MaxPreCommitGasFee types.FIL
	MaxCommitGasFee    types.FIL

	// maxBatchFee = maxBase + maxPerSector * nSectors
	MaxPreCommitBatchGasFee BatchFeeConfig
	MaxCommitBatchGasFee    BatchFeeConfig

	MaxTerminateGasFee     types.FIL
	MaxWindowPoStGasFee    types.FIL
	MaxPublishDealsFee     types.FIL
	MaxMarketBalanceAddFee types.FIL
}

type MinerAddressConfig struct {
	PreCommitControl []string
	CommitControl    []string
	TerminateControl []string

	// DisableOwnerFallback disables usage of the owner address for messages
	// sent automatically
	DisableOwnerFallback bool
	// DisableWorkerFallback disables usage of the worker address for messages
	// sent automatically, if control addresses are configured.
	// A control address that doesn't have enough funds will still be chosen
	// over the worker address if this flag is set.
	DisableWorkerFallback bool
}

// API contains configs for API endpoint
type API struct {
	ListenAddress       string
	RemoteListenAddress string
	Timeout             Duration
}

func (api *API) APIEndpoint() (multiaddr.Multiaddr, error) {
	strma := api.ListenAddress
	strma = strings.TrimSpace(strma)

	apima, err := multiaddr.NewMultiaddr(strma)
	if err != nil {
		return nil, err
	}
	return apima, nil
}

type FeeConfig struct {
	DefaultMaxFee types.FIL
}

type NetParamsConfig struct {
	UpgradeIgnitionHeight   abi.ChainEpoch
	ForkLengthThreshold     abi.ChainEpoch
	BlockDelaySecs          uint64
	PreCommitChallengeDelay abi.ChainEpoch
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
