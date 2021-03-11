package repo

import (
	"github.com/filecoin-project/venus-sealer/types"
)

type SectorInfoRepo interface {
	GetSectorInfoByID(sectorNumber uint64) (*types.SectorInfo, error)
	HasSectorInfo(sectorNumber uint64) (bool, error)
	Save(sector *types.SectorInfo) error
	GetAllSectorInfos() ([]*types.SectorInfo, error)
	DeleteBySectorId(sectorNumber uint64) error
	UpdateSectorInfoBySectorId(sectorInfo *types.SectorInfo, sectorNumber uint64) error
}
