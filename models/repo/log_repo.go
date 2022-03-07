package repo

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/types"
)

type LogRepo interface {
	Count(sectorNumber abi.SectorNumber) (int64, error)
	TruncateAppend (log *types.Log) error
	Truncate(sectorNumber abi.SectorNumber) error
	Append(log *types.Log) error
	List(sectorNumber abi.SectorNumber) ([]*types.Log, error)
	DelLogs(sectorNumber uint64) error
	LatestLog(sectorNumber uint64) (*types.Log, error)
}
