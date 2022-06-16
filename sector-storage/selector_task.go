package sectorstorage

import (
	"context"
	"strconv"
	"strings"

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
	tasks, err := whnd.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	_, supported := tasks[task]

	// Check the number of tasks
	taskNum, err := whnd.workerRpc.TaskNumbers(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task number: %w", err)
	}

	log.Debugf("tasks allocate: %s for %s", taskNum, whnd.info.Hostname)
	nums := strings.Split(taskNum, "-")
	if len(nums) == 2 {
		curNum, _ := strconv.ParseInt(nums[0], 10, 64)
		total, _ := strconv.ParseInt(nums[1], 10, 64)
		if total > 0 && curNum >= total {
			return false, nil
		}
	}

	return supported, nil
}

func (s *taskSelector) Cmp(ctx context.Context, _ types.TaskType, a, b *workerHandle) (bool, error) {
	atasks, err := a.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	btasks, err := b.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	if len(atasks) != len(btasks) {
		return len(atasks) < len(btasks), nil // prefer workers which can do less
	}

	return a.utilization() < b.utilization(), nil
}

var _ WorkerSelector = &taskSelector{}
