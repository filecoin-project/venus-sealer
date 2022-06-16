package constants

import (
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/raulk/clock"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/filecoin-project/go-state-types/network"
)

var InsecurePoStValidation = false

const (
	NewestNetworkVersion = network.Version15
	MessageConfidence    = uint64(2)
)

// /////
// Address

const AddressMainnetEnvVar = "_mainnet_"

// Blocks (e)
var BlocksPerEpoch = uint64(builtin2.ExpectedLeadersPerEpoch)

var (
	FullAPIVersion0 = newVer(1, 5, 0)
	FullAPIVersion1 = newVer(2, 2, 0)

	MinerAPIVersion0  = newVer(1, 5, 0)
	WorkerAPIVersion0 = newVer(1, 6, 0)

	MinerVersion = newVer(1, 3, 0)
)
var Clock = clock.New()

func SetAddressNetwork(n address.Network) {
	address.CurrentNetwork = n
}

func init() {
	if os.Getenv("VENUS_ADDRESS_TYPE") == AddressMainnetEnvVar {
		SetAddressNetwork(address.Mainnet)
	}
}

const BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)

const MinerFDLimit uint64 = 100_000
