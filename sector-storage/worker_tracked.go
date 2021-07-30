package sectorstorage

import (
	"context"
	"sync"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type trackedWork struct {
	job            storiface.WorkerJob
	worker         WorkerID
	workerHostname string
}

type workTracker struct {
	lk sync.Mutex

	done    map[types.CallID]struct{}
	running map[types.CallID]trackedWork

	// TODO: done, aggregate stats, queue stats, scheduler feedback
}

func (wt *workTracker) onDone(ctx context.Context, callID types.CallID) {
	wt.lk.Lock()
	defer wt.lk.Unlock()

	_, ok := wt.running[callID]
	if !ok {
		wt.done[callID] = struct{}{}
		return
	}

	delete(wt.running, callID)
}

func (wt *workTracker) track(ctx context.Context, wid WorkerID, wi storiface.WorkerInfo, sid storage.SectorRef, task types.TaskType) func(types.CallID, error) (types.CallID, error) {
	return func(callID types.CallID, err error) (types.CallID, error) {
		if err != nil {
			return callID, err
		}

		wt.lk.Lock()
		defer wt.lk.Unlock()

		_, done := wt.done[callID]
		if done {
			delete(wt.done, callID)
			return callID, err
		}

		wt.running[callID] = trackedWork{
			job: storiface.WorkerJob{
				ID:     callID,
				Sector: sid.ID,
				Task:   task,
				Start:  time.Now(),
			},
			worker:         wid,
			workerHostname: wi.Hostname,
		}

		return callID, err
	}
}

func (wt *workTracker) worker(wid WorkerID, wi storiface.WorkerInfo, w Worker) Worker {
	return &trackedWorker{
		Worker:     w,
		wid:        wid,
		workerInfo: wi,

		tracker: wt,
	}
}

func (wt *workTracker) Running() []trackedWork {
	wt.lk.Lock()
	defer wt.lk.Unlock()

	out := make([]trackedWork, 0, len(wt.running))
	for _, job := range wt.running {
		out = append(out, job)
	}

	return out
}

type trackedWorker struct {
	Worker
	wid        WorkerID
	workerInfo storiface.WorkerInfo

	tracker *workTracker
}

func (t *trackedWorker) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTPreCommit1)(t.Worker.SealPreCommit1(ctx, sector, ticket, pieces))
}

func (t *trackedWorker) SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTPreCommit2)(t.Worker.SealPreCommit2(ctx, sector, pc1o))
}

func (t *trackedWorker) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTCommit1)(t.Worker.SealCommit1(ctx, sector, ticket, seed, pieces, cids))
}

func (t *trackedWorker) SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTCommit2)(t.Worker.SealCommit2(ctx, sector, c1o))
}

func (t *trackedWorker) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTFinalize)(t.Worker.FinalizeSector(ctx, sector, keepUnsealed))
}

func (t *trackedWorker) AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, sector, types.TTAddPiece)(t.Worker.AddPiece(ctx, sector, pieceSizes, newPieceSize, pieceData))
}

func (t *trackedWorker) Fetch(ctx context.Context, s storage.SectorRef, ft storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, s, types.TTFetch)(t.Worker.Fetch(ctx, s, ft, ptype, am))
}

func (t *trackedWorker) UnsealPiece(ctx context.Context, id storage.SectorRef, index storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, randomness abi.SealRandomness, cid cid.Cid) (types.CallID, error) {
	return t.tracker.track(ctx, t.wid, t.workerInfo, id, types.TTUnseal)(t.Worker.UnsealPiece(ctx, id, index, size, randomness, cid))
}

var _ Worker = &trackedWorker{}
