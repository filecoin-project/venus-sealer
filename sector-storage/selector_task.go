package sectorstorage

import (
	"context"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/types"
)

type taskSelector struct {
	best []stores.StorageInfo //nolint: unused, structcheck
}

func newTaskSelector() *taskSelector {
	return &taskSelector{}
}

func (s *taskSelector) Ok(ctx context.Context, task types.TaskType, spt abi.RegisteredSealProof, whnd *workerHandle) (bool, error) {
	tasks, err := whnd.workerRpc.TaskTypes(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task types: %w", err)
	}
	_, supported := tasks[task]

	// Check the number of tasks
	taskNum , err := whnd.workerRpc.TaskNumbers(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker task number: %w", err)
	}

	info , err := whnd.workerRpc.Info(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting supported worker info: %w", err)
	}

	log.Infof("tasks allocate: %s for %s", taskNum,  info.Hostname)
	nums := strings.Split(taskNum, "-")
	if len(nums) == 2 {
		curNum, _ := strconv.ParseInt(nums[0],10,64)
		total, _ := strconv.ParseInt(nums[1],10,64)
		if total > 0 && curNum >= total {
			return false, nil
		}
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
