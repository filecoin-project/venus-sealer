package sectorstorage

import (
	"context"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/types"
)

type taskSelector struct {
	best []stores.StorageInfo //nolint: unused, structcheck
}

func newTaskSelector() *taskSelector {
	return &taskSelector{}
}

func (s *taskSelector) Ok(ctx context.Context, task types.TaskType, spt abi.RegisteredSealProof, sector storage.SectorRef, whnd *workerHandle) (bool, error) {
	tasks, err := whnd.workerRpc.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	_, supported := tasks[task]

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

	return supported, nil
}

func (s *taskSelector) Cmp(ctx context.Context, _ types.TaskType, a, b *workerHandle) (bool, error) {
	atasks, err := a.workerRpc.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	btasks, err := b.workerRpc.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	if len(atasks) != len(btasks) {
		return len(atasks) < len(btasks), nil // prefer workers which can do less
	}

	return a.utilization() < b.utilization(), nil
}

var _ WorkerSelector = &taskSelector{}
