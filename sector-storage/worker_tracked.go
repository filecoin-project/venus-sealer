package sectorstorage

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-sealer/sector-storage/storiface"
	"github.com/filecoin-project/venus-sealer/types"
)

type trackedWork struct {
	job            storiface.WorkerJob
	worker         storiface.WorkerID
	workerHostname string
}

type workTracker struct {
	lk sync.Mutex

	done     map[types.CallID]struct{}
	running  map[types.CallID]trackedWork
	prepared map[uuid.UUID]trackedWork

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

func (wt *workTracker) track(ctx context.Context, ready chan struct{}, wid storiface.WorkerID, wi storiface.WorkerInfo, sid storage.SectorRef, task types.TaskType, cb func() (types.CallID, error)) (types.CallID, error) {
	tracked := func(rw int, callID types.CallID) trackedWork {
		return trackedWork{
			job: storiface.WorkerJob{
				ID:      callID,
				Sector:  sid.ID,
				Task:    task,
				Start:   time.Now(),
				RunWait: rw,
			},
			worker:         wid,
			workerHostname: wi.Hostname,
		}
	}

	wt.lk.Lock()
	defer wt.lk.Unlock()

	select {
	case <-ready:
	case <-ctx.Done():
		return types.UndefCall, ctx.Err()
	default:
		prepID := uuid.New()

		wt.prepared[prepID] = tracked(storiface.RWPrepared, types.UndefCall)

		wt.lk.Unlock()

		select {
		case <-ready:
		case <-ctx.Done():
			wt.lk.Lock()
			delete(wt.prepared, prepID)
			return types.UndefCall, ctx.Err()
		}

		wt.lk.Lock()
		delete(wt.prepared, prepID)
	}

	wt.lk.Unlock()
	callID, err := cb()
	wt.lk.Lock()
	if err != nil {
		return callID, err
	}

	_, done := wt.done[callID]
	if done {
		delete(wt.done, callID)
		return callID, err
	}

	wt.running[callID] = tracked(storiface.RWRunning, callID)

	return callID, err
}

func (wt *workTracker) worker(wid storiface.WorkerID, wi storiface.WorkerInfo, w Worker) *trackedWorker {
	return &trackedWorker{
		Worker:     w,
		wid:        wid,
		workerInfo: wi,

		execute: make(chan struct{}),

		tracker: wt,
	}
}

func (wt *workTracker) Running() ([]trackedWork, []trackedWork) {
	wt.lk.Lock()
	defer wt.lk.Unlock()

	running := make([]trackedWork, 0, len(wt.running))
	for _, job := range wt.running {
		running = append(running, job)
	}
	prepared := make([]trackedWork, 0, len(wt.prepared))
	for _, job := range wt.prepared {
		prepared = append(prepared, job)
	}

	return running, prepared
}

type trackedWorker struct {
	Worker
	wid        storiface.WorkerID
	workerInfo storiface.WorkerInfo

	execute chan struct{} // channel blocking execution in case we're waiting for resources but the task is ready to execute

	tracker *workTracker
}

func (t *trackedWorker) start() {
	close(t.execute)
}

func (t *trackedWorker) SealPreCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, pieces []abi.PieceInfo) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTPreCommit1, func() (types.CallID, error) { return t.Worker.SealPreCommit1(ctx, sector, ticket, pieces) })
}

func (t *trackedWorker) SealPreCommit2(ctx context.Context, sector storage.SectorRef, pc1o storage.PreCommit1Out) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTPreCommit2, func() (types.CallID, error) { return t.Worker.SealPreCommit2(ctx, sector, pc1o) })
}

func (t *trackedWorker) SealCommit1(ctx context.Context, sector storage.SectorRef, ticket abi.SealRandomness, seed abi.InteractiveSealRandomness, pieces []abi.PieceInfo, cids storage.SectorCids) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTCommit1, func() (types.CallID, error) { return t.Worker.SealCommit1(ctx, sector, ticket, seed, pieces, cids) })
}

func (t *trackedWorker) SealCommit2(ctx context.Context, sector storage.SectorRef, c1o storage.Commit1Out) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTCommit2, func() (types.CallID, error) { return t.Worker.SealCommit2(ctx, sector, c1o) })
}

func (t *trackedWorker) FinalizeSector(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTFinalize, func() (types.CallID, error) { return t.Worker.FinalizeSector(ctx, sector, keepUnsealed) })
}

func (t *trackedWorker) AddPiece(ctx context.Context, sector storage.SectorRef, pieceSizes []abi.UnpaddedPieceSize, newPieceSize abi.UnpaddedPieceSize, pieceData storage.Data) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTAddPiece, func() (types.CallID, error) {
		return t.Worker.AddPiece(ctx, sector, pieceSizes, newPieceSize, pieceData)
	})
}

func (t *trackedWorker) Fetch(ctx context.Context, s storage.SectorRef, ft storiface.SectorFileType, ptype storiface.PathType, am storiface.AcquireMode) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, s, types.TTFetch, func() (types.CallID, error) { return t.Worker.Fetch(ctx, s, ft, ptype, am) })
}

func (t *trackedWorker) UnsealPiece(ctx context.Context, id storage.SectorRef, index storiface.UnpaddedByteIndex, size abi.UnpaddedPieceSize, randomness abi.SealRandomness, cid cid.Cid) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, id, types.TTUnseal, func() (types.CallID, error) { return t.Worker.UnsealPiece(ctx, id, index, size, randomness, cid) })
}

func (t *trackedWorker) ReplicaUpdate(ctx context.Context, sector storage.SectorRef, pieces []abi.PieceInfo) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTReplicaUpdate, func() (types.CallID, error) {
		return t.Worker.ReplicaUpdate(ctx, sector, pieces)
	})
}

func (t *trackedWorker) ProveReplicaUpdate1(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTProveReplicaUpdate1, func() (types.CallID, error) {
		return t.Worker.ProveReplicaUpdate1(ctx, sector, sectorKey, newSealed, newUnsealed)
	})
}

func (t *trackedWorker) ProveReplicaUpdate2(ctx context.Context, sector storage.SectorRef, sectorKey, newSealed, newUnsealed cid.Cid, vanillaProofs storage.ReplicaVanillaProofs) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTProveReplicaUpdate2, func() (types.CallID, error) {
		return t.Worker.ProveReplicaUpdate2(ctx, sector, sectorKey, newSealed, newUnsealed, vanillaProofs)
	})
}

func (t *trackedWorker) FinalizeReplicaUpdate(ctx context.Context, sector storage.SectorRef, keepUnsealed []storage.Range) (types.CallID, error) {
	return t.tracker.track(ctx, t.execute, t.wid, t.workerInfo, sector, types.TTFinalizeReplicaUpdate, func() (types.CallID, error) { return t.Worker.FinalizeReplicaUpdate(ctx, sector, keepUnsealed) })
}

var _ Worker = &trackedWorker{}
