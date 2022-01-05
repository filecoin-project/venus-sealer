package sectorstorage

import (
	"context"
	"strconv"
	"strings"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/stores"
	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type moveSelector struct {
	index      stores.SectorIndex
	sector     abi.SectorID
	alloc      storiface.SectorFileType
	allowFetch bool
	ptype      storiface.PathType
	spt        abi.RegisteredSealProof
}

func newMoveSelector(index stores.SectorIndex, sector abi.SectorID, spt abi.RegisteredSealProof, alloc storiface.SectorFileType, ptype storiface.PathType, allowFetch bool) *moveSelector {
	return &moveSelector{
		index:      index,
		sector:     sector,
		alloc:      alloc,
		allowFetch: allowFetch,
		ptype:      ptype,
		spt:        spt,
	}
}

func (s *moveSelector) Ok(ctx context.Context, task types.TaskType, spt abi.RegisteredSealProof, sector storage.SectorRef, whnd *workerHandle) (bool, error) {
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

	log.Debugf("tasks allocate: %s for %s", taskNum, whnd.info.Hostname)
	nums := strings.Split(taskNum, "-")
	if len(nums) == 2 {
		curNum, _ := strconv.ParseInt(nums[0], 10, 64)
		total, _ := strconv.ParseInt(nums[1], 10, 64)
		if total > 0 && curNum >= total {
			return false, nil
		}
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

func (s *moveSelector) Cmp(ctx context.Context, task types.TaskType, a, b *workerHandle) (bool, error) {
	aExist := false
	bExist := false

	ssize, err := s.spt.SectorSize()
	if err != nil {
		return false, xerrors.Errorf("getting sector size: %w", err)
	}

	best, err := s.index.StorageFindSector(ctx, s.sector, s.alloc, ssize, s.allowFetch)
	if err != nil {
		return false, xerrors.Errorf("finding best storage: %w", err)
	}

	// a
	paths, err := a.workerRpc.Paths(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting worker paths: %w", err)
	}

	have := map[stores.ID]struct{}{}
	for _, path := range paths {
		have[path.ID] = struct{}{}
	}

	for _, info := range best {
		if _, ok := have[info.ID]; ok {
			aExist = true
			break
		}
	}

	// b
	paths, err = b.workerRpc.Paths(ctx)
	if err != nil {
		return false, xerrors.Errorf("getting worker paths: %w", err)
	}

	have = map[stores.ID]struct{}{}
	for _, path := range paths {
		have[path.ID] = struct{}{}
	}

	for _, info := range best {
		if _, ok := have[info.ID]; ok {
			bExist = true
			break
		}
	}

	log.Debugf("%s %v, %s %v", a.info.Hostname, aExist, b.info.Hostname, bExist)
	if aExist && !bExist {
		return true, nil
	}

	if !aExist && bExist {
		return false, nil
	}

	return a.utilization() < b.utilization(), nil
}

var _ WorkerSelector = &moveSelector{}
