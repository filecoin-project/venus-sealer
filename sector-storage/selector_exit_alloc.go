package sectorstorage

import (
	"context"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type exitAndAllocSelector struct {
	*allocSelector
	*existingSelector
}

func newExitAndAllocSelector(index stores.SectorIndex, sector abi.SectorID, exit storiface.SectorFileType, alloc storiface.SectorFileType, ptype storiface.PathType, allowFetch bool) *exitAndAllocSelector {
	return &exitAndAllocSelector{
		allocSelector:    newAllocSelector(index, alloc, ptype),
		existingSelector: newExistingSelector(index, sector, exit, allowFetch),
	}
}

func (s *exitAndAllocSelector) Ok(ctx context.Context, task types.TaskType, spt abi.RegisteredSealProof, whnd *workerHandle) (bool, error) {
	ok, err := s.existingSelector.Ok(ctx, task, spt, whnd)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	return s.allocSelector.Ok(ctx, task, spt, whnd)
}

func (s *exitAndAllocSelector) Cmp(ctx context.Context, task types.TaskType, a, b *workerHandle) (bool, error) {
	return a.utilization() < b.utilization(), nil
}

var _ WorkerSelector = &exitAndAllocSelector{}
