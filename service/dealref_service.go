package service

import (
	"github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/venus-sealer/models/repo"
	"github.com/filecoin-project/venus-sealer/types"
)

var _ types.DealRef = (*DealRefService)(nil)

type DealRefService struct {
	repo.DealRefRepo
}

func NewDealRefServiceService(repo repo.Repo) *DealRefService {
	return &DealRefService{DealRefRepo: repo.DealRefRepo()}
}

func (d *DealRefService) Get(dealId uint64) (types.SealedRefs, error) {
	return d.DealRefRepo.Get(dealId)
}

func (d *DealRefService) Save(dealId uint64, ref types.SealedRef, dealProposal *market.DealProposal) error {
	return d.DealRefRepo.Save(dealId, ref, dealProposal)
}

func (d *DealRefService) Has(dealId uint64) (bool, error) {
	return d.DealRefRepo.Has(dealId)
}

func (d *DealRefService) List() (map[uint64][]types.SealedRef, error) {
	return d.DealRefRepo.List()
}
