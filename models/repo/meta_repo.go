package repo

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

type MetaDataRepo interface {
	SaveMinerAddress(mAddr address.Address) error
	GetMinerAddress() (address.Address, error)
	IncreaseStorageCounter() (abi.SectorNumber, error)
	GetStorageCounter() (abi.SectorNumber, error)
	SetStorageCounter(counter uint64) error
}
