package constants

import (
	"github.com/raulk/clock"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/go-state-types/network"
)

var InsecurePoStValidation = false

const (
	NewestNetworkVersion = network.Version13
	MessageConfidence    = uint64(5)
)

// Blocks (e)
var BlocksPerEpoch = uint64(builtin2.ExpectedLeadersPerEpoch)

var (
	FullAPIVersion0   = newVer(1, 3, 0)
	FullAPIVersion1   = newVer(2, 1, 0)
	MinerAPIVersion0  = newVer(1, 2, 0)
	WorkerAPIVersion0 = newVer(1, 1, 0)

	MinerVersion = newVer(1, 0, 2)
)
var Clock = clock.New()
