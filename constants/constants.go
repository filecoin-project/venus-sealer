package constants

import (
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/pkg/specactors/policy"
	"github.com/raulk/clock"
)

const (
	NewestNetworkVersion = network.Version9
	MessageConfidence    = uint64(5)
	BlocksPerEpoch       = uint64(5)
	ForkLengthThreshold2 = policy.ChainFinality
)

var (
	FullAPIVersion  = newVer(1, 0, 0)
	MinerAPIVersion = newVer(1, 0, 1)

	MinerVersion = newVer(0, 0, 1)
)
var Clock = clock.New()
