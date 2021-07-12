package config

import (
	"encoding"
	"net/http"
	"strings"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/multiformats/go-multiaddr"

	"github.com/filecoin-project/venus/pkg/types"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
)

type HomeDir string

type StorageWorker struct {
	ConfigPath string `toml:"-"`
	DataDir    string
	Sealer     NodeConfig
	DB         DbConfig
}

func (cfg StorageWorker) LocalStorage() *LocalStorage {
	return NewLocalStorage(cfg.DataDir, cfg.DataDir)
}

// StorageMiner is a miner config
type StorageMiner struct {
	DataDir       string
	API           API
	Sealing       SealingConfig
	Storage       sectorstorage.SealerConfig
	Fees          MinerFeeConfig
	Addresses     MinerAddressConfig
	NetParams     NetParamsConfig
	DB            DbConfig
	Node          NodeConfig
	JWT           JWTConfig
	Messager      MessagerConfig
	RegisterProof RegisterProofConfig

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

type RegisterProofConfig struct {
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
type SealingConfig struct {
	// 0 = no limit
	MaxWaitDealsSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectors uint64

	// includes failed, 0 = no limit
	MaxSealingSectorsForDeals uint64

	WaitDealsDelay Duration

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

	// network BaseFee below which to stop doing commit aggregation, instead
	// submitting proofs to the chain individually
	AggregateAboveBaseFee types.FIL

	TerminateBatchMax  uint64
	TerminateBatchMin  uint64
	TerminateBatchWait Duration

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
	InsecurePoStValidation  bool
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
