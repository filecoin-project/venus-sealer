package types

import (
	"github.com/filecoin-project/go-state-types/builtin/v8/market"
)

type DealRef interface {
	Get(uint64) (SealedRefs, error)
	Save(uint64, SealedRef, *market.DealProposal) error
	Has(uint64) (bool, error)
	List() (map[uint64][]SealedRef, error)
}
