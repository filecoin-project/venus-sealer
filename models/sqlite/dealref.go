package sqlite

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type dealRef struct {
	Id        string `gorm:"column:id;type:varchar(36);primary_key;" json:"id"` // 主键
	DealId    uint64 `gorm:"column:deal_id;type:unsigned bigint;" json:"deal_id"`
	SectorId  uint64 `gorm:"column:sector_id;type:unsigned bigint;" json:"sector_id"`
	PadOffset uint64 `gorm:"column:offset_pad;type:unsigned bigint;" json:"offset_pad"`
	UnPadSize uint64 `gorm:"column:size_unpad;type:unsigned bigint;" json:"size_unpad"`
}

func (dealRef *dealRef) TableName() string {
	return "deal_refs"
}

var _ repo.DealRefRepo = (*dealRefRepo)(nil)

type dealRefRepo struct {
	*gorm.DB
}

func newDealRefRepo(db *gorm.DB) *dealRefRepo {
	return &dealRefRepo{DB: db}
}

func (d *dealRefRepo) Get(dealId uint64) (types.SealedRefs, error) {
	var dealRefs []*dealRef
	err := d.DB.Find(&dealRefs, "deal_id=?", dealId).Error
	if err != nil {
		return types.SealedRefs{}, err
	}
	refs := types.SealedRefs{}
	refs.Refs = make([]types.SealedRef, len(dealRefs))
	for index, ref := range dealRefs {
		refs.Refs[index] = types.SealedRef{
			SectorID: abi.SectorNumber(ref.SectorId),
			Offset:   abi.PaddedPieceSize(ref.PadOffset),
			Size:     abi.UnpaddedPieceSize(ref.UnPadSize),
		}
	}
	return refs, nil
}

func (d *dealRefRepo) Save(dealId uint64, ref types.SealedRef, dealProposal *market.DealProposal) error {
	return d.DB.Save(&dealRef{
		Id:        uuid.New().String(),
		DealId:    dealId,
		SectorId:  uint64(ref.SectorID),
		PadOffset: uint64(ref.Offset),
		UnPadSize: uint64(ref.Size),
	}).Error
}

func (d *dealRefRepo) Has(dealId uint64) (bool, error) {
	var count int64
	err := d.DB.Table("deal_refs").Where("deal_id=?", dealId).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (d *dealRefRepo) List() (map[uint64][]types.SealedRef, error) {
	var dealRefs []*dealRef
	if err := d.DB.Find(&dealRefs).Error; err != nil {
		return nil, err
	}

	results := make(map[uint64][]types.SealedRef)
	for _, ref := range dealRefs {
		if val, ok := results[ref.DealId]; ok {
			results[ref.DealId] = append(val, types.SealedRef{
				SectorID: abi.SectorNumber(ref.SectorId),
				Offset:   abi.PaddedPieceSize(ref.PadOffset),
				Size:     abi.UnpaddedPieceSize(ref.UnPadSize),
			})
		} else {
			results[ref.DealId] = []types.SealedRef{types.SealedRef{
				SectorID: abi.SectorNumber(ref.SectorId),
				Offset:   abi.PaddedPieceSize(ref.PadOffset),
				Size:     abi.UnpaddedPieceSize(ref.UnPadSize),
			}}
		}
	}
	return results, nil
}
