package sectorstorage

import (
	"context"
	"github.com/filecoin-project/venus-sealer/types"
	"io"
	"sync"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
)

type trackedWork struct {
	job    storiface.WorkerJob
	worker WorkerID
}

type workTracker struct {
	lk sync.Mutex

	done    map[types.CallID]struct{}
	running map[types.CallID]trackedWork

	// TODO: done, aggregate stats, queue stats, scheduler feedback
}

func (wt *workTracker) onDone(callID types.CallID) {
	wt.lk.Lock()
	defer wt.lk.Unlock()

	_, ok := wt.running[callID]
	if !ok {
		wt.done[callID] = struct{}{}
		return
	}

	delete(wt.running, callID)
}

func (wt *workTracker) track(wid WorkerID, sid storage.SectorRef, task types.TaskType) func(types.CallID, error) (types.CallID, error) {
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
			worker: wid,
		}

		return callID, err
	}
}

func (wt *workTracker) worker(wid WorkerID, w Worker) Worker {
	return &trackedWorker{
		Worker: w,
		wid:    wid,

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
	wid WorkerID

	tracker *workTracker
}

func (t *trackedWorker) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTPreCommit1)(t.Worker.SealPreCommit1(ctx, sector, ticket, pieces))
}

func (t *trackedWorker) SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTPreCommit2)(t.Worker.SealPreCommit2(ctx, sector, pc1o))
}

func (t *trackedWorker) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTCommit1)(t.Worker.SealCommit1(ctx, sector, ticket, seed, pieces, cids))
}

func (t *trackedWorker) SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTCommit2)(t.Worker.SealCommit2(ctx, sector, c1o))
}

func (t *trackedWorker) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTFinalize)(t.Worker.FinalizeSector(ctx, sector, keepUnsealed))
}

func (t *trackedWorker) AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	return t.tracker.track(t.wid, sector, types.TTAddPiece)(t.Worker.AddPiece(ctx, sector, pieceSizes, newPieceSize, pieceData))
}

func (t *trackedWorker) Fetch(ctx context.Context, s storage.SectorRef, ft storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (types.CallID, error) {
	return t.tracker.track(t.wid, s, types.TTFetch)(t.Worker.Fetch(ctx, s, ft, ptype, am))
}

func (t *trackedWorker) UnsealPiece(ctx context.Context, id storage.SectorRef, index storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, randomness abi.SealRandomness, cid cid.Cid) (types.CallID, error) {
	return t.tracker.track(t.wid, id, types.TTUnseal)(t.Worker.UnsealPiece(ctx, id, index, size, randomness, cid))
}

func (t *trackedWorker) ReadPiece(ctx context.Context, writer io.Writer, id storage.SectorRef, index storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize) (types.CallID, error) {
	return t.tracker.track(t.wid, id, types.TTReadUnsealed)(t.Worker.ReadPiece(ctx, writer, id, index, size))
}

var _ Worker = &trackedWorker{}
