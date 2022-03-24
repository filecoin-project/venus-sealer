package sectorstorage

import (
	"context"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type allocSelector struct {
	index stores.SectorIndex
	alloc storiface.SectorFileType
	ptype storiface.PathType
}

func newAllocSelector(index stores.SectorIndex, alloc storiface.SectorFileType, ptype storiface.PathType) *allocSelector {
	return &allocSelector{
		index: index,
		alloc: alloc,
		ptype: ptype,
	}
}

func (s *allocSelector) Ok(ctx context.Context, task types.TaskType, spt abi.RegisteredSealProof, sector storage.SectorRef, whnd *workerHandle) (bool, error) {
	tasks, err := whnd.workerRpc.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	if _, supported := tasks[task]; !supported {
		return false, nil
	}

	// Check the number of tasks
	taskNum, err := whnd.workerRpc.TaskNumbers(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task number: %w", err)
	}

	log.Debugf("tasks allocate for %s: cur %v, total: %v", whnd.info.Hostname, taskNum.Current, taskNum.Total)
	if taskNum.Total > 0 && taskNum.Current >= taskNum.Total {
		log.Warnf("The number of tasks for %s is full, cur %v, total: %v", whnd.info.Hostname, taskNum.Current, taskNum.Total)
		return false, nil
	}

	paths, err := whnd.workerRpc.Paths(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting worker paths: %w", err)
	}

	have := map[stores.ID]struct{}{}
	for _, path := range paths {
		have[path.ID] = struct{}{}
	}

	ssize, err := spt.SectorSize()
	if err != nil {
		return false, xerrors.Errorf("getting sector size: %w", err)
	}

	best, err := s.index.StorageBestAlloc(ctx, s.alloc, ssize, s.ptype)
	if err != nil {
		return false, xerrors.Errorf("finding best alloc storage: %w", err)
	}

	for _, info := range best {
		if _, ok := have[info.ID]; ok {
			return true, nil
		}
	}

	return false, nil
}

func (s *allocSelector) Cmp(ctx context.Context, task types.TaskType, a, b *workerHandle) (bool, error) {
	return a.utilization() < b.utilization(), nil
}

var _ WorkerSelector = &allocSelector{}
