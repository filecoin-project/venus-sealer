package dtypes

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/extern/storage-sealing/sealiface"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/ipfs/go-datastore"
	"github.com/multiformats/go-multiaddr"
)

// ShutdownChan is a channel to which you send a value if you intend to shut
// down the daemon (or miner), including the node and RPC server.
type ShutdownChan chan struct{}

type MinerAddress address.Address
type MinerID abi.ActorID
type NetworkName string
type MetadataDS datastore.Batching
type APIAlg jwt.HMACSHA
type APIEndpoint multiaddr.Multiaddr

// SetSealingDelay sets how long a sector waits for more deals before sealing begins.
type SetSealingConfigFunc func(sealiface.Config) error

// GetSealingDelay returns how long a sector waits for more deals before sealing begins.
type GetSealingConfigFunc func() (sealiface.Config, error)
