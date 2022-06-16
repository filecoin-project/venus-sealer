package repo

import (
	"github.com/filecoin-project/go-state-types/builtin/v8/market"
	"github.com/filecoin-project/venus-sealer/types"
)

type DealRefRepo interface {
	Get(dealId uint64) (types.SealedRefs, error)
	Save(dealId uint64, ref types.SealedRef, dealProposal *market.DealProposal) error
	Has(dealId uint64) (bool, error)
	List() (map[uint64][]types.SealedRef, error)
}
