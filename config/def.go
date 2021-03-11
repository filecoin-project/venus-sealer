package config

import (
	"encoding"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/multiformats/go-multiaddr"
	"net/http"
	"strings"
	"time"

	sectorstorage "github.com/filecoin-project/venus-sealer/sector-storage"
	"github.com/filecoin-project/venus/pkg/types"
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
	DataDir   string
	API       API
	Sealing   SealingConfig
	Storage   sectorstorage.SealerConfig
	Fees      MinerFeeConfig
	Addresses MinerAddressConfig
	NetParams NetParamsConfig
	DB        DbConfig
	Node      NodeConfig
	JWT       JWTConfig

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
	UpgradeIgnitionHeight  abi.ChainEpoch
	ForkLengthThreshold    abi.ChainEpoch
	InsecurePoStValidation bool
	BlockDelaySecs         uint64
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
